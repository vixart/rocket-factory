package app

import (
	"time"

	"github.com/IBM/sarama"

	"github.com/vixart/rocket-factory/assembly/internal/app"
	assemblyConsumer "github.com/vixart/rocket-factory/assembly/internal/consumer/order_paid"
	shipAssembledProducer "github.com/vixart/rocket-factory/assembly/internal/producer/ship_assembled"
	assemblyService "github.com/vixart/rocket-factory/assembly/internal/service/assembly"
	wrappedKafkaConsumer "github.com/vixart/rocket-factory/platform/pkg/kafka/consumer"
	wrappedKafkaProducer "github.com/vixart/rocket-factory/platform/pkg/kafka/producer"
	kafkaMiddleware "github.com/vixart/rocket-factory/platform/pkg/middleware/kafka"
)

type Config struct {
	OrderPaidTopic     string
	ShipAssembledTopic string
	MinBuildTimeSec    int
	MaxBuildTimeSec    int
}

func New(syncProducer sarama.SyncProducer, cg sarama.ConsumerGroup, config Config) app.ConsumerService {
	minBuildTime := time.Duration(config.MinBuildTimeSec) * time.Second
	maxBuildTime := time.Duration(config.MaxBuildTimeSec) * time.Second
	shipAssembledService := assemblyService.NewService(
		newShipAssembledProducer(syncProducer, config),
		minBuildTime,
		maxBuildTime,
	)
	orderPaidConsumer := newOrderPaidConsumer(cg, config)

	return assemblyConsumer.NewService(orderPaidConsumer, shipAssembledService)
}

func newShipAssembledProducer(syncProducer sarama.SyncProducer, config Config) assemblyService.ShipAssembledProducer {
	wrappedShipAssembledProducer := wrappedKafkaProducer.NewProducer(
		syncProducer,
		config.ShipAssembledTopic,
	)
	return shipAssembledProducer.NewService(wrappedShipAssembledProducer)
}

func newOrderPaidConsumer(cg sarama.ConsumerGroup, config Config) *wrappedKafkaConsumer.Consumer {
	return wrappedKafkaConsumer.NewConsumer(
		cg,
		[]string{
			config.OrderPaidTopic,
		},
		wrappedKafkaConsumer.WithMiddlewares(
			kafkaMiddleware.ConsumerSession(),
			kafkaMiddleware.ConsumerLogging(),
		),
	)
}
