package app

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/IBM/sarama"

	"github.com/vixart/rocket-factory/assembly/internal/config"
	assemblyConsumer "github.com/vixart/rocket-factory/assembly/internal/consumer/order_paid"
	shipAssembledProducer "github.com/vixart/rocket-factory/assembly/internal/producer/ship_assembled"
	assemblyService "github.com/vixart/rocket-factory/assembly/internal/service/assembly"
	"github.com/vixart/rocket-factory/platform/pkg/closer"
	wrappedKafkaConsumer "github.com/vixart/rocket-factory/platform/pkg/kafka/consumer"
	wrappedKafkaProducer "github.com/vixart/rocket-factory/platform/pkg/kafka/producer"
	kafkaMiddleware "github.com/vixart/rocket-factory/platform/pkg/middleware/kafka"
)

const (
	minBuildTime = 1 * time.Second
	maxBuildTime = 3 * time.Second
)

// ConsumerService определяет контракт для запуска Kafka-потребителей.
type ConsumerService interface {
	RunConsumer(ctx context.Context) error
}

type diContainer struct {
	syncProducer  sarama.SyncProducer
	consumerGroup sarama.ConsumerGroup

	orderPaidConsumer     *wrappedKafkaConsumer.Consumer
	shipAssembledProducer *wrappedKafkaProducer.Producer

	// Сервисы
	orderPaidConsumerSvc     ConsumerService
	shipAssembledProducerSvc assemblyService.ShipAssembledProducer
	shipAssembleSvc          assemblyConsumer.ShipAssembleService
}

// ConsumerGroup возвращает Kafka consumer group.
// При первом вызове создаёт группу и регистрирует closer.
func (d *diContainer) ConsumerGroup() sarama.ConsumerGroup {
	if d.consumerGroup == nil {
		consumerGroup, err := sarama.NewConsumerGroup(
			config.AppConfig().Kafka.Brokers,
			config.AppConfig().OrderPaidConsumer.GroupID(),
			config.AppConfig().OrderPaidConsumer.SaramaConfig(),
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

// SyncProducer возвращает синхронный Kafka-продюсер.
// При первом вызове создаёт продюсер и регистрирует closer.
func (d *diContainer) SyncProducer() sarama.SyncProducer {
	if d.syncProducer == nil {
		p, err := sarama.NewSyncProducer(
			config.AppConfig().Kafka.Brokers,
			config.AppConfig().ShipAssembledProducer.SaramaConfig(),
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

// ShipAssembledProducer возвращает обёртку Kafka-продюсера для событий ShipAssembled.
func (d *diContainer) ShipAssembledProducer() *wrappedKafkaProducer.Producer {
	if d.shipAssembledProducer == nil {
		d.shipAssembledProducer = wrappedKafkaProducer.NewProducer(
			d.SyncProducer(),
			config.AppConfig().ShipAssembledProducer.Topic(),
		)
	}

	return d.shipAssembledProducer
}

// OrderPaidConsumer возвращает обёртку Kafka-потребителя для событий OrderPaid.
func (d *diContainer) OrderPaidConsumer() *wrappedKafkaConsumer.Consumer {
	if d.orderPaidConsumer == nil {
		d.orderPaidConsumer = wrappedKafkaConsumer.NewConsumer(
			d.ConsumerGroup(),
			[]string{
				config.AppConfig().OrderPaidConsumer.Topic(),
			},
			wrappedKafkaConsumer.WithMiddlewares(
				kafkaMiddleware.ConsumerLogging(),
			),
		)
	}

	return d.orderPaidConsumer
}

func (d *diContainer) OrderPaidConsumerService() ConsumerService {
	if d.orderPaidConsumerSvc == nil {
		d.orderPaidConsumerSvc = assemblyConsumer.NewService(d.OrderPaidConsumer(), d.shipAssembleService())
	}

	return d.orderPaidConsumerSvc
}

func (d *diContainer) ShipAssembledProducerService() assemblyService.ShipAssembledProducer {
	if d.shipAssembledProducerSvc == nil {
		d.shipAssembledProducerSvc = shipAssembledProducer.NewService(d.ShipAssembledProducer())
	}

	return d.shipAssembledProducerSvc
}

func (d *diContainer) shipAssembleService() assemblyConsumer.ShipAssembleService {
	if d.shipAssembleSvc == nil {
		d.shipAssembleSvc = assemblyService.NewService(d.ShipAssembledProducerService(), minBuildTime, maxBuildTime)
	}

	return d.shipAssembleSvc
}
