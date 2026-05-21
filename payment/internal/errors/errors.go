package errs

import "errors"

var (
	ErrPaymentMethodNotSpecified = errors.New("не задан способ оплаты")
	ErrInvalidUUID               = errors.New("неверный uuid")
)
