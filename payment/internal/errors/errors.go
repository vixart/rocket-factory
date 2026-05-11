package errs

import "errors"

var ErrPaymentMethodNotSpecified = errors.New("не задан способ оплаты")
