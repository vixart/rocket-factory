package v1

import (
	"context"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/order/internal/model"
	"github.com/vixart/rocket-factory/order/internal/service/input"
)

type OrderService interface {
	Get(ctx context.Context, uuid uuid.UUID) (model.Order, error)
	Create(ctx context.Context, orderParts input.OrderParts) (*model.Order, error)
	Pay(ctx context.Context, orderUuid uuid.UUID, paymentMethod model.PaymentMethod) (uuid.UUID, error)
	Cancel(ctx context.Context, uuid uuid.UUID) error
}
