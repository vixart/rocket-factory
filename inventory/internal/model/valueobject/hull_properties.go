package valueobject

import (
	"fmt"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
)

// HullProperties are the hull properties (value object).
type HullProperties struct {
	strength int
}

func (h *HullProperties) Strength() int { return h.strength }

// NewHullProperties creates hull properties. Strength must be between 30 and 200.
func NewHullProperties(strength int) (PartProperties, error) {
	if strength < 30 || strength > 200 {
		return PartProperties{}, fmt.Errorf("hull strength must be between 30 and 200, got %d: %w", strength, errs.ErrInvalidProperties)
	}
	return PartProperties{
		hull: &HullProperties{strength: strength},
	}, nil
}

// CanSupport reports whether the hull can carry the engine.
func (h *HullProperties) CanSupport(e *EngineProperties) bool {
	return h.strength >= e.requiredStrength
}
