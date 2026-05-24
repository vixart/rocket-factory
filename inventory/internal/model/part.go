package model

import (
	"time"

	"github.com/google/uuid"
)

type PartType string

const (
	PartTypeEngine      PartType = "ENGINE"
	PartTypeHull        PartType = "HULL"
	PartTypeShield      PartType = "SHIELD"
	PartTypeWeapon      PartType = "WEAPON"
	PartTypeUnspecified PartType = "UNSPECIFIED"
)

type Part struct {
	UUID          uuid.UUID
	Name          string
	Description   string
	Price         int64
	PartType      PartType
	StockQuantity int64
	CreatedAt     time.Time
}

type PartFilter struct {
	Uuids    []uuid.UUID
	PartType PartType
}
