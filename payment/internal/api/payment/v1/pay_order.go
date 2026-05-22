package v1

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/payment/internal/api/converter"
	errs "github.com/vixart/rocket-factory/payment/internal/errors"
	paymentv1 "github.com/vixart/rocket-factory/shared/pkg/proto/payment/v1"
)

func (a *api) PayOrder(
	ctx context.Context,
	req *paymentv1.PayOrderRequest,
) (*paymentv1.PayOrderResponse, error) {
	if req.GetOrderUuid() == "" {
		return nil, fmt.Errorf("order_uuid не может быть пустым, %w", errs.ErrInvalidUUID)
	}

	parsedUuid, err := uuid.Parse(req.GetOrderUuid())
	if err != nil {
		return nil, fmt.Errorf("неверный формат order_uuid: %s, %w", req.GetOrderUuid(), errs.ErrInvalidUUID)
	}

	txUuid, err := a.paymentService.PayOrder(ctx, parsedUuid, converter.PaymentMethodProtoToModel(req.GetPaymentMethod()))
	if err != nil {
		return nil, err
	}

	slog.Info("оплата прошла успешно",
		"order_uuid", req.GetOrderUuid(),
		"transaction_uuid", txUuid,
	)

	return &paymentv1.PayOrderResponse{TransactionUuid: txUuid.String()}, nil
}
