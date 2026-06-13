package part

import (
	"context"

	"github.com/vixart/rocket-factory/inventory/internal/model/entity"
	"github.com/vixart/rocket-factory/inventory/internal/service/input"
)

func (r *repository) ListForUpdate(
	ctx context.Context,
	partFilter input.PartFilter,
) ([]entity.Part, error) {
	return r.list(ctx, partFilter, true)
}
