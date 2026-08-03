package assembly

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/assembly/internal/model"
)

func (s *service) Assemble(
	ctx context.Context,
	orderUUID uuid.UUID,
	userUUID uuid.UUID,
) error {
	sleepDuration, err := s.generateRandomSleepDuration()
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "сборка корабля начата",
		"order_uuid", orderUUID, "user_uuid", userUUID, "build_time_sec", sleepDuration.Seconds())

	select {
	case <-time.After(sleepDuration):
	case <-ctx.Done():
		slog.WarnContext(ctx, "сборка прервана до завершения",
			"order_uuid", orderUUID, "user_uuid", userUUID, "error", ctx.Err())
		return ctx.Err()
	}

	event := model.ShipAssembledEvent{
		UUID:         orderUUID.String(),
		OrderUUID:    orderUUID.String(),
		UserUUID:     userUUID.String(),
		BuildTimeSec: int64(sleepDuration.Seconds()),
		AssembledAt:  time.Now(),
	}

	err = s.shipAssembledProducer.ProduceShipAssembled(ctx, event)
	if err != nil {
		return fmt.Errorf("не удалось отправить событие ShipAssembled: %w", err)
	}

	slog.InfoContext(ctx, "корабль собран",
		"order_uuid", orderUUID,
		"user_uuid", userUUID,
		"build_time_sec", event.BuildTimeSec,
	)

	return nil
}

func (s *service) generateRandomSleepDuration() (time.Duration, error) {
	var sleepDuration time.Duration
	if s.maxBuildTime > s.minBuildTime {
		diff := int64(s.maxBuildTime - s.minBuildTime)
		n, err := rand.Int(rand.Reader, big.NewInt(diff))
		if err != nil {
			return time.Duration(0), fmt.Errorf("генерация случайного числа: %w", err)
		}
		sleepDuration = time.Duration(n.Int64()) + s.minBuildTime
	} else if s.maxBuildTime == s.minBuildTime {
		sleepDuration = s.minBuildTime
	}

	return sleepDuration, nil
}
