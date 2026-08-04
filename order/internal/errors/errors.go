package errs

import "github.com/go-faster/errors"

var (
	ErrInvalidUUID          = errors.New("invalid uuid")
	ErrPaymentFailed        = errors.New("failed to pay for the order")
	ErrInvalidPaymentMethod = errors.New("invalid payment method")
	ErrInvalidOrderStatus   = errors.New("order is in an invalid status")
	ErrOrderNotFound        = errors.New("order not found")
	ErrInternalError        = errors.New("internal error")

	ErrPartNotFound      = errors.New("part not found")
	ErrOutOfStock        = errors.New("part is out of stock")
	ErrIncompatibleParts = errors.New("parts are incompatible")
	ErrPartTypeMismatch  = errors.New("invalid part type")

	ErrUnauthorized = errors.New("authentication failed")
)
