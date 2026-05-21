package converter

import (
	"github.com/vixart/rocket-factory/order/internal/model"
	orderv1 "github.com/vixart/rocket-factory/shared/pkg/openapi/order/v1"
)

func OrderStatusFromModelToApi(status model.OrderStatus) orderv1.OrderStatus {
	switch status {
	case model.OrderStatusPendingPayment:
		return orderv1.OrderStatusPENDINGPAYMENT

	case model.OrderStatusPaid:
		return orderv1.OrderStatusPAID

	case model.OrderStatusCanceled:
		return orderv1.OrderStatusCANCELLED

	default:
		return ""
	}
}
