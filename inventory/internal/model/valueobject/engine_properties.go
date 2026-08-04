package valueobject

import (
	"fmt"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
)

type EngineClass string

const (
	EngineClassA EngineClass = "A"
	EngineClassB EngineClass = "B"
	EngineClassC EngineClass = "C"
)

func (c EngineClass) IsValid() bool {
	switch c {
	case EngineClassA,
		EngineClassB,
		EngineClassC:
		return true
	default:
		return false
	}
}

// EngineProperties are the engine properties (value object).
type EngineProperties struct {
	class            EngineClass
	requiredStrength int
}

func (e *EngineProperties) Class() EngineClass { return e.class }

func (e *EngineProperties) RequiredStrength() int { return e.requiredStrength }

// NewEngineProperties creates engine properties.
func NewEngineProperties(class EngineClass, requiredStrength int) (PartProperties, error) {
	if !class.IsValid() {
		return PartProperties{}, fmt.Errorf("invalid engine class, got %s: %w", class, errs.ErrInvalidProperties)
	}

	return PartProperties{
		engine: &EngineProperties{class: class, requiredStrength: requiredStrength},
	}, nil
}
