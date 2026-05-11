package part

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model"
	"github.com/vixart/rocket-factory/inventory/internal/repository/converter"
)

func (r *repository) List(_ context.Context, uuids []uuid.UUID, partType model.PartType) ([]model.Part, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	parts := make([]model.Part, 0)
	if len(uuids) > 0 {
		for _, partUuid := range uuids {
			part, ok := r.data[partUuid]
			if !ok {
				return []model.Part{}, fmt.Errorf("деталь не найдена у репозитории: %w", errs.ErrPartNotFound)
			}

			parts = append(parts, converter.PartRecordToModel(part))
		}
	} else {
		for _, part := range r.data {
			if partType == model.PartTypeUnspecified || partType == part.PartType {
				parts = append(parts, converter.PartRecordToModel(part))
			}
		}

		sort.Slice(parts, func(i, j int) bool {
			return parts[i].Name < parts[j].Name
		})
	}

	return parts, nil
}
