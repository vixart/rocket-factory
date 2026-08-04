package order

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
)

func (s *service) Pay(ctx context.Context, orderUUID uuid.UUID, paymentMethod model.PaymentMethod) (uuid.UUID, error) {
	if paymentMethod == model.PaymentMethodUnspecified {
		return uuid.Nil, errs.ErrInvalidPaymentMethod
	}

	var transactionUUID uuid.UUID
	var order model.Order

	err := s.txManager.Do(ctx, func(ctx context.Context) error {
		var err error
		// 1. Read the order inside the transaction
		order, err = s.orderRepository.GetForUpdate(ctx, orderUUID)
		if err != nil {
			return fmt.Errorf("fetch order: %w", err)
		}

		// 2. Check the status
		if order.Status != model.OrderStatusPendingPayment {
			return errs.ErrInvalidOrderStatus
		}

		// 3. Call PaymentService (a gRPC call inside a transaction: educational example)
		transactionUUID, err = s.paymentClient.PayOrder(ctx, orderUUID, paymentMethod)
		if err != nil {
			return fmt.Errorf("pay for the order: %w", err)
		}

		// 4. Update the order
		order.Status = model.OrderStatusPaid
		order.TransactionUUID = &transactionUUID
		order.PaymentMethod = &paymentMethod

		err = s.orderRepository.Update(ctx, order)
		if err != nil {
			return fmt.Errorf("update order: %w", err)
		}

		event := model.OrderPaidEvent{
			UUID:      order.UUID.String(),
			OrderUUID: order.UUID.String(),
			UserUUID:  order.UserUUID.String(),
		}

		err = s.orderPaidProducer.ProduceOrderPaid(ctx, event)
		if err != nil {
			return fmt.Errorf("send OrderPaid: %w", err)
		}

		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}

	ordersPaidTotal.Add(ctx, 1)
	ordersRevenueTotal.Add(ctx, order.TotalPrice())

	return transactionUUID, nil
}
