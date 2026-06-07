package domain

import (
	"fmt"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model/entity"
	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
)

// compatibilityChecker проверяет совместимость деталей космического корабля.
type compatibilityChecker struct{}

func NewCompatibilityChecker() *compatibilityChecker {
	return &compatibilityChecker{}
}

// Check проверяет бизнес-правила совместимости для набора деталей.
func (c *compatibilityChecker) Check(parts entity.ResolvedShipSlots) error {
	if !parts.Hull.Properties().Hull().CanSupport(parts.Engine.Properties().Engine()) {
		return fmt.Errorf("корпус несовместим с двигателем: %w", errs.ErrIncompatibleParts)
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
		return fmt.Errorf("щит несовместим с оружием: %w", errs.ErrIncompatibleParts)
	}

	return nil
}
