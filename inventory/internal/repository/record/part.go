package record

import (
	"time"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/inventory/internal/model"
)

type Part struct {
	UUID          uuid.UUID
	Name          string
	Description   string
	Price         int64 // в копейках
	PartType      model.PartType
	StockQuantity int64
	CreatedAt     *time.Time
}
