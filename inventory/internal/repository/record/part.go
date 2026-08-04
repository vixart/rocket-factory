package record

import (
	"time"

	"github.com/google/uuid"
)

// PartRecord is the flat structure a database row maps onto.
type PartRecord struct {
	UUID          uuid.UUID `db:"uuid"`
	Name          string    `db:"name"`
	Description   string    `db:"description"`
	PartType      string    `db:"part_type"`
	Price         int64     `db:"price"`
	StockQuantity int       `db:"stock_quantity"`
	Reserved      int       `db:"reserved"`
	Properties    []byte    `db:"properties"` // JSONB coming from PostgreSQL
	CreatedAt     time.Time `db:"created_at"`
}
