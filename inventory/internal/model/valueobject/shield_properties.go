package valueobject

import (
	"fmt"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
)

type ShieldType string

const (
	EnergyShield ShieldType = "energy"
	PlasmaShield ShieldType = "plasma"
)

func (s ShieldType) IsValid() bool {
	switch s {
	case EnergyShield, PlasmaShield:
		return true
	default:
		return false
	}
}

// ShieldProperties are the shield properties (value object).
type ShieldProperties struct {
	shieldType ShieldType
}

func (s *ShieldProperties) Type() ShieldType { return s.shieldType }

// NewShieldProperties creates shield properties.
func NewShieldProperties(shieldType ShieldType) (PartProperties, error) {
	if !shieldType.IsValid() {
		return PartProperties{}, fmt.Errorf("invalid shield type, got %s: %w", shieldType, errs.ErrInvalidProperties)
	}

	return PartProperties{
		shield: &ShieldProperties{shieldType: shieldType},
	}, nil
}

// ConflictsWith reports whether the shield conflicts with the weapon.
func (s *ShieldProperties) ConflictsWith(w *WeaponProperties) bool {
	if s.Type() == PlasmaShield && w.Type() == LaserWeapon {
		return true
	}
	return false
}
