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
		slog.ErrorContext(ctx, "не удалось декодировать ShipAssembled", "error", err)
		return nil
	}

	slog.DebugContext(
		ctx, "получено сообщение ShipAssembled",
		"topic", msg.Topic,
		"partition", msg.Partition,
		"offset", msg.Offset,
		"event_uuid", event.UUID,
		"order_uuid", event.OrderUUID,
		"user_uuid", event.UserUUID,
	)

	orderUUID, err := uuid.Parse(event.OrderUUID)
	if err != nil {
		slog.ErrorContext(ctx, "не удалось распознать OrderUUID",
			"order_uuid", event.OrderUUID, "error", err)
		return nil
	}

	return s.txManager.Do(ctx, func(ctx context.Context) error {
		order, err := s.orderRepository.GetForUpdate(ctx, orderUUID)
		if err != nil {
			return err
		}

		if order.Status == model.OrderStatusAssembled {
			slog.InfoContext(ctx, "сообщение пропущено: заказ уже собран",
				"order_uuid", order.UUID, "status", order.Status)
			return nil
		}

		err = s.inventoryClient.CommitParts(ctx, order.ItemsUUIDs())
		if err != nil {
			return err
		}

		order.Status = model.OrderStatusAssembled

		if err = s.orderRepository.Update(ctx, order); err != nil {
			return err
		}

		slog.InfoContext(ctx, "заказ переведён в статус ASSEMBLED",
			"order_uuid", order.UUID,
			"user_uuid", order.UserUUID,
			"committed_parts", len(order.Items),
		)

		return nil
	})
}
