package v1

import (
	"context"
	"errors"
	"net/http"

	"github.com/vixart/rocket-factory/order/internal/api/order/shared"
	errs "github.com/vixart/rocket-factory/order/internal/errors"
	orderv1 "github.com/vixart/rocket-factory/shared/pkg/openapi/order/v1"
)

func (a *api) PayOrder(ctx context.Context, req *orderv1.PayOrderRequest, params orderv1.PayOrderParams) (orderv1.PayOrderRes, error) {
	paymentMethod := shared.PaymentMethodFromApiToModel(req.GetPaymentMethod())

	txUuid, err := a.orderService.Pay(ctx, params.OrderUUID, paymentMethod)
	if err != nil {
		return mapError(err), nil
	}

	return &orderv1.PayOrderResponse{
		TransactionUUID: *txUuid,
	}, nil
}

func mapError(err error) orderv1.PayOrderRes {
	switch {
	case errors.Is(err, errs.ErrOrderNotFound):
		return &orderv1.PayOrderNotFound{
			Code:    http.StatusNotFound,
			Message: "заказ не найден",
		}

	case errors.Is(err, errs.ErrInvalidPaymentMethod):
		return &orderv1.PayOrderBadRequest{
			Code:    http.StatusBadRequest,
			Message: "передан недопустимый метод оплаты",
		}

	case errors.Is(err, errs.ErrInvalidOrderStatus):
		return &orderv1.PayOrderConflict{
			Code:    http.StatusConflict,
			Message: "заказ имеет недопустимый статус",
		}

	case errors.Is(err, errs.ErrPaymentFailed):
		return &orderv1.PayOrderInternalServerError{
			Code:    http.StatusInternalServerError,
			Message: "нет соединения с платежным сервисом",
		}

	default:
		return &orderv1.PayOrderInternalServerError{
			Code:    http.StatusInternalServerError,
			Message: "что-то пошло не так",
		}
	}
}
