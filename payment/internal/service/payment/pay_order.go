package payment

import (
	"context"

	"github.com/google/uuid"

	errs "github.com/vixart/rocket-factory/payment/internal/errors"
	"github.com/vixart/rocket-factory/payment/internal/model"
)

func (s *service) PayOrder(_ context.Context, _ uuid.UUID, paymentMethod model.PaymentMethod) (*uuid.UUID, error) {
	if paymentMethod == model.PaymentMethodUnspecified {
		return nil, errs.ErrPaymentMethodNotSpecified
	}

	return new(uuid.New()), nil
}
