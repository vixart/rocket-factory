package model

type PaymentMethod string

const (
	PaymentMethodUnspecified   PaymentMethod = "unspecified"
	PaymentMethodCard          PaymentMethod = "card"
	PaymentMethodSBP           PaymentMethod = "sbp"
	PaymentMethodCreditCard    PaymentMethod = "credit_card"
	PaymentMethodInvestorMoney PaymentMethod = "investor_money"
)
