package record

import (
	"time"

	"github.com/google/uuid"
)

// PartRecord — плоская структура для маппинга строки из БД.
type PartRecord struct {
	UUID          uuid.UUID `db:"uuid"`
	Name          string    `db:"name"`
	Description   string    `db:"description"`
	PartType      string    `db:"part_type"`
	Price         int64     `db:"price"`
	StockQuantity int       `db:"stock_quantity"`
	Reserved      int       `db:"reserved"`
	Properties    []byte    `db:"properties"` // JSONB из PostgreSQL
	CreatedAt     time.Time `db:"created_at"`
}
