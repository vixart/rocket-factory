package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
)

func TestNewWeaponProperties(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   valueobject.WeaponType
		want    valueobject.WeaponType
		wantErr error
	}{
		{
			name:  "valid laser weapon",
			input: valueobject.LaserWeapon,
			want:  valueobject.LaserWeapon,
		},
		{
			name:  "valid missile weapon",
			input: valueobject.MissileWeapon,
			want:  valueobject.MissileWeapon,
		},
		{
			name:    "invalid weapon type",
			input:   valueobject.WeaponType("plasma"),
			wantErr: errs.ErrInvalidProperties,
		},
		{
			name:    "empty weapon type",
			input:   valueobject.WeaponType(""),
			wantErr: errs.ErrInvalidProperties,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pp, err := valueobject.NewWeaponProperties(tc.input)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Equal(t, valueobject.PartProperties{}, pp)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, pp)

			weapon := pp.Weapon()

			assert.Equal(t, tc.want, weapon.Type())
		})
	}
}
