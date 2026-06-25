package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/IBM/sarama"
	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	orderApiV1 "github.com/vixart/rocket-factory/order/internal/api/order/v1"
	iamClientV1 "github.com/vixart/rocket-factory/order/internal/client/grpc/iam/v1"
	inventoryClientV1 "github.com/vixart/rocket-factory/order/internal/client/grpc/inventory/v1"
	paymentClientV1 "github.com/vixart/rocket-factory/order/internal/client/grpc/payment/v1"
	"github.com/vixart/rocket-factory/order/internal/config"
	orderShipAssembledConsumerService "github.com/vixart/rocket-factory/order/internal/consumer/assembly_consumer"
	"github.com/vixart/rocket-factory/order/internal/interceptor"
	orderPaidProducerService "github.com/vixart/rocket-factory/order/internal/producer/order_producer"
	orderRepository "github.com/vixart/rocket-factory/order/internal/repository/order"
	"github.com/vixart/rocket-factory/order/internal/service/order"
	"github.com/vixart/rocket-factory/platform/pkg/closer"
	wrappedKafkaConsumer "github.com/vixart/rocket-factory/platform/pkg/kafka/consumer"
	wrappedKafkaProducer "github.com/vixart/rocket-factory/platform/pkg/kafka/producer"
	kafkaMiddleware "github.com/vixart/rocket-factory/platform/pkg/middleware/kafka"
	orderv1 "github.com/vixart/rocket-factory/shared/pkg/openapi/order/v1"
	authv1 "github.com/vixart/rocket-factory/shared/pkg/proto/auth/v1"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
	paymentv1 "github.com/vixart/rocket-factory/shared/pkg/proto/payment/v1"
)

// ConsumerService определяет контракт для запуска Kafka-потребителей.
type ConsumerService interface {
	RunConsumer(ctx context.Context) error
}

type diContainer struct {
	// Инфраструктура
	pgPool        *pgxpool.Pool
	txManager     orderRepository.TxManager
	syncProducer  sarama.SyncProducer
	consumerGroup sarama.ConsumerGroup

	orderPaidProducer     *wrappedKafkaProducer.Producer
	shipAssembledConsumer *wrappedKafkaConsumer.Consumer

	// Клиенты
	iam       order.IAMClient
	inventory order.InventoryClient
	payment   order.PaymentClient

	// Репозитории
	orderRepo order.Repository

	// Сервисы
	orderSvc                 orderApiV1.OrderService
	shipAssembledConsumerSvc ConsumerService
	orderPaidProducerSvc     order.OrderPaidProducer

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
		inventoryConn, err := newGRPCConnection(
			config.AppConfig().Client.InventoryAddress,
			"InventoryService",
			grpc.WithUnaryInterceptor(interceptor.SessionForwarder()),
		)
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
		paymentConn, err := newGRPCConnection(
			config.AppConfig().Client.PaymentAddress,
			"PaymentService",
			grpc.WithUnaryInterceptor(interceptor.SessionForwarder()),
		)
		if err != nil {
			slog.Error("не удалось создать клиент PaymentService", "error", err)
			os.Exit(1)
		}
		closer.Add("PaymentService grpc client", func(_ context.Context) error {
			return paymentConn.Close()
		})

		paymentServiceClient := paymentv1.NewPaymentServiceClient(paymentConn)
		d.payment = paymentClientV1.NewClient(paymentServiceClient)
	}

	return d.payment
}

func (d *diContainer) IAMClient() order.IAMClient {
	if d.iam == nil {
		iamConn, err := newGRPCConnection(config.AppConfig().Client.IAMAddress, "IAMService")
		if err != nil {
			slog.Error("не удалось создать клиент IAMService", "error", err)
			os.Exit(1)
		}
		closer.Add("IAMService grpc client", func(_ context.Context) error {
			return iamConn.Close()
		})

		iamServiceClient := authv1.NewAuthServiceClient(iamConn)
		d.iam = iamClientV1.NewClient(iamServiceClient)
	}

	return d.iam
}

