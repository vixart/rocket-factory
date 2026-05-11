package part

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/inventory/internal/model"
)

func (s *service) List(ctx context.Context, uuids []uuid.UUID, partType model.PartType) ([]model.Part, error) {
	parts, err := s.partRepo.List(ctx, uuids, partType)
	if err != nil {
		return []model.Part{}, fmt.Errorf("не удалось получить детали в сервисе: %w", err)
	}

	return parts, nil
}
