package order_paid

import (
	"context"

	"github.com/google/uuid"
	"github.com/vixart/rocket-factory/platform/pkg/kafka"
)

type Consumer interface {
	Consume(ctx context.Context, handler kafka.MessageHandler) error
}

type ShipAssembleService interface {
	Assemble(ctx context.Context, orderUUID uuid.UUID, userUUID uuid.UUID) error
}
