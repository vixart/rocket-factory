package assembly

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vixart/rocket-factory/assembly/internal/model"
)

func (s *service) Assemble(
	ctx context.Context,
	orderUUID uuid.UUID,
	userUUID uuid.UUID,
) error {
	time.Sleep(time.Millisecond * 3000)

	event := model.ShipAssembledEvent{
		UUID:         orderUUID.String(),
		OrderUUID:    orderUUID.String(),
		UserUUID:     userUUID.String(),
		BuildTimeSec: int64(3),
		AssembledAt:  time.Now(),
	}

	err := s.shipAssembledProducer.ProduceShipAssembled(ctx, event)
	if err != nil {
		return fmt.Errorf("не удалось отправить событие ShipAssembled: %w", err)
	}

	return nil
}
