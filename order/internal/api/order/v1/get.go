package v1

import (
	"context"
	"errors"
	"net/http"

	"github.com/vixart/rocket-factory/order/internal/api/order/v1/converter"
	errs "github.com/vixart/rocket-factory/order/internal/errors"
	orderv1 "github.com/vixart/rocket-factory/shared/pkg/openapi/order/v1"
)

func (a *api) GetOrder(ctx context.Context, params orderv1.GetOrderParams) (orderv1.GetOrderRes, error) {
	order, err := a.orderService.Get(ctx, params.OrderUUID)

	if errors.Is(err, errs.ErrOrderNotFound) {
		return &orderv1.GetOrderNotFound{
			Code:    http.StatusNotFound,
			Message: "заказ не найден",
		}, nil
	} else if err != nil {
		return &orderv1.GetOrderInternalServerError{
			Code:    http.StatusInternalServerError,
			Message: "что-то пошло не так",
		}, nil
	}

	return &orderv1.OrderDto{
		OrderUUID:       order.OrderUUID,
		HullUUID:        order.HullUUID,
		EngineUUID:      order.EngineUUID,
		ShieldUUID:      converter.OptNilUUIDFromPtr(order.ShieldUUID),
		WeaponUUID:      converter.OptNilUUIDFromPtr(order.WeaponUUID),
		TotalPrice:      order.TotalPrice,
		TransactionUUID: converter.OptNilUUIDFromPtr(order.TransactionUUID),
		PaymentMethod:   converter.OptNilPaymentMethodFromPtr(order.PaymentMethod),
		Status:          converter.OrderStatusFromModelToApi(order.Status),
		CreatedAt:       order.CreatedAt,
	}, nil
}
