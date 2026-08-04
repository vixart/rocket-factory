package model

import "github.com/google/uuid"

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
	PartType      PartType
	Name          string
	Price         int64 // in kopecks
	StockQuantity int64
}
