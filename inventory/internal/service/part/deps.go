package part

import (
	"context"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/inventory/internal/model"
)

type Repository interface {
	Get(ctx context.Context, uuid uuid.UUID) (model.Part, error)
	List(_ context.Context, partFilter model.PartFilter) ([]model.Part, error)
}
