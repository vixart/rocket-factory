package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vixart/rocket-factory/inventory/internal/api/inventory/v1"
	"github.com/vixart/rocket-factory/inventory/internal/api/inventory/v1/mocks"
	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
)

func TestValidateCompatibility(t *testing.T) {
	t.Parallel()

	type args struct {
		req *inventoryv1.ValidateCompatibilityRequest
	}

	type expected struct {
		err error
	}

	var (
		ctx = context.Background()

		hullUUID   = uuid.New()
		engineUUID = uuid.New()
		shieldUUID = uuid.New()
		weaponUUID = uuid.New()

		serviceErr = errors.New("service error")
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(svc *mocks.InventoryService)
		expected  expected
	}{
		{
			name: "compatibility check succeeds for required slots only",
			args: args{
				req: &inventoryv1.ValidateCompatibilityRequest{
					HullUuid:   hullUUID.String(),
					EngineUuid: engineUUID.String(),
				},
			},
			setupMock: func(svc *mocks.InventoryService) {
				svc.EXPECT().
					ValidateCompatibility(
						ctx,
						valueobject.ShipSlots{
							HullUUID:   hullUUID,
							EngineUUID: engineUUID,
						},
					).
					Return(nil)
			},
		},
		{
			name: "compatibility check succeeds for all slots",
			args: args{
				req: &inventoryv1.ValidateCompatibilityRequest{
					HullUuid:   hullUUID.String(),
					EngineUuid: engineUUID.String(),
					ShieldUuid: shieldUUID.String(),
					WeaponUuid: weaponUUID.String(),
				},
			},
			setupMock: func(svc *mocks.InventoryService) {
				svc.EXPECT().
					ValidateCompatibility(
						ctx,
						valueobject.ShipSlots{
							HullUUID:   hullUUID,
							EngineUUID: engineUUID,
							ShieldUUID: shieldUUID,
							WeaponUUID: weaponUUID,
						},
					).
					Return(nil)
			},
		},
		{
			name: "invalid hull uuid",
			args: args{
				req: &inventoryv1.ValidateCompatibilityRequest{
					HullUuid:   "invalid",
					EngineUuid: engineUUID.String(),
				},
			},
			setupMock: func(svc *mocks.InventoryService) {},
			expected: expected{
				err: errs.ErrInvalidUUID,
			},
		},
		{
			name: "invalid engine uuid",
			args: args{
				req: &inventoryv1.ValidateCompatibilityRequest{
					HullUuid:   hullUUID.String(),
					EngineUuid: "invalid",
				},
			},
			setupMock: func(svc *mocks.InventoryService) {},
			expected: expected{
				err: errs.ErrInvalidUUID,
			},
		},
		{
			name: "invalid shield uuid",
			args: args{
				req: &inventoryv1.ValidateCompatibilityRequest{
					HullUuid:   hullUUID.String(),
					EngineUuid: engineUUID.String(),
					ShieldUuid: "invalid",
				},
			},
			setupMock: func(svc *mocks.InventoryService) {},
			expected: expected{
				err: errs.ErrInvalidUUID,
			},
		},
		{
			name: "invalid weapon uuid",
			args: args{
				req: &inventoryv1.ValidateCompatibilityRequest{
					HullUuid:   hullUUID.String(),
					EngineUuid: engineUUID.String(),
					WeaponUuid: "invalid",
				},
			},
			setupMock: func(svc *mocks.InventoryService) {},
			expected: expected{
				err: errs.ErrInvalidUUID,
			},
		},
		{
			name: "service fails",
			args: args{
				req: &inventoryv1.ValidateCompatibilityRequest{
					HullUuid:   hullUUID.String(),
					EngineUuid: engineUUID.String(),
				},
			},
			setupMock: func(svc *mocks.InventoryService) {
				svc.EXPECT().
					ValidateCompatibility(
						ctx,
						valueobject.ShipSlots{
							HullUUID:   hullUUID,
							EngineUUID: engineUUID,
						},
					).
					Return(serviceErr)
			},
			expected: expected{
				err: serviceErr,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockSvc := mocks.NewInventoryService(t)
			api := v1.NewApi(mockSvc)

			tc.setupMock(mockSvc)

			res, err := api.ValidateCompatibility(ctx, tc.args.req)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())
				assert.Nil(t, res)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)
		})
	}
}
