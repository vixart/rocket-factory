package part

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/inventory/internal/model"
)

func (s *service) Get(ctx context.Context, uuid uuid.UUID) (model.Part, error) {
	part, err := s.partRepo.Get(ctx, uuid)
	if err != nil {
		return model.Part{}, fmt.Errorf("не удалось получить деталь в сервисе: %w", err)
	}

	return part, nil
}
