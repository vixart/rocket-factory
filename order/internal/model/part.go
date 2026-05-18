package model

import "github.com/google/uuid"

type Part struct {
	UUID          uuid.UUID
	Name          string
	Price         int64 // в копейках
	StockQuantity int64
}
