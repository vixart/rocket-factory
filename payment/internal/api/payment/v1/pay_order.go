package v1

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vixart/rocket-factory/payment/internal/api/payment/converter"
	paymentv1 "github.com/vixart/rocket-factory/shared/pkg/proto/payment/v1"
)

func (a *api) PayOrder(
	ctx context.Context,
	req *paymentv1.PayOrderRequest,
) (*paymentv1.PayOrderResponse, error) {
	if req.GetOrderUuid() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_uuid не может быть пустым")
	}

	parsedUuid, err := uuid.Parse(req.GetOrderUuid())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "неверный формат order_uuid: %s", req.GetOrderUuid())
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
