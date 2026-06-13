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
		slog.Debug("детали получены", slog.Any("parts", parts))
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
		slog.Debug("детали обновлены", slog.Any("parts", parts))
		return err
	})

	return err
}
