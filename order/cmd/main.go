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

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/vixart/rocket-factory/order/pkg/app"
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
	ctx := context.Background()

	err := godotenv.Load("order.env")
	if err != nil {
		return err
	}

	inventoryConn, err := newGRPCConnection(inventoryServiceAddress, "InventoryService")
	if err != nil {
		return err
	}
	defer closeGRPCConnection(inventoryConn, "InventoryService")

	paymentConn, err := newGRPCConnection(paymentServiceAddress, "PaymentService")
	if err != nil {
		return err
	}
	defer closeGRPCConnection(paymentConn, "PaymentService")

	pool, err := newPostgresPool(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	txManager, err := manager.New(trmpgx.NewDefaultFactory(pool))
	if err != nil {
		return err
	}

	server, err := newHTTPServer(
		pool,
		txManager,
		inventoryConn,
		paymentConn,
	)
	if err != nil {
		return err
	}

	return runHTTPServer(server)
}

func newGRPCConnection(address, serviceName string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(
		address,
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
		return nil, fmt.Errorf("не удалось подключиться к %s: %w", serviceName, err)
	}

	return conn, nil
}

func closeGRPCConnection(conn *grpc.ClientConn, serviceName string) {
	if err := conn.Close(); err != nil {
		slog.Error(
			"ошибка закрытия gRPC соединения",
			"service", serviceName,
			"error", err,
		)
	}
}

func newPostgresPool(ctx context.Context) (*pgxpool.Pool, error) {
	dbURI := os.Getenv("DB_URI")

	pool, err := pgxpool.New(ctx, dbURI)
	if err != nil {
		return nil, err
	}

	if err = pool.Ping(ctx); err != nil {
		pool.Close()

		return nil, err
	}

	slog.Info("подключение к PostgreSQL установлено")

	return pool, nil
}

func newHTTPServer(
	pool *pgxpool.Pool,
	txManager *manager.Manager,
	inventoryConn *grpc.ClientConn,
	paymentConn *grpc.ClientConn,
) (*http.Server, error) {
	orderServer, err := app.NewHTTPHandler(
		pool,
		txManager,
		inventoryv1.NewInventoryServiceClient(inventoryConn),
		paymentv1.NewPaymentServiceClient(paymentConn),
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания сервера OpenAPI: %w", err)
	}

	return &http.Server{
		Addr:              net.JoinHostPort("localhost", httpPort),
		Handler:           orderServer,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}, nil
}

func runHTTPServer(server *http.Server) error {
	slog.Info("запуск OrderService", "port", httpPort)

	serverErr := make(chan error, 1)

	go func() {
		slog.Info("🚀 HTTP-сервер запущен", "port", httpPort)

		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

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
