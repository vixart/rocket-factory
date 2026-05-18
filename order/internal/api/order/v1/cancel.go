package v1

import (
	"context"
	"errors"
	"net/http"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
	orderv1 "github.com/vixart/rocket-factory/shared/pkg/openapi/order/v1"
)

func (a *api) CancelOrder(ctx context.Context, params orderv1.CancelOrderParams) (orderv1.CancelOrderRes, error) {
	err := a.orderService.Cancel(ctx, params.OrderUUID)
	if err != nil {
		return mapCancelOrderError(err), nil
	}

	return &orderv1.CancelOrderResponse{}, nil
}

func mapCancelOrderError(err error) orderv1.CancelOrderRes {
	if errors.Is(err, errs.ErrOrderNotFound) {
		return &orderv1.CancelOrderNotFound{
			Code:    http.StatusNotFound,
			Message: "заказ не найден",
		}
	} else if errors.Is(err, errs.ErrInvalidOrderStatus) {
		return &orderv1.CancelOrderConflict{
			Code:    http.StatusConflict,
			Message: "неверный статус заказа",
		}
	}

	return &orderv1.CancelOrderInternalServerError{
		Code:    http.StatusInternalServerError,
		Message: "непоправимая ошибка",
	}
}
