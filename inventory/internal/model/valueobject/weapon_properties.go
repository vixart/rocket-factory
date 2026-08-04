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

// WeaponProperties are the weapon properties (value object).
type WeaponProperties struct {
	weaponType WeaponType
}

func (w *WeaponProperties) Type() WeaponType { return w.weaponType }

// NewWeaponProperties creates weapon properties.
func NewWeaponProperties(weaponType WeaponType) (PartProperties, error) {
	if !weaponType.IsValid() {
		return PartProperties{}, fmt.Errorf("invalid weapon type, got %s: %w", weaponType, errs.ErrInvalidProperties)
	}

	return PartProperties{
		weapon: &WeaponProperties{weaponType: weaponType},
	}, nil
}
