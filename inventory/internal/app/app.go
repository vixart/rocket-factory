package app

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"github.com/vixart/rocket-factory/inventory/internal/config"
	"github.com/vixart/rocket-factory/inventory/internal/interceptor"
	"github.com/vixart/rocket-factory/platform/pkg/closer"
	"github.com/vixart/rocket-factory/platform/pkg/grpc/health"
	"github.com/vixart/rocket-factory/platform/pkg/logger"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
)

const (
	// gRPC keepalive параметры.
	grpcMaxConnectionIdle     = 15 * time.Minute // Закрыть idle-соединения (нет активных RPC)
	grpcMaxConnectionAge      = 30 * time.Minute // Принудительная ротация для балансировки
	grpcMaxConnectionAgeGrace = 5 * time.Second  // Время на завершение активных RPC
	grpcKeepaliveTime         = 5 * time.Minute  // Интервал ping'ов для обнаружения мёртвых соединений
	grpcKeepaliveTimeout      = 1 * time.Second  // Тайм-аут ожидания pong
	grpcMinPingInterval       = 5 * time.Minute  // Минимальный интервал ping'ов от клиента (защита от DoS)
	shutdownTimeout           = 5 * time.Second
)

// App — корневая структура приложения, управляющая жизненным циклом всех компонентов.
type App struct {
	diContainer *diContainer
	grpcServer  *grpc.Server
	listener    net.Listener
}

// New создаёт и инициализирует приложение.
func New(ctx context.Context) *App {
	a := &App{}

	a.initDeps(ctx)

	return a
}

// Run управляет жизненным циклом приложения: запускает gRPC-сервер,
// обрабатывает сигналы ОС и выполняет graceful shutdown
//
// Сервер запускается в отдельной горутине, а main-горутина синхронно ждёт
// либо сигнал SIGINT/SIGTERM, либо падение сервера. После этого
// closer.CloseAll вызывается синхронно — main-горутина гарантированно
// дожидается завершения всех закрытий перед выходом из Run.
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
		// сервер сам упал (например, bind: address already in use)
	case <-ctx.Done():
		slog.Info("получен сигнал завершения, начинаем graceful shutdown")
	}
	cancel() // снимаем перехват сигналов, повторный Ctrl+C завершит процесс принудительно

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if err := closer.CloseAll(shutdownCtx); err != nil {
		slog.Error("ошибка при завершении работы", "error", err)
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
		a.initListener,
		a.initGRPCServer,
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

// initListener создаёт TCP-листенер для gRPC-сервера.
func (a *App) initListener(_ context.Context) {
	listener, err := net.Listen("tcp", config.AppConfig().GRPC.Address()) //nolint:noctx // net.Listen не требует контекст, адрес из конфига
	if err != nil {
		slog.Error("не удалось создать TCP-листенер", "error", err)
		os.Exit(1)
	}

	a.listener = listener
}

// initGRPCServer создаёт и настраивает gRPC-сервер, регистрирует обработчики.
func (a *App) initGRPCServer(ctx context.Context) {
	authInterceptor := interceptor.NewAuthInterceptor(a.diContainer.IAMClient())
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
			PermitWithoutStream: false,
		}),
		grpc.ChainUnaryInterceptor(
			interceptor.ErrorInterceptor,
			authInterceptor.AuthInterceptor,
		),
	)

	// Получаем API-обработчик до регистрации closer'а: ленивая инициализация
	// зацепит за собой создание пула БД и зарегистрирует его в closer'е.
	// Closer работает по LIFO, поэтому пул должен попасть туда раньше gRPC-сервера —
	// тогда при shutdown сначала остановится приём запросов, а уже потом закроется БД
	api := a.diContainer.InventoryV1API(ctx)

	closer.Add("gRPC server", func(_ context.Context) error {
		a.grpcServer.GracefulStop()
		return nil
	})

	reflection.Register(a.grpcServer)

	// Регистрируем health service для проверки работоспособности
	health.RegisterService(a.grpcServer)

	inventoryv1.RegisterInventoryServiceServer(a.grpcServer, api)
}

// runGRPCServer запускает gRPC-сервер и блокирует до его остановки.
func (a *App) runGRPCServer() error {
	slog.Info("🚀 gRPC-сервер запущен", "address", config.AppConfig().GRPC.Address())

	return a.grpcServer.Serve(a.listener)
}
