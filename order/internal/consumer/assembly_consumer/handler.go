package assembly_consumer

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/order/internal/model"
	"github.com/vixart/rocket-factory/platform/pkg/kafka"
)

func (s *service) ShipAssembledHandler(ctx context.Context, msg kafka.Message) error {
	event, err := decodeShipAssembled(msg.Value)
	if err != nil {
		slog.ErrorContext(ctx, "failed to decode ShipAssembled", "error", err)
		return nil
	}

	slog.InfoContext(
		ctx, "handling a message",
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

	return s.txManager.Do(ctx, func(ctx context.Context) error {
		order, err := s.orderRepository.GetForUpdate(ctx, orderUUID)
		if err != nil {
			return err
		}

		if order.Status == model.OrderStatusAssembled {
			slog.Info("order is already ASSEMBLED, skipping the message", "order_uuid", order.UUID)
			return nil
		}

		err = s.inventoryClient.CommitParts(ctx, order.ItemsUUIDs())
		if err != nil {
			return err
		}

		order.Status = model.OrderStatusAssembled

		return s.orderRepository.Update(ctx, order)
	})
}
