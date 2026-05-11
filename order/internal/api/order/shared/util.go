package shared

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
	if pm == nil {
		return orderv1.OptNilPaymentMethod{}
	}
	return orderv1.NewOptNilPaymentMethod(orderv1.PaymentMethod(*pm))
}

func PaymentMethodFromApiToModel(pmApi orderv1.PaymentMethod) model.PaymentMethod {
	switch pmApi {
	case orderv1.PaymentMethodCARD:
		return model.PaymentMethodCard
	case orderv1.PaymentMethodSBP:
		return model.PaymentMethodSBP
	case orderv1.PaymentMethodCREDITCARD:
		return model.PaymentMethodCreditCard
	case orderv1.PaymentMethodINVESTORMONEY:
		return model.PaymentMethodInvestorMoney
	default:
		return model.PaymentMethodUnspecified
	}
}
