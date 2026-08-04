package tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model/entity"
	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
	"github.com/vixart/rocket-factory/inventory/internal/service/domain"
)

func TestCompatibilityChecker_Check(t *testing.T) {
	t.Parallel()

	type args struct {
		parts entity.ResolvedShipSlots
	}

	type expected struct {
		err error
	}

	hullID := uuid.New()
	engineID := uuid.New()

	hullProps, _ := valueobject.NewHullProperties(120)
	engineProps, _ := valueobject.NewEngineProperties(valueobject.EngineClassA, 80)

	shieldProps, _ := valueobject.NewShieldProperties(valueobject.PlasmaShield)
	weaponProps, _ := valueobject.NewWeaponProperties(valueobject.LaserWeapon)

	hull := entity.RestorePart(
		hullID,
		"hull",
		"",
		valueobject.PartTypeHull,
		0,
		0,
		0,
		hullProps,
		time.Time{},
	)

	engine := entity.RestorePart(
		engineID,
		"engine",
		"",
		valueobject.PartTypeEngine,
		0,
		0,
		0,
		engineProps,
		time.Time{},
	)

	shield := entity.RestorePart(
		uuid.New(),
		"shield",
		"",
		valueobject.PartTypeShield,
		0,
		0,
		0,
		shieldProps,
		time.Time{},
	)

	weapon := entity.RestorePart(
		uuid.New(),
		"weapon",
		"",
		valueobject.PartTypeWeapon,
		0,
		0,
		0,
		weaponProps,
		time.Time{},
	)

	tests := []struct {
		name     string
		args     args
		expected expected
	}{
		{
			name: "parts are compatible",
			args: args{
				parts: entity.ResolvedShipSlots{
					Hull:   hull,
					Engine: engine,
				},
			},
			expected: expected{},
		},
		{
			name: "hull cannot support the engine",
			args: args{
				parts: entity.ResolvedShipSlots{
					Hull: entity.RestorePart(
						uuid.New(),
						"bad-hull",
						"",
						valueobject.PartTypeHull,
						0,
						0,
						0,
						func() valueobject.PartProperties {
							p, _ := valueobject.NewHullProperties(30)
							return p
						}(),
						time.Time{},
					),
					Engine: engine,
				},
			},
			expected: expected{
				err: errs.ErrIncompatibleParts,
			},
		},
		{
			name: "shield conflicts with the weapon",
			args: args{
				parts: entity.ResolvedShipSlots{
					Hull:   hull,
					Engine: engine,
					Shield: &shield,
					Weapon: &weapon,
				},
			},
			expected: expected{
				err: errs.ErrIncompatibleParts,
			},
		},
		{
			name: "shield and weapon are absent — OK",
			args: args{
				parts: entity.ResolvedShipSlots{
					Hull:   hull,
					Engine: engine,
				},
			},
			expected: expected{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			checker := domain.NewCompatibilityChecker()

			err := checker.Check(tc.args.parts)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, errs.ErrIncompatibleParts)
				return
			}

			require.NoError(t, err)
		})
	}
}
