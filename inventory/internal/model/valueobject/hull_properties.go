package valueobject

import (
	"fmt"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
)

// HullProperties — свойства корпуса (Value Object).
type HullProperties struct {
	strength int
}

func (h *HullProperties) Strength() int { return h.strength }

// NewHullProperties создаёт свойства корпуса. Прочность должна быть в диапазоне 30–200.
func NewHullProperties(strength int) (PartProperties, error) {
	if strength < 30 || strength > 200 {
		return PartProperties{}, fmt.Errorf("прочность корпуса должна быть от 30 до 200, получено %d: %w", strength, errs.ErrInvalidProperties)
	}
	return PartProperties{
		hull: &HullProperties{strength: strength},
	}, nil
}

// CanSupport проверяет, выдержит ли корпус нагрузку двигателя.
func (h *HullProperties) CanSupport(e *EngineProperties) bool {
	return h.strength >= e.requiredStrength
}
