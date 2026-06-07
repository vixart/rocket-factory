package order

import (
	"context"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/order/internal/model"
	"github.com/vixart/rocket-factory/order/internal/service/input"
)

type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type Repository interface {
	Get(ctx context.Context, uuid uuid.UUID) (model.Order, error)
	Create(_ context.Context, order model.Order) error
	Update(ctx context.Context, order model.Order) error
}

type InventoryClient interface {
	ListParts(ctx context.Context, uuids []uuid.UUID) ([]model.Part, error)
	ReserveParts(ctx context.Context, uuids []uuid.UUID) error
	ReleaseParts(ctx context.Context, uuids []uuid.UUID) error
	ValidateCompatibility(ctx context.Context, orderParts input.OrderParts) error
}

type PaymentClient interface {
	PayOrder(ctx context.Context, orderUuid uuid.UUID, paymentMethod model.PaymentMethod) (txUuid uuid.UUID, e error)
}
