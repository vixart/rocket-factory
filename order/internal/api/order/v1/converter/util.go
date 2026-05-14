package converter

import (
	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/order/internal/model"
	orderv1 "github.com/vixart/rocket-factory/shared/pkg/openapi/order/v1"
)

func OptNilUUIDFromPtr(u *uuid.UUID) orderv1.OptNilUUID {
	if u == nil {
		return orderv1.OptNilUUID{}
	}
	return orderv1.NewOptNilUUID(*u)
}

func OptNilPaymentMethodFromPtr(pm *model.PaymentMethod) orderv1.OptNilPaymentMethod {
	apiPm := PaymentMethodFromModelToApi(pm)

	if apiPm == nil {
		return orderv1.OptNilPaymentMethod{}
	}

	return orderv1.NewOptNilPaymentMethod(*apiPm)
}
