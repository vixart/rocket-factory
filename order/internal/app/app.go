package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-faster/errors"

	"github.com/vixart/rocket-factory/order/internal/config"
	"github.com/vixart/rocket-factory/order/internal/middleware"
	"github.com/vixart/rocket-factory/platform/pkg/closer"
	"github.com/vixart/rocket-factory/platform/pkg/logger"
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
	logger.Init(config.AppConfig().Logger.Level)
}

func (a *App) initHTTPServer(ctx context.Context) {
	authMiddleware := middleware.NewAuthMiddleware(a.diContainer.IAMClient())
	serverHandler := a.diContainer.OrderV1Server(ctx)
	a.httpServer = &http.Server{
		Addr:              config.AppConfig().HTTP.Address(),
		Handler:           authMiddleware.AuthMiddleware(serverHandler),
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
