package part

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
	"github.com/vixart/rocket-factory/inventory/internal/service/input"
)

func (s *service) Commit(ctx context.Context, uuids []uuid.UUID) error {
	partFilter := input.PartFilter{
		UUIDs:    uuids,
		PartType: valueobject.PartTypeUnspecified,
	}

	err := s.txManager.Do(ctx, func(ctx context.Context) error {
		parts, err := s.partRepo.ListForUpdate(ctx, partFilter)
		if err != nil {
			return fmt.Errorf("failed to fetch parts: %w", err)
		}

		for i := range parts {
			err = parts[i].Commit()
			if err != nil {
				return fmt.Errorf("failed to commit parts: %w", err)
			}
		}

		return s.partRepo.UpdateReservedBatch(ctx, parts)
	})

	return err
}
