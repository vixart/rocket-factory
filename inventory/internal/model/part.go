package model

import (
	"time"

	"github.com/google/uuid"
)

type PartType string

const (
	PartTypeEngine      PartType = "engine"
	PartTypeHull        PartType = "hull"
	PartTypeShield      PartType = "shield"
	PartTypeWeapon      PartType = "weapon"
	PartTypeUnspecified PartType = "unspecified"
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

type PartFilter struct {
	Uuids    []uuid.UUID
	PartType PartType
}
