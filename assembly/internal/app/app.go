package app

import (
	"context"
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	"github.com/vixart/rocket-factory/assembly/internal/config"
	"github.com/vixart/rocket-factory/platform/pkg/closer"
	"github.com/vixart/rocket-factory/platform/pkg/logger"
)

const (
	shutdownTimeout = 5 * time.Second
)

type App struct {
	diContainer *diContainer
}

func New(ctx context.Context) *App {
	a := &App{}

	a.initDeps(ctx)

	return a
}

func (a *App) initDeps(ctx context.Context) {
	inits := []func(context.Context){
		a.initDI,
		a.initLogger,
	}

	for _, f := range inits {
		f(ctx)
	}
}

func (a *App) initDI(_ context.Context) {
	a.diContainer = &diContainer{}
}

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

func (a *App) Run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	errCh := make(chan error, 2) //nolint:mnd // two components: the gRPC server and the Kafka consumer

	go func() {
		if err := a.runConsumer(ctx); err != nil {
			errCh <- fmt.Errorf("consumer failed: %w", err)
			return
		}
		errCh <- nil
	}()

	var runErr error
	select {
	case runErr = <-errCh:
		// one of the servers failed on its own
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

// runConsumer starts the OrderPaid Kafka consumer.
func (a *App) runConsumer(ctx context.Context) error {
	slog.Info("🚀 OrderPaid Kafka consumer started")

	return a.diContainer.OrderPaidConsumerService().RunConsumer(ctx)
}
