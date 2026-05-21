package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/vixart/rocket-factory/payment/pkg/app"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	// Адрес сервера.
	grpcAddress = "localhost:50052"
)

func main() {
	ctx := context.Background()
	lc := net.ListenConfig{}
	lis, err := lc.Listen(ctx, "tcp", grpcAddress)
	if err != nil {
		slog.Error("не удалось создать listener", "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer(app.Interceptors()...)

	app.RegisterServices(grpcServer)

	// Включаем reflection для postman/grpcurl
	reflection.Register(grpcServer)

	slog.Info("запуск PaymentService", "адрес", grpcAddress)

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
}
