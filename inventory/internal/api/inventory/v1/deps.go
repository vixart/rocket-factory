package v1

import (
	"context"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/inventory/internal/model/entity"
	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
)

type InventoryService interface {
	Get(ctx context.Context, uuid uuid.UUID) (entity.Part, error)
	List(ctx context.Context, uuids []uuid.UUID, partType valueobject.PartType) ([]entity.Part, error)
	Release(ctx context.Context, uuids []uuid.UUID) error
	Reserve(ctx context.Context, uuids []uuid.UUID) error
	Commit(ctx context.Context, uuids []uuid.UUID) error
	ValidateCompatibility(ctx context.Context, slots valueobject.ShipSlots) error
}
