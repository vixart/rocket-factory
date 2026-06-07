package v1

import (
	"context"

	"github.com/vixart/rocket-factory/order/internal/api/order/converter"
	orderv1 "github.com/vixart/rocket-factory/shared/pkg/openapi/order/v1"
)

func (a *api) PayOrder(ctx context.Context, req *orderv1.PayOrderRequest, params orderv1.PayOrderParams) (orderv1.PayOrderRes, error) {
	paymentMethod := converter.PaymentMethodFromApiToModel(req.GetPaymentMethod())

	txUuid, err := a.orderService.Pay(ctx, params.OrderUUID, paymentMethod)
	if err != nil {
		return nil, err
	}

	return &orderv1.PayOrderResponse{
		TransactionUUID: txUuid,
	}, nil
}
