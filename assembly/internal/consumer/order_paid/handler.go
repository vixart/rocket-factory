package order_paid

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/platform/pkg/kafka"
)

func (s *service) OrderPaidHandler(ctx context.Context, msg kafka.Message) error {
	event, err := decodeOrderPaid(msg.Value)
	if err != nil {
		slog.ErrorContext(ctx, "failed to decode OrderPaid", "error", err)
		return nil
	}

	slog.InfoContext(
		ctx, "handling an OrderPaid message",
		"topic", msg.Topic,
		"partition", msg.Partition,
		"offset", msg.Offset,
		"sighting_uuid", event.UUID,
		"order_uuid", event.OrderUUID,
		"user_uuid", event.UserUUID,
	)

	orderUUID, err := uuid.Parse(event.OrderUUID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to parse OrderUUID", "error", err)
		return nil
	}

	userUUID, err := uuid.Parse(event.UserUUID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to parse UserUUID", "error", err)
		return nil
	}

	err = s.shipAssembleService.Assemble(ctx, orderUUID, userUUID)
	if err != nil {
		return fmt.Errorf("failed to assemble the ship: %w", err)
	}

	return nil
}
