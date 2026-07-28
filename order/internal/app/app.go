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

// App — корневая структура приложения, управляющая жизненным циклом всех компонентов.
type App struct {
	diContainer *diContainer
	httpServer  *http.Server
}

// New создаёт и инициализирует приложение.
func New(ctx context.Context) *App {
	a := &App{}

	a.initDeps(ctx)

	return a
}

// Run управляет жизненным циклом приложения: запускает gRPC-сервер,
// обрабатывает сигналы ОС и выполняет graceful shutdown.
//
// Сервер запускается в отдельной горутине, а main-горутина синхронно ждёт
// либо сигнал SIGINT/SIGTERM, либо падение сервера. После этого
// closer.CloseAll вызывается синхронно — main-горутина гарантированно
// дожидается завершения всех закрытий перед выходом из Run.
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
			errCh <- fmt.Errorf("потребитель упал: %w", err)
			return
		}
		errCh <- nil
	}()

	var runErr error

	select {
	case err := <-errCh:
		// сервер упал сам
		slog.Info("сервер упал", "error", err)
		if !errors.Is(err, http.ErrServerClosed) {
			runErr = err
		}
	case <-ctx.Done():
		slog.Info("получен сигнал завершения")
	}

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		shutdownTimeout,
	)
	defer shutdownCancel()

	if err := closer.CloseAll(shutdownCtx); err != nil {
		slog.Error("ошибка graceful shutdown", "error", err)

		if runErr == nil {
			runErr = err
		}
	}

	return runErr
}

// initDeps последовательно инициализирует все зависимости приложения.
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

// initDI создаёт DI-контейнер.
func (a *App) initDI(_ context.Context) {
	a.diContainer = &diContainer{}
}

// initLogger настраивает глобальный slog с уровнем из конфига.
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
		slog.Error("не удалось инициализировать tracer", "error", err)
		os.Exit(1)
	}
	closer.Add("Tracer", func(ctx context.Context) error {
		return shutdown(ctx)
	})
}

func (a *App) initMetrics(_ context.Context) {
	// instanceID уникален для каждого контейнера — Docker присваивает
	// контейнеру hostname = его container ID, если hostname не задан явно
	instanceID, err := os.Hostname()
	if err != nil {
		slog.Error("не удалось инициализировать metrics", "error", err)
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

	// Порядок — от ядра к внешнему слою.
	// Запрос проходит цепочку в обратном порядке:
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
		"🚀 HTTP-сервер запущен",
		"port",
		config.AppConfig().HTTP.Port,
	)

	return a.httpServer.ListenAndServe()
}

// runConsumer запускает Kafka-потребитель ShipAssembledConsumer.
func (a *App) runConsumer(ctx context.Context) error {
	slog.Info("🚀 Kafka-потребитель ShipAssembled запущен")

	return a.diContainer.ShipAssembledConsumerSvc(ctx).RunConsumer(ctx)
}
