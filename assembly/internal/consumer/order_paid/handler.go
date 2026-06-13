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
		slog.ErrorContext(ctx, "не удалось декодировать OrderPaid", "error", err)
		return nil
	}

	slog.InfoContext(
		ctx, "обработка сообщения",
		"topic", msg.Topic,
		"partition", msg.Partition,
		"offset", msg.Offset,
		"sighting_uuid", event.UUID,
		"order_uuid", event.OrderUUID,
		"user_uuid", event.UserUUID,
	)

	orderUUID, err := uuid.Parse(event.OrderUUID)
	if err != nil {
		slog.ErrorContext(ctx, "не удалось распознать OrderUUID: %w", "error", err)
		return nil
	}

	userUUID, err := uuid.Parse(event.UserUUID)
	if err != nil {
		slog.ErrorContext(ctx, "не удалось распознать UserUUID: %w", "error", err)
		return nil
	}

	err = s.shipAssembleService.Assemble(ctx, orderUUID, userUUID)
	if err != nil {
		return fmt.Errorf("не удалось собрать корабль: %w", err)
	}

	return nil
}
