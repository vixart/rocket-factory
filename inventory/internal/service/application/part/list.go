package part

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/inventory/internal/model/entity"
	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
	"github.com/vixart/rocket-factory/inventory/internal/service/input"
)

func (s *service) List(ctx context.Context, uuids []uuid.UUID, partType valueobject.PartType) ([]entity.Part, error) {
	partFilter := input.PartFilter{
		UUIDs:    uuids,
		PartType: partType,
	}

	parts, err := s.partRepo.List(ctx, partFilter)
	if err != nil {
		return []entity.Part{}, fmt.Errorf("не удалось получить детали: %w", err)
	}

	return parts, nil
}
