package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/vixart/rocket-factory/inventory/pkg/app"
)

const (
	// Адрес сервера.
	grpcAddress = "localhost:50051"
)

func main() {
	if err := run(); err != nil {
		slog.Error("сервис не запустился", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()
	lc := net.ListenConfig{}
	lis, err := lc.Listen(ctx, "tcp", grpcAddress)
	if err != nil {
		slog.Error("не удалось создать listener", "error", err)
		return err
	}

	defer func() {
		if err := lis.Close(); err != nil {
			slog.Error("не удалось закрыть listener", "error", err)
		}
	}()

	err = godotenv.Load("inventory.env")
	if err != nil {
		return err
	}

	dbURI := os.Getenv("DB_URI")

	// 1. Создаём пул соединений к PostgreSQL
	pool, err := pgxpool.New(ctx, dbURI)
	if err != nil {
		return err
	}
	defer pool.Close()

	err = pool.Ping(ctx)
	if err != nil {
		return err
	}

	slog.Info("подключение к PostgreSQL установлено")

	grpcServer := grpc.NewServer(app.Interceptors()...)

	app.RegisterServices(grpcServer, pool)

	// Включаем reflection для postman/grpcurl
	reflection.Register(grpcServer)

	slog.Info("запуск InventoryService", "адрес", grpcAddress)

	go func() {
		slog.Info("🚀 gRPC сервер запущен", "address", grpcAddress)
		if serveErr := grpcServer.Serve(lis); serveErr != nil {
			slog.Error("ошибка запуска сервера", "error", serveErr)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("🛑 остановка gRPC сервера")
	grpcServer.GracefulStop()
	slog.Info("✅ сервер остановлен")

	return nil
}
