package domain

import (
	"fmt"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model/entity"
	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
)

// compatibilityChecker validates the compatibility of spaceship parts.
type compatibilityChecker struct{}

func NewCompatibilityChecker() *compatibilityChecker {
	return &compatibilityChecker{}
}

// Check applies the compatibility business rules to a set of parts.
func (c *compatibilityChecker) Check(parts entity.ResolvedShipSlots) error {
	if !parts.Hull.Properties().Hull().CanSupport(parts.Engine.Properties().Engine()) {
		return fmt.Errorf("hull is incompatible with the engine: %w", errs.ErrIncompatibleParts)
	}

	var shieldProperties *valueobject.ShieldProperties
	if parts.Shield != nil {
		shieldProperties = parts.Shield.Properties().Shield()
	}

	var weaponProperties *valueobject.WeaponProperties
	if parts.Weapon != nil {
		weaponProperties = parts.Weapon.Properties().Weapon()
	}

	if shieldProperties != nil && weaponProperties != nil && shieldProperties.ConflictsWith(weaponProperties) {
		return fmt.Errorf("shield is incompatible with the weapon: %w", errs.ErrIncompatibleParts)
	}

	return nil
}
