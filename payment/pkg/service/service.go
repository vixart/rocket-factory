package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	paymentv1 "github.com/vixart/rocket-factory/shared/pkg/proto/payment/v1"
)

// PaymentServer реализует gRPC сервис оплаты
type PaymentServer struct {
	paymentv1.UnimplementedPaymentServiceServer
}

// PayOrder обрабатывает оплату заказа
func (s *PaymentServer) PayOrder(
	ctx context.Context,
	req *paymentv1.PayOrderRequest,
) (*paymentv1.PayOrderResponse, error) {
	if req.GetOrderUuid() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_uuid не может быть пустым")
	}

	_, err := uuid.Parse(req.GetOrderUuid())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "неверный формат order_uuid: %s", req.GetOrderUuid())
	}

	if req.GetPaymentMethod() == paymentv1.PaymentMethod_UNSPECIFIED {
		return nil, status.Errorf(codes.InvalidArgument, "не задан способ оплаты")
	}

	txUuid := uuid.New()

	slog.Info("оплата прошла успешно",
		"order_uuid", req.GetOrderUuid(),
		"transaction_uuid", txUuid,
	)

	return &paymentv1.PayOrderResponse{TransactionUuid: txUuid.String()}, nil
}
