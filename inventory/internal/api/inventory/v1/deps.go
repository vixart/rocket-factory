package v1

import (
	"context"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/inventory/internal/model"
)

type InventoryService interface {
	Get(ctx context.Context, uuid uuid.UUID) (model.Part, error)
	List(ctx context.Context, uuids []uuid.UUID, partType model.PartType) ([]model.Part, error)
}
