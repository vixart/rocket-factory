package assembly

import (
	"context"

	"github.com/vixart/rocket-factory/assembly/internal/model"
)

type ShipAssembledProducer interface {
	ProduceShipAssembled(ctx context.Context, event model.ShipAssembledEvent) error
}
