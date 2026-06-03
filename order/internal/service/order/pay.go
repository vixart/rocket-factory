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

	err := s.txManager.Do(ctx, func(txCtx context.Context) error {
		// 1. Читаем заказ в транзакции
		order, err := s.orderRepository.Get(txCtx, orderUUID)
		if err != nil {
			return fmt.Errorf("получить заказ: %w", err)
		}

		// 2. Проверяем статус
		if order.Status != model.OrderStatusPendingPayment {
			return errs.ErrInvalidOrderStatus
		}

		// 3. Вызываем PaymentService (gRPC внутри транзакции — учебный пример)
		transactionUUID, err = s.paymentClient.PayOrder(txCtx, orderUUID, paymentMethod)
		if err != nil {
			return fmt.Errorf("оплатить заказ: %w", err)
		}

		// 4. Обновляем заказ
		order.Status = model.OrderStatusPaid
		order.TransactionUUID = &transactionUUID
		order.PaymentMethod = &paymentMethod

		err = s.orderRepository.Update(txCtx, order)
		if err != nil {
			return fmt.Errorf("обновить заказ: %w", err)
		}

		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}

	return transactionUUID, nil
}
