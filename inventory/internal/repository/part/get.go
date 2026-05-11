package part

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model"
	"github.com/vixart/rocket-factory/inventory/internal/repository/converter"
)

func (r *repository) Get(_ context.Context, uuid uuid.UUID) (model.Part, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	part, ok := r.data[uuid]
	if !ok {
		return model.Part{}, fmt.Errorf("деталь не найдена в репозитории: %w", errs.ErrPartNotFound)
	}

	return converter.PartRecordToModel(part), nil
}
