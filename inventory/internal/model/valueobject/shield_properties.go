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

// ShieldProperties — свойства двигателя (Value Object).
type ShieldProperties struct {
	shieldType ShieldType
}

func (s *ShieldProperties) Type() ShieldType { return s.shieldType }

// NewShieldProperties создаёт свойства двигателя.
func NewShieldProperties(shieldType ShieldType) (PartProperties, error) {
	if !shieldType.IsValid() {
		return PartProperties{}, fmt.Errorf("недопустимый тип щита, получено %s: %w", shieldType, errs.ErrInvalidProperties)
	}

	return PartProperties{
		shield: &ShieldProperties{shieldType: shieldType},
	}, nil
}

// ConflictsWith проверяет, выдержит ли корпус нагрузку двигателя.
func (s *ShieldProperties) ConflictsWith(w *WeaponProperties) bool {
	if s.Type() == PlasmaShield && w.Type() == LaserWeapon {
		return true
	}
	return false
}
