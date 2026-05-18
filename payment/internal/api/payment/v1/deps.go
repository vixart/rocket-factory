package v1

import (
	"context"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/payment/internal/model"
)

type PaymentService interface {
	PayOrder(_ context.Context, orderUuid uuid.UUID, paymentMethod model.PaymentMethod) (*uuid.UUID, error)
}
