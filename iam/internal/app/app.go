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

	"github.com/vixart/rocket-factory/iam/internal/config"
	"github.com/vixart/rocket-factory/iam/internal/interceptor"
	"github.com/vixart/rocket-factory/platform/pkg/closer"
	"github.com/vixart/rocket-factory/platform/pkg/grpc/health"
	"github.com/vixart/rocket-factory/platform/pkg/logger"
	"github.com/vixart/rocket-factory/platform/pkg/tracing"
	authv1 "github.com/vixart/rocket-factory/shared/pkg/proto/auth/v1"
	userv1 "github.com/vixart/rocket-factory/shared/pkg/proto/user/v1"
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

func New(ctx context.Context) *App {
	a := &App{}

	a.initDeps(ctx)

	return a
}

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
		a.initTracer,
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
	closer.Add("Tracer", func(_ context.Context) error {
		return shutdown(ctx)
	})
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
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			interceptor.ErrorInterceptor,
			tracing.TraceIDUnaryServerInterceptor(),
		),
	)

	userApi := a.diContainer.UserV1API(ctx)
	authApi := a.diContainer.AuthV1API(ctx)

	closer.Add("gRPC server", func(_ context.Context) error {
		a.grpcServer.GracefulStop()
		return nil
	})

	reflection.Register(a.grpcServer)

	// Регистрируем health service для проверки работоспособности
	health.RegisterService(a.grpcServer)

	userv1.RegisterUserServiceServer(a.grpcServer, userApi)
	authv1.RegisterAuthServiceServer(a.grpcServer, authApi)
}

// runGRPCServer запускает gRPC-сервер и блокирует до его остановки.
func (a *App) runGRPCServer() error {
	slog.Info("🚀 gRPC-сервер запущен", "address", config.AppConfig().GRPC.Address())

	return a.grpcServer.Serve(a.listener)
}
