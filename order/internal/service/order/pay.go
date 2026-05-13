package order

import (
	"context"

	"github.com/google/uuid"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
)

func (s *service) Pay(ctx context.Context, orderUuid uuid.UUID, paymentMethod model.PaymentMethod) (*uuid.UUID, error) {
	if paymentMethod == model.PaymentMethodUnspecified {
		return nil, errs.ErrInvalidPaymentMethod
	}

	order, err := s.orderRepository.Get(ctx, orderUuid)
	if err != nil {
		return nil, err
	}

	if order.Status != model.OrderStatusPendingPayment {
		return nil, errs.ErrInvalidOrderStatus
	}

	txUuid, err := s.paymentClient.PayOrder(ctx, order.OrderUUID, paymentMethod)
	if err != nil {
		return nil, err
	}

	order.PaymentMethod = &paymentMethod
	order.Status = model.OrderStatusPaid
	order.TransactionUUID = txUuid

	if err := s.orderRepository.Update(ctx, *order); err != nil {
		return nil, err
	}

	return order.TransactionUUID, nil
}
