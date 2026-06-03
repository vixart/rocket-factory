package v1

import (
	"context"
	"errors"
	"net/http"

	"github.com/vixart/rocket-factory/order/internal/api/order/converter"
	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
	"github.com/vixart/rocket-factory/order/internal/service/input"
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

	orderParts := getOrderItemByType(order.Items)

	return &orderv1.OrderDto{
		OrderUUID:       order.UUID,
		HullUUID:        orderParts.HullUUID,
		EngineUUID:      orderParts.EngineUUID,
		ShieldUUID:      converter.OptNilUUIDFromPtr(orderParts.ShieldUUID),
		WeaponUUID:      converter.OptNilUUIDFromPtr(orderParts.WeaponUUID),
		TotalPrice:      order.TotalPrice(),
		TransactionUUID: converter.OptNilUUIDFromPtr(order.TransactionUUID),
		PaymentMethod:   converter.OptNilPaymentMethodFromPtr(order.PaymentMethod),
		Status:          converter.OrderStatusFromModelToApi(order.Status),
		CreatedAt:       order.CreatedAt,
	}, nil
}

func getOrderItemByType(orderItems []model.OrderItem) input.OrderParts {
	var orderParts input.OrderParts

	for _, item := range orderItems {
		switch item.PartType {
		case model.PartTypeHull:
			orderParts.HullUUID = item.UUID
		case model.PartTypeEngine:
			orderParts.EngineUUID = item.UUID
		case model.PartTypeShield:
			orderParts.ShieldUUID = &item.UUID
		case model.PartTypeWeapon:
			orderParts.WeaponUUID = &item.UUID
		}
	}

	return orderParts
}
