package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	orderHandler "github.com/vixart/rocket-factory/order/pkg/handler"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
	paymentv1 "github.com/vixart/rocket-factory/shared/pkg/proto/payment/v1"
)

const (
	inventoryServiceAddress = "localhost:50051"
	paymentServiceAddress   = "localhost:50052"

	httpPort          = "8080"
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 10 * time.Second
)

func main() {
	if err := run(); err != nil {
		slog.Error("сервис не запустился", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// gRPC соединение с InventoryService
	inventoryConn, err := grpc.NewClient(
		inventoryServiceAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(
			keepalive.ClientParameters{
				Time:                10 * time.Second,
				Timeout:             3 * time.Second,
				PermitWithoutStream: true,
			},
		),
	)
	if err != nil {
		return fmt.Errorf("не удалось подключиться к InventoryService: %w", err)
	}
	defer func() {
		if cerr := inventoryConn.Close(); cerr != nil {
			slog.Error("ошибка закрытия соединения к InventoryService", "error", cerr)
		}
	}()

	// gRPC соединение с PaymentService
	paymentConn, err := grpc.NewClient(
		paymentServiceAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(
			keepalive.ClientParameters{
				Time:                10 * time.Second,
				Timeout:             3 * time.Second,
				PermitWithoutStream: true,
			},
		),
	)
	if err != nil {
		return fmt.Errorf("не удалось подключиться к PaymentService: %w", err)
	}
	defer func() {
		if cerr := paymentConn.Close(); cerr != nil {
			slog.Error("ошибка закрытия соединения к PaymentService", "error", cerr)
		}
	}()

	// Создаём хранилище и обработчик
	store := orderHandler.NewOrderStore()
	h := orderHandler.NewOrderHandler(
		inventoryv1.NewInventoryServiceClient(inventoryConn),
		paymentv1.NewPaymentServiceClient(paymentConn),
		store,
	)

	// OpenAPI сервер
	orderServer, err := orderHandler.SetupServer(h)
	if err != nil {
		return fmt.Errorf("ошибка создания сервера OpenAPI: %w", err)
	}

	// HTTP сервер
	server := &http.Server{
		Addr:              net.JoinHostPort("localhost", httpPort),
		Handler:           orderServer,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	slog.Info("запуск OrderService", "port", httpPort)

	// канал ошибок сервера
	serverErr := make(chan error, 1)

	go func() {
		slog.Info("🚀 HTTP-сервер запущен", "port", httpPort)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		slog.Info("🛑 получен сигнал", "signal", sig)
	case err := <-serverErr:
		return fmt.Errorf("ошибка HTTP сервера: %w", err)
	}

	slog.Info("🛑 завершение работы сервера...")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("ошибка при остановке сервера: %w", err)
	}

	slog.Info("✅ сервер остановлен")

	return nil
}
