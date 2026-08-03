package payment

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	errs "github.com/vixart/rocket-factory/payment/internal/errors"
	"github.com/vixart/rocket-factory/payment/internal/model"
)

func (s *service) PayOrder(ctx context.Context, orderUUID uuid.UUID, paymentMethod model.PaymentMethod) (*uuid.UUID, error) {
	if paymentMethod == model.PaymentMethodUnspecified {
		slog.WarnContext(ctx, "платёж отклонён: способ оплаты не указан", "order_uuid", orderUUID)
		return nil, errs.ErrPaymentMethodNotSpecified
	}

	transactionUUID := uuid.New()

	slog.InfoContext(ctx, "платёж проведён",
		"order_uuid", orderUUID,
		"transaction_uuid", transactionUUID,
		"payment_method", paymentMethod,
	)

	return &transactionUUID, nil
}
