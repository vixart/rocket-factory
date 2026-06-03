package part

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
	"github.com/vixart/rocket-factory/inventory/internal/service/input"
)

func (s *service) Reserve(ctx context.Context, uuids []uuid.UUID) error {
	partFilter := input.PartFilter{
		UUIDs:    uuids,
		PartType: valueobject.PartTypeUnspecified,
	}

	parts, err := s.partRepo.List(ctx, partFilter)
	if err != nil {
		return fmt.Errorf("не удалось получить детали: %w", err)
	}

	for i := range parts {
		err = parts[i].Reserve()
		if err != nil {
			return fmt.Errorf("не удалось зарезервировать детали детали: %w", err)
		}
	}

	err = s.partRepo.UpdateReservedBatch(ctx, parts)
	if err != nil {
		return err
	}

	return nil
}
