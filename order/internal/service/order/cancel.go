package order

import (
	"context"

	"github.com/google/uuid"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
)

func (s *service) Cancel(ctx context.Context, orderUuid uuid.UUID) error {
	return s.txManager.Do(ctx, func(ctx context.Context) error {
		order, err := s.orderRepository.GetForUpdate(ctx, orderUuid)
		if err != nil {
			return err
		}

		if order.Status != model.OrderStatusPendingPayment {
			return errs.ErrInvalidOrderStatus
		}

		err = s.inventoryClient.ReleaseParts(ctx, order.ItemsUUIDs())
		if err != nil {
			return err
		}

		order.Status = model.OrderStatusCancelled
		if err = s.orderRepository.Update(ctx, order); err != nil {
			return err
		}

		return nil
	})
}
