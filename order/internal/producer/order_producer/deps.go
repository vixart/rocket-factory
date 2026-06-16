package order_producer

import (
	"context"

	"github.com/vixart/rocket-factory/platform/pkg/kafka"
)

type KafkaProducer interface {
	Send(ctx context.Context, msg *kafka.Message) error
}
