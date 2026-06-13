package assembly_consumer

import (
	"context"

	"github.com/vixart/rocket-factory/platform/pkg/kafka"
)

type Consumer interface {
	Consume(ctx context.Context, handler kafka.MessageHandler) error
}
