package valueobject

import (
	"fmt"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
)

type WeaponType string

const (
	LaserWeapon   WeaponType = "laser"
	MissileWeapon WeaponType = "missile"
)

func (w WeaponType) IsValid() bool {
	switch w {
	case LaserWeapon, MissileWeapon:
		return true
	default:
		return false
	}
}

// WeaponProperties — свойства двигателя (Value Object).
type WeaponProperties struct {
	weaponType WeaponType
}

func (w *WeaponProperties) Type() WeaponType { return w.weaponType }

// NewWeaponProperties создаёт свойства двигателя.
func NewWeaponProperties(weaponType WeaponType) (PartProperties, error) {
	if !weaponType.IsValid() {
		return PartProperties{}, fmt.Errorf("недопустимый тип оружия, получено %s: %w", weaponType, errs.ErrInvalidProperties)
	}

	return PartProperties{
		weapon: &WeaponProperties{weaponType: weaponType},
	}, nil
}
