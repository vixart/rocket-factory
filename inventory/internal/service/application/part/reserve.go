package part

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
	"github.com/vixart/rocket-factory/inventory/internal/service/input"
)

func (s *service) Reserve(ctx context.Context, uuids []uuid.UUID) error {
	partFilter := input.PartFilter{
		UUIDs:    uuids,
		PartType: valueobject.PartTypeUnspecified,
	}

	err := s.txManager.Do(ctx, func(ctx context.Context) error {
		parts, err := s.partRepo.ListForUpdate(ctx, partFilter)
		slog.Debug("parts fetched", slog.Any("parts", parts))
		if err != nil {
			return fmt.Errorf("failed to fetch parts: %w", err)
		}

		for i := range parts {
			err = parts[i].Reserve()
			if err != nil {
				return fmt.Errorf("failed to reserve parts: %w", err)
			}
		}

		err = s.partRepo.UpdateReservedBatch(ctx, parts)
		slog.Debug("parts updated", slog.Any("parts", parts))
		return err
	})

	return err
}
