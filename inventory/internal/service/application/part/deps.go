package part

import (
	"context"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/inventory/internal/model/entity"
	"github.com/vixart/rocket-factory/inventory/internal/service/input"
)

type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type Repository interface {
	Get(ctx context.Context, uuid uuid.UUID) (entity.Part, error)
	List(_ context.Context, partFilter input.PartFilter) ([]entity.Part, error)
	UpdateReservedBatch(ctx context.Context, parts []entity.Part) error
}

type CompatibilityChecker interface {
	Check(parts entity.ResolvedShipSlots) error
}
