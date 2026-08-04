package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-faster/errors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/vixart/rocket-factory/order/internal/config"
	"github.com/vixart/rocket-factory/order/internal/middleware"
	"github.com/vixart/rocket-factory/platform/pkg/closer"
	"github.com/vixart/rocket-factory/platform/pkg/logger"
	"github.com/vixart/rocket-factory/platform/pkg/metrics"
	"github.com/vixart/rocket-factory/platform/pkg/ratelimit"
	"github.com/vixart/rocket-factory/platform/pkg/tracing"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 10 * time.Second
)

// App is the application root that owns the lifecycle of every component.
type App struct {
	diContainer *diContainer
	httpServer  *http.Server
}

// New creates and initializes the application.
func New(ctx context.Context) *App {
	a := &App{}

	a.initDeps(ctx)

	return a
}

// Run drives the application lifecycle: it starts the HTTP server, handles OS
// signals and performs the graceful shutdown.
//
// The server runs in its own goroutine while the main goroutine waits for either
// SIGINT/SIGTERM or a server failure. closer.CloseAll then runs synchronously, so
// the main goroutine is guaranteed to wait for every close to finish before Run
// returns.
func (a *App) Run() error {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cancel()

	errCh := make(chan error, 1)

	go func() {
		errCh <- a.runHTTPServer()
	}()

	go func() {
		if err := a.runConsumer(ctx); err != nil {
			errCh <- fmt.Errorf("consumer failed: %w", err)
			return
		}
		errCh <- nil
	}()

	var runErr error

	select {
	case err := <-errCh:
		// the server failed on its own
		slog.Info("server stopped", "error", err)
		if !errors.Is(err, http.ErrServerClosed) {
			runErr = err
		}
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		shutdownTimeout,
	)
	defer shutdownCancel()

	if err := closer.CloseAll(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)

		if runErr == nil {
			runErr = err
		}
	}

	return runErr
}

// initDeps initializes every application dependency in order.
func (a *App) initDeps(ctx context.Context) {
	inits := []func(context.Context){
		a.initDI,
		a.initLogger,
		a.initMetrics,
		a.initTracer,
		a.initHTTPServer,
	}

	for _, f := range inits {
		f(ctx)
	}
}

// initDI creates the DI container.
func (a *App) initDI(_ context.Context) {
	a.diContainer = &diContainer{}
}

// initLogger configures the global slog logger with the level from the config.
func (a *App) initLogger(_ context.Context) {
	cfg := logger.Config{
		Level:             config.AppConfig().Logger.Level,
		ServiceName:       config.AppConfig().Env.ServiceName,
		Environment:       config.AppConfig().Env.AppEnv,
		EnableOTLP:        config.AppConfig().Otel.EnableOTLP,
		CollectorEndpoint: config.AppConfig().Otel.Endpoint,
	}
	logger.Init(cfg)
	closer.Add("Logger", func(_ context.Context) error {
		return logger.Close()
	})
}

func (a *App) initTracer(ctx context.Context) {
	cfg := tracing.Config{
		CollectorEndpoint: config.AppConfig().Otel.Endpoint,
		ServiceName:       config.AppConfig().Env.ServiceName,
		Environment:       config.AppConfig().Env.AppEnv,
		ServiceVersion:    config.AppConfig().Env.ServiceVersion,
	}
	shutdown, err := tracing.InitTracer(ctx, cfg)
	if err != nil {
		slog.Error("failed to initialize the tracer", "error", err)
		os.Exit(1)
	}
	closer.Add("Tracer", func(ctx context.Context) error {
		return shutdown(ctx)
	})
}

func (a *App) initMetrics(_ context.Context) {
	// instanceID is unique per container: Docker sets the container hostname to its
	// container ID unless a hostname is given explicitly
	instanceID, err := os.Hostname()
	if err != nil {
		slog.Error("failed to initialize metrics", "error", err)
		os.Exit(1)
	}

	cfg := metrics.Config{
		ServiceName:       config.AppConfig().Env.ServiceName,
		Environment:       config.AppConfig().Env.AppEnv,
		InstanceID:        instanceID,
		CollectorEndpoint: config.AppConfig().Otel.Endpoint,
	}

	metrics.Init(cfg)
	closer.Add("Metrics", func(_ context.Context) error {
		return metrics.Close()
	})
}

func (a *App) initHTTPServer(ctx context.Context) {
	cfg := config.AppConfig()

	authMiddleware := middleware.NewAuthMiddleware(a.diContainer.IAMClient())
	rateLimiterMiddleware := ratelimit.Middleware(
		a.diContainer.RateLimiter(ctx),
		cfg.RateLimit.Rate,
		cfg.RateLimit.Burst,
	)

	router := a.diContainer.OrderV1Server(ctx)

	// Built from the innermost layer outwards.
	// A request travels the chain in the opposite direction:
	// otel -> rate limit -> trace ID -> auth -> router.
	var handler http.Handler = router
	handler = authMiddleware.AuthMiddleware(handler)
	handler = tracing.TraceIDMiddleware(handler)
	handler = rateLimiterMiddleware(handler)
	handler = otelhttp.NewHandler(handler, cfg.Env.ServiceName)

	a.httpServer = &http.Server{
		Addr:              cfg.HTTP.Address(),
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	closer.Add("http server", func(ctx context.Context) error {
		return a.httpServer.Shutdown(ctx)
	})
}

func (a *App) runHTTPServer() error {
	slog.Info(
		"🚀 HTTP server started",
		"port",
		config.AppConfig().HTTP.Port,
	)

	return a.httpServer.ListenAndServe()
}

// runConsumer starts the ShipAssembled Kafka consumer.
func (a *App) runConsumer(ctx context.Context) error {
	slog.Info("🚀 ShipAssembled Kafka consumer started")

	return a.diContainer.ShipAssembledConsumerSvc(ctx).RunConsumer(ctx)
}
