package order

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
)

func (s *service) Cancel(ctx context.Context, orderUuid uuid.UUID) error {
	var cancelled model.Order

	err := s.txManager.Do(ctx, func(ctx context.Context) error {
		order, err := s.orderRepository.GetForUpdate(ctx, orderUuid)
		if err != nil {
			return err
		}

		if order.Status != model.OrderStatusPendingPayment {
			slog.WarnContext(ctx, "отмена отклонена: неподходящий статус заказа",
				"order_uuid", orderUuid, "status", order.Status)
			return errs.ErrInvalidOrderStatus
		}

		err = s.inventoryClient.ReleaseParts(ctx, order.ItemsUUIDs())
		if err != nil {
			return err
		}

		slog.DebugContext(ctx, "детали освобождены",
			"order_uuid", orderUuid, "part_uuids", order.ItemsUUIDs())

		order.Status = model.OrderStatusCancelled
		if err = s.orderRepository.Update(ctx, order); err != nil {
			return err
		}

		cancelled = order

		return nil
	})
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "заказ отменён",
		"order_uuid", orderUuid,
		"user_uuid", cancelled.UserUUID,
		"released_parts", len(cancelled.Items),
	)

	return nil
}
