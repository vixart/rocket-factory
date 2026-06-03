package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
)

func TestNewEngineProperties(t *testing.T) {
	t.Parallel()

	type args struct {
		class            valueobject.EngineClass
		requiredStrength int
	}

	tests := []struct {
		name      string
		args      args
		wantClass valueobject.EngineClass
		wantStr   int
		wantErr   error
	}{
		{
			name: "валидный A класс",
			args: args{
				class:            valueobject.EngineClassA,
				requiredStrength: 100,
			},
			wantClass: valueobject.EngineClassA,
			wantStr:   100,
		},
		{
			name: "валидный B класс",
			args: args{
				class:            valueobject.EngineClassB,
				requiredStrength: 150,
			},
			wantClass: valueobject.EngineClassB,
			wantStr:   150,
		},
		{
			name: "валидный C класс",
			args: args{
				class:            valueobject.EngineClassC,
				requiredStrength: 200,
			},
			wantClass: valueobject.EngineClassC,
			wantStr:   200,
		},
		{
			name: "невалидный класс двигателя",
			args: args{
				class:            valueobject.EngineClass("X"),
				requiredStrength: 100,
			},
			wantErr: errs.ErrInvalidProperties,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pp, err := valueobject.NewEngineProperties(tc.args.class, tc.args.requiredStrength)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Equal(t, valueobject.PartProperties{}, pp)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, pp)

			engine := pp.Engine()

			assert.Equal(t, tc.wantClass, engine.Class())
			assert.Equal(t, tc.wantStr, engine.RequiredStrength())
		})
	}
}
