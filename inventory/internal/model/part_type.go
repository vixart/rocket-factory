package model

type PartType string

const (
	PartTypeEngine      PartType = "engine"
	PartTypeHull        PartType = "hull"
	PartTypeShield      PartType = "shield"
	PartTypeWeapon      PartType = "weapon"
	PartTypeUnspecified PartType = "unspecified"
)