func (d *diContainer) OrderService(ctx context.Context) orderApiV1.OrderService {
	if d.orderSvc == nil {
		d.orderSvc = order.NewService(
			d.OrderRepo(ctx),
			d.OrderPaidProducerSvc(),
			d.InventoryClient(),
			d.PaymentClient(),
			d.TxManager(ctx),
		)
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

// SyncProducer возвращает синхронный Kafka-продюсер.
// При первом вызове создаёт продюсер и регистрирует closer.
func (d *diContainer) SyncProducer() sarama.SyncProducer {
	if d.syncProducer == nil {
		p, err := sarama.NewSyncProducer(
			config.AppConfig().Kafka.Brokers,
			config.AppConfig().OrderPaidProducer.SaramaConfig(),
		)
		if err != nil {
			slog.Error("не удалось создать sync producer", "error", err)
			os.Exit(1)
		}

		closer.Add("Kafka sync producer", func(_ context.Context) error {
			return p.Close()
		})

		d.syncProducer = p
	}

	return d.syncProducer
}

// ConsumerGroup возвращает Kafka consumer group.
// При первом вызове создаёт группу и регистрирует closer.
func (d *diContainer) ConsumerGroup() sarama.ConsumerGroup {
	if d.consumerGroup == nil {
		consumerGroup, err := sarama.NewConsumerGroup(
			config.AppConfig().Kafka.Brokers,
			config.AppConfig().ShipAssembledConsumer.GroupID(),
			config.AppConfig().ShipAssembledConsumer.SaramaConfig(),
		)
		if err != nil {
			slog.Error("не удалось создать consumer group", "error", err)
			os.Exit(1)
		}

		closer.Add("Kafka consumer group", func(_ context.Context) error {
			return consumerGroup.Close()
		})

		d.consumerGroup = consumerGroup
	}

	return d.consumerGroup
}

// OrderPaidProducer возвращает обёртку Kafka-продюсера для событий OrderPaid.
func (d *diContainer) OrderPaidProducer() *wrappedKafkaProducer.Producer {
	if d.orderPaidProducer == nil {
		d.orderPaidProducer = wrappedKafkaProducer.NewProducer(
			d.SyncProducer(),
			config.AppConfig().OrderPaidProducer.Topic(),
		)
	}

	return d.orderPaidProducer
}

// ShipAssembledConsumer возвращает обёртку Kafka-потребителя для событий ShipAssembled.
func (d *diContainer) ShipAssembledConsumer() *wrappedKafkaConsumer.Consumer {
	if d.shipAssembledConsumer == nil {
		d.shipAssembledConsumer = wrappedKafkaConsumer.NewConsumer(
			d.ConsumerGroup(),
			[]string{
				config.AppConfig().ShipAssembledConsumer.Topic(),
			},
			wrappedKafkaConsumer.WithMiddlewares(
				kafkaMiddleware.ConsumerSession(),
				kafkaMiddleware.ConsumerLogging(),
			),
		)
	}

	return d.shipAssembledConsumer
}

func (d *diContainer) OrderPaidProducerSvc() order.OrderPaidProducer {
	if d.orderPaidProducerSvc == nil {
		d.orderPaidProducerSvc = orderPaidProducerService.New(d.OrderPaidProducer())
	}
	return d.orderPaidProducerSvc
}

func (d *diContainer) ShipAssembledConsumerSvc(ctx context.Context) ConsumerService {
	if d.shipAssembledConsumerSvc == nil {
		d.shipAssembledConsumerSvc = orderShipAssembledConsumerService.NewService(
			d.ShipAssembledConsumer(),
			d.OrderRepo(ctx),
			d.InventoryClient(),
			d.TxManager(ctx),
		)
	}
	return d.shipAssembledConsumerSvc
}

func newGRPCConnection(address, serviceName string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	defaultOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(
			keepalive.ClientParameters{
				Time:                10 * time.Second,
				Timeout:             3 * time.Second,
				PermitWithoutStream: true,
			},
		),
	}

	conn, err := grpc.NewClient(
		address,
		append(defaultOpts, opts...)...,
	)
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к %s: %w", serviceName, err)
	}

	return conn, nil
}
