//go:build apitest

// Package tests holds the integration API test of the distributed rate limiter.
//
// Against a single live Redis container (testcontainers) it checks that the middleware
// answers 429 once the limit is exceeded, and that it fails open (the request goes
// through) when Redis is unavailable.
package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"

	"github.com/vixart/rocket-factory/platform/pkg/ratelimit"
)

// startRateLimitRedis starts Redis in a testcontainer and returns host:port.
// The container is stopped via t.Cleanup.
func startRateLimitRedis(t *testing.T) string {
	t.Helper()
	ctx := context.Background()

	container, err := tcredis.Run(ctx, "redis:8.6.1-alpine3.23")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = testcontainers.TerminateContainer(container)
	})

	addr, err := container.ConnectionString(ctx)
	require.NoError(t, err)

	// ConnectionString returns redis://host:port — strip the scheme
	const prefix = "redis://"
	addr = strings.TrimPrefix(addr, prefix)

	return addr
}

// okHandler is a trivial stub handler: it always answers 200.
func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestRateLimit_429AfterBurstExceeded(t *testing.T) {
	addr := startRateLimitRedis(t)

	rdb := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { _ = rdb.Close() })

	limiter := redis_rate.NewLimiter(rdb)

	// A small limit, so that 429 shows up reliably within a dozen requests.
	// rate=2 means two requests per second; burst=2 allows at most two at once.
	const rate = 2
	const burst = 2

	ts := httptest.NewServer(ratelimit.Middleware(limiter, rate, burst)(okHandler()))
	t.Cleanup(ts.Close)

	// Fire 10 requests at the same path in a row: the first burst goes through and
	// the rest must get 429.
	const totalRequests = 10
	statuses := make([]int, 0, totalRequests)
	for range totalRequests {
		resp, err := http.Get(ts.URL + "/limited")
		require.NoError(t, err)
		statuses = append(statuses, resp.StatusCode)
		_ = resp.Body.Close()
	}

	var ok, tooMany int
	for _, s := range statuses {
		switch s {
		case http.StatusOK:
			ok++
		case http.StatusTooManyRequests:
			tooMany++
		}
	}
	require.Greater(t, ok, 0, "at least one request had to pass: %v", statuses)
	require.Greater(t, tooMany, 0, "at least one request had to get 429: %v", statuses)
	require.LessOrEqual(t, ok, burst+1,
		"no more than burst may pass (+1 tolerance for the GCRA refill): %v", statuses)
}

func TestRateLimit_FailOpen_OnRedisUnavailable(t *testing.T) {
	// 127.0.0.1:1 is a privileged port nobody listens on → ECONNREFUSED. That is simpler
	// than starting a testcontainer just to kill it, and more stable (port 0 would be
	// remapped to a dynamic one; port 1 is reliably dead).
	rdb := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 200 * time.Millisecond,
	})
	t.Cleanup(func() { _ = rdb.Close() })

	limiter := redis_rate.NewLimiter(rdb)

	ts := httptest.NewServer(ratelimit.Middleware(limiter, 100, 200)(okHandler()))
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/whatever")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Fail-open: the rate limiter could not reach Redis, yet the request must still
	// reach the handler.
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestRateLimit_Distributed_LimitSharedBetweenInstances is the central scenario:
// "make sure the limit is shared, not tripled across three instances".
//
// One Redis, three independent redis_rate.Limiter instances and three httptest servers,
// all working through the same Redis key. In total exactly `burst` requests must pass,
// not `burst × 3` — that is the distributed effect.
func TestRateLimit_Distributed_LimitSharedBetweenInstances(t *testing.T) {
	addr := startRateLimitRedis(t)
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { _ = rdb.Close() })

	const (
		instances = 3
		rate      = 5 // 5 rps
		burst     = 5
	)

	servers := make([]*httptest.Server, instances)
	for i := range instances {
		// EACH instance gets its OWN redis_rate.NewLimiter, but they share the Redis
		// client and the key (the path), so the limit is consumed jointly.
		limiter := redis_rate.NewLimiter(rdb)
		servers[i] = httptest.NewServer(ratelimit.Middleware(limiter, rate, burst)(okHandler()))
	}
	t.Cleanup(func() {
		for _, s := range servers {
			s.Close()
		}
	})

	const totalRequests = 30 // far more than burst*instances
	const path = "/distributed"

	var (
		ok      atomic.Int64
		tooMany atomic.Int64
		others  atomic.Int64
	)

	// Walk the instances in turn: request i goes to server i%3. The point is to see
	// that no more than burst+tolerance pass through all three combined.
	for i := range totalRequests {
		srv := servers[i%instances]
		resp, err := http.Get(srv.URL + path)
		require.NoError(t, err)
		switch resp.StatusCode {
		case http.StatusOK:
			ok.Add(1)
		case http.StatusTooManyRequests:
			tooMany.Add(1)
		default:
			others.Add(1)
		}
		_ = resp.Body.Close()
	}

	assert.Zero(t, others.Load(),
		"only 200/429 were expected, no other codes")
	assert.Positive(t, ok.Load(), "at least one request must pass")
	assert.Positive(t, tooMany.Load(), "at least one request must get 429")

	// The core distributed assertion: no more than burst+1 passed through all three
	// instances rather than burst*3. With a per-instance counter we would see up to
	// 15 successful responses.
	assert.LessOrEqualf(t, ok.Load(), int64(burst+1),
		"distributed: at most burst+tolerance may pass through %d instances combined, "+
			"not burst×instances=%d. Got: ok=%d tooMany=%d",
		instances, burst*instances, ok.Load(), tooMany.Load())
}

