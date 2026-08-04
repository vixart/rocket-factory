package errs

import "errors"

var (
	ErrPaymentMethodNotSpecified = errors.New("payment method is not specified")
	ErrInvalidUUID               = errors.New("invalid uuid")
)
