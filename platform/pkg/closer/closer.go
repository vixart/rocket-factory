package closer

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// closeFn is a single close function together with the resource name.
type closeFn struct {
	name string
	fn   func(context.Context) error
}

// closer drives the graceful shutdown of the application.
// Close functions run in reverse order (LIFO): the resource added last is closed
// first, which keeps the dependency order correct.
type closer struct {
	mu    sync.Mutex
	once  sync.Once
	funcs []closeFn
}

// globalCloser is the package-level instance, initialized when the package loads,
// so the closer is ready to use immediately: no manual construction is needed,
// the package-level Add and CloseAll are enough.
var globalCloser = newCloser()

// newCloser creates a new closer instance.
func newCloser() *closer {
	return &closer{}
}

// Add registers a close function in the global closer.
func Add(name string, f func(context.Context) error) {
	globalCloser.Add(name, f)
}

// CloseAll runs every close function of the global closer.
func CloseAll(ctx context.Context) error {
	return globalCloser.CloseAll(ctx)
}

// Add registers a close function under a resource name.
func (c *closer) Add(name string, f func(context.Context) error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.funcs = append(c.funcs, closeFn{name: name, fn: f})
}

// CloseAll runs every registered close function in reverse order (LIFO).
// It is safe to call repeatedly: the work happens only once.
func (c *closer) CloseAll(ctx context.Context) error {
	var result error

	c.once.Do(func() {
		c.mu.Lock()
		funcs := c.funcs
		c.funcs = nil
		c.mu.Unlock()

		if len(funcs) == 0 {
			return
		}

		slog.Info("starting graceful shutdown", "count", len(funcs))

		// Iterate in reverse (LIFO): resources added last are closed first.
		// This matters because dependencies are registered in creation order: database first,
		// then services, then the gRPC server. On shutdown the server must stop first (stop
		// accepting requests), then in-flight business logic finishes, and only then the database closes.
		for i := len(funcs) - 1; i >= 0; i-- {
			f := funcs[i]

			start := time.Now()
			slog.Info("closing resource", "name", f.name)

			if err := f.fn(ctx); err != nil {
				slog.Error("failed to close resource", "name", f.name, "error", err, "duration", time.Since(start))

				if result == nil {
					result = err
				}
			} else {
				slog.Info("resource closed", "name", f.name, "duration", time.Since(start))
			}
		}

		slog.Info("graceful shutdown finished", "count", len(funcs))
	})

	return result
}
