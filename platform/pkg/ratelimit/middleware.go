package ratelimit

import (
	"log/slog"
	"net/http"

	"github.com/go-redis/redis_rate/v10"
)

// Middleware builds an HTTP middleware with a distributed rate limiter backed by redis_rate.
//
// redis_rate implements GCRA (Generic Cell Rate Algorithm), a token bucket variant
// executed atomically in Redis through a Lua script. Because the counter lives in
// Redis, every instance of the service shares it, so the limit is global rather than
// per instance.
//
// rate  — average number of requests per second allowed.
// burst — maximum burst size (how many requests may be made at once).
func Middleware(limiter *redis_rate.Limiter, rate, burst int) func(http.Handler) http.Handler {
	limit := redis_rate.Limit{
		Rate:   rate,
		Burst:  burst,
		Period: redis_rate.PerSecond(rate).Period,
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// redis_rate prefixes the key with "rate:" on its own,
			// so no extra prefix is needed here.
			key := r.URL.Path

			res, err := limiter.Allow(r.Context(), key, limit)
			if err != nil {
				slog.WarnContext(
					r.Context(), "rate limiter is unavailable, letting the request through",
					"error", err,
					"path", r.URL.Path,
				)
				next.ServeHTTP(w, r)
				return
			}

			if res.Allowed == 0 {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusTooManyRequests)

				if _, wErr := w.Write([]byte("request rate limit exceeded")); wErr != nil {
					slog.WarnContext(r.Context(), "failed to write the 429 response body", "error", wErr)
				}

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
