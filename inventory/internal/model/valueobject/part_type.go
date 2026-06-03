package valueobject

import (
	"fmt"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
)

type PartType string

const (
	PartTypeEngine      PartType = "ENGINE"
	PartTypeHull        PartType = "HULL"
	PartTypeShield      PartType = "SHIELD"
	PartTypeWeapon      PartType = "WEAPON"
	PartTypeUnspecified PartType = "UNSPECIFIED"
)

// NewPartType создаёт тип детали с валидацией.
func NewPartType(s string) (PartType, error) {
	pt := PartType(s)
	switch pt {
	case PartTypeHull, PartTypeEngine, PartTypeShield, PartTypeWeapon:
		return pt, nil
	default:
		return "", fmt.Errorf("неизвестный тип детали %q: %w", s, errs.ErrInvalidProperties)
	}
}
