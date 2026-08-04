package app

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"github.com/vixart/rocket-factory/payment/internal/config"
	"github.com/vixart/rocket-factory/payment/internal/interceptor"
	"github.com/vixart/rocket-factory/platform/pkg/closer"
	"github.com/vixart/rocket-factory/platform/pkg/grpc/health"
	"github.com/vixart/rocket-factory/platform/pkg/logger"
	"github.com/vixart/rocket-factory/platform/pkg/tracing"
	paymentv1 "github.com/vixart/rocket-factory/shared/pkg/proto/payment/v1"
)

const (
	// gRPC keepalive parameters.
	grpcMaxConnectionIdle     = 15 * time.Minute // close idle connections (no active RPCs)
	grpcMaxConnectionAge      = 30 * time.Minute // forced rotation, helps load balancing
	grpcMaxConnectionAgeGrace = 5 * time.Second  // grace period for in-flight RPCs
	grpcKeepaliveTime         = 5 * time.Minute  // ping interval to detect dead connections
	grpcKeepaliveTimeout      = 1 * time.Second  // pong wait timeout
	grpcMinPingInterval       = 5 * time.Second  // minimum client ping interval (must be below the client keepalive.Time)
	shutdownTimeout           = 5 * time.Second
)

// App is the application root that owns the lifecycle of every component.
type App struct {
	diContainer *diContainer
	grpcServer  *grpc.Server
	listener    net.Listener
}

// New creates and initializes the application.
func New(ctx context.Context) *App {
	a := &App{}

	a.initDeps(ctx)

	return a
}

// Run drives the application lifecycle: it starts the gRPC server, handles OS
// signals and performs the graceful shutdown.
//
// The server runs in its own goroutine while the main goroutine waits for either
// SIGINT/SIGTERM or a server failure. closer.CloseAll then runs synchronously, so
// the main goroutine is guaranteed to wait for every close to finish before Run
// returns.
func (a *App) Run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- a.runGRPCServer()
	}()

	var runErr error
	select {
	case runErr = <-errCh:
		// the server failed on its own (bind: address already in use, for example)
	case <-ctx.Done():
		slog.Info("shutdown signal received, starting graceful shutdown")
	}
	cancel() // stop intercepting signals: a second Ctrl+C terminates the process immediately

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if err := closer.CloseAll(shutdownCtx); err != nil {
		slog.Error("shutdown failed", "error", err)
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
		a.initTracer,
		a.initListener,
		a.initGRPCServer,
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

// initListener creates the TCP listener for the gRPC server.
func (a *App) initListener(_ context.Context) {
	listener, err := net.Listen("tcp", config.AppConfig().GRPC.Address()) //nolint:noctx // net.Listen takes no context; the address comes from the config
	if err != nil {
		slog.Error("failed to create the TCP listener", "error", err)
		os.Exit(1)
	}

	a.listener = listener
}

// initGRPCServer creates and configures the gRPC server and registers the handlers.
func (a *App) initGRPCServer(ctx context.Context) {
	a.grpcServer = grpc.NewServer(
		grpc.Creds(insecure.NewCredentials()),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     grpcMaxConnectionIdle,
			MaxConnectionAge:      grpcMaxConnectionAge,
			MaxConnectionAgeGrace: grpcMaxConnectionAgeGrace,
			Time:                  grpcKeepaliveTime,
			Timeout:               grpcKeepaliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             grpcMinPingInterval,
			PermitWithoutStream: true, // clients ping without active RPCs, otherwise the connection is dropped
		}),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			interceptor.ErrorInterceptor,
			tracing.TraceIDUnaryServerInterceptor(),
		),
	)

	// Build the API handler before registering the closer: lazy initialization pulls in
	// the database pool and registers it in the closer.
	// The closer is LIFO, so the pool must be registered before the gRPC server: on
	// shutdown the server stops accepting requests first and the database closes after.
	api := a.diContainer.PaymentV1API(ctx)

	closer.Add("gRPC server", func(_ context.Context) error {
		a.grpcServer.GracefulStop()
		return nil
	})

	reflection.Register(a.grpcServer)

	// Register the health service for liveness/readiness probes
	health.RegisterService(a.grpcServer)

	paymentv1.RegisterPaymentServiceServer(a.grpcServer, api)
}

// runGRPCServer starts the gRPC server and blocks until it stops.
func (a *App) runGRPCServer() error {
	slog.Info("🚀 gRPC server started", "address", config.AppConfig().GRPC.Address())

	return a.grpcServer.Serve(a.listener)
}
