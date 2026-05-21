package converter

import (
	"github.com/vixart/rocket-factory/order/internal/model"
	paymentv1 "github.com/vixart/rocket-factory/shared/pkg/proto/payment/v1"
)

func MapPaymentMethod(paymentMethod model.PaymentMethod) paymentv1.PaymentMethod {
	switch paymentMethod {
	case model.PaymentMethodCard:
		return paymentv1.PaymentMethod_PAYMENT_METHOD_CARD
	case model.PaymentMethodSBP:
		return paymentv1.PaymentMethod_PAYMENT_METHOD_SBP
	case model.PaymentMethodCreditCard:
		return paymentv1.PaymentMethod_PAYMENT_METHOD_CREDIT_CARD
	case model.PaymentMethodInvestorMoney:
		return paymentv1.PaymentMethod_PAYMENT_METHOD_INVESTOR_MONEY
	default:
		return paymentv1.PaymentMethod_PAYMENT_METHOD_UNSPECIFIED
	}
}
