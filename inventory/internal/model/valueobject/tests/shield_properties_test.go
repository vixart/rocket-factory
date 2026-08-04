package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
)

func TestNewShieldProperties(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   valueobject.ShieldType
		want    valueobject.ShieldType
		wantErr error
	}{
		{
			name:  "valid energy shield",
			input: valueobject.EnergyShield,
			want:  valueobject.EnergyShield,
		},
		{
			name:  "valid plasma shield",
			input: valueobject.PlasmaShield,
			want:  valueobject.PlasmaShield,
		},
		{
			name:    "invalid shield type",
			input:   valueobject.ShieldType("void"),
			wantErr: errs.ErrInvalidProperties,
		},
		{
			name:    "empty shield type",
			input:   valueobject.ShieldType(""),
			wantErr: errs.ErrInvalidProperties,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pp, err := valueobject.NewShieldProperties(tc.input)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Equal(t, valueobject.PartProperties{}, pp)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, pp)

			shield := pp.Shield()

			assert.Equal(t, tc.want, shield.Type())
		})
	}
}

func TestShieldProperties_ConflictsWith(t *testing.T) {
	t.Parallel()

	ppShield, err := valueobject.NewShieldProperties(valueobject.PlasmaShield)
	require.NoError(t, err)

	ppWeaponLaser, err := valueobject.NewWeaponProperties(valueobject.LaserWeapon)
	require.NoError(t, err)

	ppWeaponOther, err := valueobject.NewWeaponProperties(valueobject.MissileWeapon)
	require.NoError(t, err)

	shield := ppShield.Shield()

	assert.True(t, shield.ConflictsWith(ppWeaponLaser.Weapon()))
	assert.False(t, shield.ConflictsWith(ppWeaponOther.Weapon()))
}
