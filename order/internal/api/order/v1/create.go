package v1

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
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

	orderParts := model.OrderParts{
		EngineUUID: engineUUID,
		HullUUID:   hullUUID,
		ShieldUUID: shieldUUID,
		WeaponUUID: weaponUUID,
	}

	order, err := a.orderService.Create(ctx, orderParts)
	if err != nil {
		return mapCreateOrderError(err), nil
	}

	return &orderv1.CreateOrderResponse{
		OrderUUID:  order.OrderUUID,
		TotalPrice: order.TotalPrice,
	}, nil
}

func mapCreateOrderError(err error) orderv1.CreateOrderRes {
	switch {
	case errors.Is(err, errs.ErrPartNotFound) || errors.Is(err, errs.ErrInventoryPartNotFound):
		return &orderv1.CreateOrderNotFound{
			Code:    http.StatusNotFound,
			Message: err.Error(),
		}

	case errors.Is(err, errs.ErrInvalidUUID) || errors.Is(err, errs.ErrOrderAlreadyExists):
		return &orderv1.CreateOrderBadRequest{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		}

	case errors.Is(err, errs.ErrPartInsufficientStock):
		return &orderv1.CreateOrderConflict{
			Code:    http.StatusConflict,
			Message: err.Error(),
		}
	}

	return &orderv1.CreateOrderInternalServerError{
		Code:    http.StatusInternalServerError,
		Message: "непоправимая ошибка",
	}
}
