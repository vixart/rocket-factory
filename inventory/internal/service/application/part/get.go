package part

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/inventory/internal/model/entity"
)

func (s *service) Get(ctx context.Context, uuid uuid.UUID) (entity.Part, error) {
	part, err := s.partRepo.Get(ctx, uuid)
	if err != nil {
		return entity.Part{}, fmt.Errorf("failed to fetch part: %w", err)
	}

	return part, nil
}
