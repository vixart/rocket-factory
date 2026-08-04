package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
)

func TestNewHullProperties(t *testing.T) {
	t.Parallel()

	type args struct {
		strength int
	}

	tests := []struct {
		name    string
		args    args
		wantStr int
		wantErr error
	}{
		{
			name: "valid strength, lower bound",
			args: args{
				strength: 30,
			},
			wantStr: 30,
		},
		{
			name: "valid strength, upper bound",
			args: args{
				strength: 200,
			},
			wantStr: 200,
		},
		{
			name: "valid strength, middle of the range",
			args: args{
				strength: 100,
			},
			wantStr: 100,
		},
		{
			name: "strength below the minimum",
			args: args{
				strength: 29,
			},
			wantErr: errs.ErrInvalidProperties,
		},
		{
			name: "strength above the maximum",
			args: args{
				strength: 201,
			},
			wantErr: errs.ErrInvalidProperties,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pp, err := valueobject.NewHullProperties(tc.args.strength)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Equal(t, valueobject.PartProperties{}, pp)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, pp)

			hull := pp.Hull()

			assert.Equal(t, tc.wantStr, hull.Strength())
		})
	}
}

func TestHullProperties_CanSupport(t *testing.T) {
	t.Parallel()

	ppHull, _ := valueobject.NewHullProperties(100)
	ppEngine, _ := valueobject.NewEngineProperties(valueobject.EngineClassA, 80)

	hull := ppHull.Hull()
	engine := ppEngine.Engine()

	assert.True(t, hull.CanSupport(engine))
}
