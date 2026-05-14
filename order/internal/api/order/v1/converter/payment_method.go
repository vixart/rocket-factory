package converter

import (
	"github.com/vixart/rocket-factory/order/internal/model"
	orderv1 "github.com/vixart/rocket-factory/shared/pkg/openapi/order/v1"
)

func PaymentMethodFromModelToApi(method *model.PaymentMethod) *orderv1.PaymentMethod {
	switch *method {
	case model.PaymentMethodCard:
		return new(orderv1.PaymentMethodCARD)
	case model.PaymentMethodSBP:
		return new(orderv1.PaymentMethodSBP)
	case model.PaymentMethodCreditCard:
		return new(orderv1.PaymentMethodCREDITCARD)
	case model.PaymentMethodInvestorMoney:
		return new(orderv1.PaymentMethodINVESTORMONEY)
	default:
		return nil
	}
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