// TestRateLimit_PerPath_IndependentLimits: the middleware keys the limit by r.URL.Path.
// Requests to /foo must not affect requests to /bar even when /foo has already
// exhausted its burst.
func TestRateLimit_PerPath_IndependentLimits(t *testing.T) {
	addr := startRateLimitRedis(t)
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { _ = rdb.Close() })

	limiter := redis_rate.NewLimiter(rdb)

	const rate = 1
	const burst = 1

	ts := httptest.NewServer(ratelimit.Middleware(limiter, rate, burst)(okHandler()))
	t.Cleanup(ts.Close)

	// /foo: hammer it until 429.
	for range 5 {
		resp, err := http.Get(ts.URL + "/foo")
		require.NoError(t, err)
		_ = resp.Body.Close()
	}

	// /bar must get 200: it has its own counter.
	respBar, err := http.Get(ts.URL + "/bar")
	require.NoError(t, err)
	defer respBar.Body.Close()

	assert.Equal(t, http.StatusOK, respBar.StatusCode,
		"per-path rate limit: /bar must have a counter independent of /foo")
}

// TestRateLimit_Concurrency_AtomicGCRA: 50 concurrent goroutines on a single path.
// The atomicity of GCRA in redis_rate must guarantee that no more than
// burst+tolerance pass, regardless of scheduler order.
func TestRateLimit_Concurrency_AtomicGCRA(t *testing.T) {
	addr := startRateLimitRedis(t)
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { _ = rdb.Close() })

	limiter := redis_rate.NewLimiter(rdb)

	const rate = 3
	const burst = 3

	ts := httptest.NewServer(ratelimit.Middleware(limiter, rate, burst)(okHandler()))
	t.Cleanup(ts.Close)

	const goroutines = 50
	var (
		wg      sync.WaitGroup
		ok      atomic.Int64
		tooMany atomic.Int64
		others  atomic.Int64
	)

	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			resp, err := http.Get(ts.URL + "/concurrent")
			if err != nil {
				others.Add(1)
				return
			}
			defer func() { _ = resp.Body.Close() }()
			switch resp.StatusCode {
			case http.StatusOK:
				ok.Add(1)
			case http.StatusTooManyRequests:
				tooMany.Add(1)
			default:
				others.Add(1)
			}
		}()
	}
	wg.Wait()

	assert.Zero(t, others.Load(), "only 200/429 were expected")
	assert.Positive(t, ok.Load(), "at least one request must pass")
	assert.Positive(t, tooMany.Load(), "most requests must be rejected with 429")

	// burst+1 tolerates the GCRA refill during the few milliseconds the 50 concurrent
	// requests take. The point is that we did not get 50 successes out of 50.
	assert.LessOrEqualf(t, ok.Load(), int64(burst+1),
		"concurrency: GCRA must stay atomic under load. ok=%d tooMany=%d",
		ok.Load(), tooMany.Load())
}
