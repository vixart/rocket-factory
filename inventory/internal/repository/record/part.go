package record

import (
	"time"

	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/inventory/internal/model"
)

type Part struct {
	UUID          uuid.UUID      `db:"uuid"`
	Name          string         `db:"name"`
	Description   string         `db:"description"`
	Price         int64          `db:"price"` // в копейках
	PartType      model.PartType `db:"part_type"`
	StockQuantity int64          `db:"stock_quantity"`
	CreatedAt     *time.Time     `db:"created_at"`
}
