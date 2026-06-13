package assembly_consumer

import (
	"context"

	"github.com/vixart/rocket-factory/platform/pkg/kafka"
)

type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type Consumer interface {
	Consume(ctx context.Context, handler kafka.MessageHandler) error
}
