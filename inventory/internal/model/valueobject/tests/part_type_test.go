package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
)

func TestNewPartType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    valueobject.PartType
		wantErr error
	}{
		{
			name:  "valid HULL",
			input: "HULL",
			want:  valueobject.PartTypeHull,
		},
		{
			name:  "valid ENGINE",
			input: "ENGINE",
			want:  valueobject.PartTypeEngine,
		},
		{
			name:  "valid SHIELD",
			input: "SHIELD",
			want:  valueobject.PartTypeShield,
		},
		{
			name:  "valid WEAPON",
			input: "WEAPON",
			want:  valueobject.PartTypeWeapon,
		},
		{
			name:    "unknown part type",
			input:   "FUEL",
			wantErr: errs.ErrInvalidProperties,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: errs.ErrInvalidProperties,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := valueobject.NewPartType(tc.input)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Equal(t, valueobject.PartType(""), got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
