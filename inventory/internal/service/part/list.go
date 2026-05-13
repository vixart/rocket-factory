package part

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/inventory/internal/model"
)

func (s *service) List(ctx context.Context, uuids []uuid.UUID, partType model.PartType) ([]model.Part, error) {
	partFilter := model.PartFilter{
		Uuids:    uuids,
		PartType: partType,
	}

	parts, err := s.partRepo.List(ctx, partFilter)
	if err != nil {
		return []model.Part{}, fmt.Errorf("не удалось получить детали: %w", err)
	}

	return parts, nil
}
