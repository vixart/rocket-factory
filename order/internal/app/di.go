package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	orderApiV1 "github.com/vixart/rocket-factory/order/internal/api/order/v1"
	inventoryClientV1 "github.com/vixart/rocket-factory/order/internal/client/grpc/inventory/v1"
	"github.com/vixart/rocket-factory/order/internal/config"
	orderRepository "github.com/vixart/rocket-factory/order/internal/repository/order"
	"github.com/vixart/rocket-factory/order/internal/service/order"
	"github.com/vixart/rocket-factory/platform/pkg/closer"
	orderv1 "github.com/vixart/rocket-factory/shared/pkg/openapi/order/v1"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
)

type diContainer struct {
	// Инфраструктура
	pgPool    *pgxpool.Pool
	txManager orderRepository.TxManager

	// Клиенты
	inventory order.InventoryClient
	payment   order.PaymentClient

	// Репозитории
	orderRepo order.Repository

	// Сервисы
	orderSvc orderApiV1.OrderService

	// API-обработчики
	orderv1Server *orderv1.Server
}

func (d *diContainer) PGPool(ctx context.Context) *pgxpool.Pool {
	if d.pgPool == nil {
		pool, err := pgxpool.New(ctx, config.AppConfig().PG.DSN())
		if err != nil {
			slog.Error("не удалось подключиться к PostgreSQL", "error", err)
			os.Exit(1)
		}

		err = pool.Ping(ctx)
		if err != nil {
			slog.Error("не удалось выполнить ping PostgreSQL", "error", err)
			os.Exit(1)
		}

		closer.Add("PostgreSQL pool", func(_ context.Context) error {
			pool.Close()
			return nil
		})

		d.pgPool = pool
	}

	return d.pgPool
}

func (d *diContainer) TxManager(ctx context.Context) orderRepository.TxManager {
	if d.txManager == nil {
		txManager, err := manager.New(trmpgx.NewDefaultFactory(d.PGPool(ctx)))
		if err != nil {
			slog.Error("не удалось создать Transaction Manager", "error", err)
			os.Exit(1)
		}
		d.txManager = txManager
	}

	return d.txManager
}

func (d *diContainer) OrderRepo(ctx context.Context) order.Repository {
	if d.orderRepo == nil {
		d.orderRepo = orderRepository.NewRepository(d.PGPool(ctx), d.TxManager(ctx))
	}

	return d.orderRepo
}

func (d *diContainer) InventoryClient() order.InventoryClient {
	if d.inventory == nil {
		inventoryConn, err := newGRPCConnection(config.AppConfig().Client.InventoryAddress, "InventoryService")
		if err != nil {
			slog.Error("не удалось создать клиент InventoryService", "error", err)
			os.Exit(1)
		}

		closer.Add("InventoryService grpc client", func(_ context.Context) error {
			return inventoryConn.Close()
		})

		inventoryServiceClient := inventoryv1.NewInventoryServiceClient(inventoryConn)
		d.inventory = inventoryClientV1.NewClient(inventoryServiceClient)
	}

	return d.inventory
}

func (d *diContainer) PaymentClient() order.PaymentClient {
	if d.payment == nil {
		paymentConn, err := newGRPCConnection(config.AppConfig().Client.PaymentAddress, "PaymentService")
		if err != nil {
			slog.Error("не удалось создать клиент PaymentService", "error", err)
			os.Exit(1)
		}
		closer.Add("PaymentService grpc client", func(_ context.Context) error {
			return paymentConn.Close()
		})

		inventoryServiceClient := inventoryv1.NewInventoryServiceClient(paymentConn)
		d.inventory = inventoryClientV1.NewClient(inventoryServiceClient)
	}

	return d.payment
}

func (d *diContainer) OrderService(ctx context.Context) orderApiV1.OrderService {
	if d.orderSvc == nil {
		d.orderSvc = order.NewService(d.OrderRepo(ctx), d.InventoryClient(), d.PaymentClient(), d.TxManager(ctx))
	}

	return d.orderSvc
}

func (d *diContainer) OrderV1Server(ctx context.Context) *orderv1.Server {
	if d.orderv1Server == nil {
		api := orderApiV1.NewApi(d.OrderService(ctx))
		orderv1Server, err := orderv1.NewServer(api, orderv1.WithErrorHandler(orderApiV1.ErrorHandler))
		if err != nil {
			slog.Error("ошибка создания сервера OpenAPI", "error", err)
			os.Exit(1)
		}
		d.orderv1Server = orderv1Server
	}

	return d.orderv1Server
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
