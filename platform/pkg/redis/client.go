package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	defaultDialTimeout  = 5 * time.Second
	defaultReadTimeout  = 3 * time.Second
	defaultWriteTimeout = 3 * time.Second
)

// NewClient creates a Redis client with the platform settings enforced,
// verifies the connection and returns the ready instance.
//
// Platform guarantees (applied on top of the given opts):
//   - ContextTimeoutEnabled = true (go-redis honours the context deadline)
//   - default timeouts when unset (dial: 5s, read/write: 3s)
//
// The logger is a *slog.Logger from the standard library (log/slog).
// slog is the standard logging facade in Go (since 1.21), the io.Writer of logging:
// switching the backend (zap, logrus, ...) only requires replacing the slog.Handler at
// startup, without touching consumer signatures.
func NewClient(opts *redis.Options, logger *slog.Logger) (*redis.Client, error) {
	applyDefaults(opts)

	rdb := redis.NewClient(opts)

	err := rdb.Ping(context.Background()).Err()
	if err != nil {
		_ = rdb.Close() //nolint:gosec // close the client after a failed Ping, ignore the Close error

		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("connected to Redis", "address", opts.Addr)

	return rdb, nil
}

// applyDefaults enforces the platform settings and fills in default timeouts
// when the caller did not provide them.
func applyDefaults(opts *redis.Options) {
	// Enforced: go-redis honours the context deadline while running commands
	opts.ContextTimeoutEnabled = true

	if opts.DialTimeout == 0 {
		opts.DialTimeout = defaultDialTimeout
	}

	if opts.ReadTimeout == 0 {
		opts.ReadTimeout = defaultReadTimeout
	}

	if opts.WriteTimeout == 0 {
		opts.WriteTimeout = defaultWriteTimeout
	}
}
