package v1

import (
	"context"

	converter2 "github.com/vixart/rocket-factory/order/internal/api/order/converter"
	orderv1 "github.com/vixart/rocket-factory/shared/pkg/openapi/order/v1"
)

func (a *api) GetOrder(ctx context.Context, params orderv1.GetOrderParams) (orderv1.GetOrderRes, error) {
	order, err := a.orderService.Get(ctx, params.OrderUUID)

	if err != nil {
		return nil, err
	}

	return &orderv1.OrderDto{
		OrderUUID:       order.OrderUUID,
		HullUUID:        order.HullUUID,
		EngineUUID:      order.EngineUUID,
		ShieldUUID:      converter2.OptNilUUIDFromPtr(order.ShieldUUID),
		WeaponUUID:      converter2.OptNilUUIDFromPtr(order.WeaponUUID),
		TotalPrice:      order.TotalPrice,
		TransactionUUID: converter2.OptNilUUIDFromPtr(order.TransactionUUID),
		PaymentMethod:   converter2.OptNilPaymentMethodFromPtr(order.PaymentMethod),
		Status:          converter2.OrderStatusFromModelToApi(order.Status),
		CreatedAt:       order.CreatedAt,
	}, nil
}
