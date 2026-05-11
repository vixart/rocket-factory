package model

import (
	"time"

	"github.com/google/uuid"
)

type Part struct {
	UUID          uuid.UUID
	Name          string
	Description   string
	Price         int64 // в копейках
	PartType      PartType
	StockQuantity int64
	CreatedAt     *time.Time
}
