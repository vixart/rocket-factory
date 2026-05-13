package model

import "github.com/google/uuid"

type PartFilter struct {
	Uuids    []uuid.UUID
	PartType PartType
}
