package v1

import (
	"context"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/order/internal/service/input"
	orderv1 "github.com/vixart/rocket-factory/shared/pkg/openapi/order/v1"
)

func (a *api) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (orderv1.CreateOrderRes, error) {
	engineUUID := req.GetEngineUUID()
	hullUUID := req.GetHullUUID()

	var shieldUUID, weaponUUID *uuid.UUID

	if v, ok := req.GetShieldUUID().Get(); ok {
		shieldUUID = &v
	}

	if v, ok := req.GetWeaponUUID().Get(); ok {
		weaponUUID = &v
	}

	orderParts := input.OrderParts{
		EngineUUID: engineUUID,
		HullUUID:   hullUUID,
		ShieldUUID: shieldUUID,
		WeaponUUID: weaponUUID,
	}

	order, err := a.orderService.Create(ctx, orderParts)
	if err != nil {
		return nil, err
	}

	return &orderv1.CreateOrderResponse{
		OrderUUID:  order.UUID,
		TotalPrice: order.TotalPrice(),
	}, nil
}
