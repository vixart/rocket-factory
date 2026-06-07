package input

import (
	"github.com/google/uuid"

	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
)

type PartFilter struct {
	UUIDs    []uuid.UUID
	PartType valueobject.PartType
}
