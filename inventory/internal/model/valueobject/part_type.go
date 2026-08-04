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

// NewPartType creates a validated part type.
func NewPartType(s string) (PartType, error) {
	pt := PartType(s)
	switch pt {
	case PartTypeHull, PartTypeEngine, PartTypeShield, PartTypeWeapon:
		return pt, nil
	default:
		return "", fmt.Errorf("unknown part type %q: %w", s, errs.ErrInvalidProperties)
	}
}
