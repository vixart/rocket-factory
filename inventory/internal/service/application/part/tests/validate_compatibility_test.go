package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model/entity"
	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
	partService "github.com/vixart/rocket-factory/inventory/internal/service/application/part"
	"github.com/vixart/rocket-factory/inventory/internal/service/application/part/mocks"
)

func TestValidateCompatibility(t *testing.T) {
	t.Parallel()

	type args struct {
		slots valueobject.ShipSlots
	}

	type expected struct {
		err error
	}

	ctx := context.Background()

	hullID := uuid.New()
	engineID := uuid.New()
	shieldID := uuid.New()
	weaponID := uuid.New()

	hull, _ := valueobject.NewHullProperties(100)
	engine, _ := valueobject.NewEngineProperties(valueobject.EngineClassA, 80)
	shield, _ := valueobject.NewShieldProperties(valueobject.EnergyShield)
	weapon, _ := valueobject.NewWeaponProperties(valueobject.LaserWeapon)

	hullPart := entity.RestorePart(hullID, "hull", "desc", valueobject.PartTypeHull, 0, 0, 0, hull, time.Time{})
	enginePart := entity.RestorePart(engineID, "engine", "desc", valueobject.PartTypeEngine, 0, 0, 0, engine, time.Time{})
	shieldPart := entity.RestorePart(shieldID, "shield", "desc", valueobject.PartTypeShield, 0, 0, 0, shield, time.Time{})
	weaponPart := entity.RestorePart(weaponID, "weapon", "desc", valueobject.PartTypeWeapon, 0, 0, 0, weapon, time.Time{})

	tests := []struct {
		name      string
		args      args
		setupMock func(repo *mocks.Repository, checker *mocks.CompatibilityChecker)
		expected  expected
	}{
		{
			name: "all slots validate successfully",
			args: args{
				slots: valueobject.ShipSlots{
					HullUUID:   hullID,
					EngineUUID: engineID,
					ShieldUUID: shieldID,
					WeaponUUID: weaponID,
				},
			},
			setupMock: func(repo *mocks.Repository, checker *mocks.CompatibilityChecker) {
				repo.EXPECT().
					List(ctx, mock.Anything).
					Return([]entity.Part{
						hullPart,
						enginePart,
						shieldPart,
						weaponPart,
					}, nil)

				checker.EXPECT().
					Check(mock.Anything).
					Return(nil)
			},
			expected: expected{},
		},
		{
			name: "error: required hull uuid is missing",
			args: args{
				slots: valueobject.ShipSlots{
					HullUUID:   uuid.Nil,
					EngineUUID: engineID,
				},
			},
			setupMock: func(repo *mocks.Repository, checker *mocks.CompatibilityChecker) {},
			expected: expected{
				err: errs.ErrInvalidUUID,
			},
		},
		{
			name: "error: part not found",
			args: args{
				slots: valueobject.ShipSlots{
					HullUUID:   hullID,
					EngineUUID: engineID,
				},
			},
			setupMock: func(repo *mocks.Repository, checker *mocks.CompatibilityChecker) {
				repo.EXPECT().
					List(ctx, mock.Anything).
					Return([]entity.Part{
						enginePart,
					}, nil)
			},
			expected: expected{
				err: errs.ErrPartNotFound,
			},
		},
		{
			name: "error: type mismatch",
			args: args{
				slots: valueobject.ShipSlots{
					HullUUID:   hullID,
					EngineUUID: engineID,
				},
			},
			setupMock: func(repo *mocks.Repository, checker *mocks.CompatibilityChecker) {
				// intentionally wrong type for hull
				wrongHull := entity.RestorePart(
					hullID,
					"hull",
					"desc",
					valueobject.PartTypeEngine, // mismatch
					0, 0, 0,
					valueobject.PartProperties{},
					time.Time{},
				)

				repo.EXPECT().
					List(ctx, mock.Anything).
					Return([]entity.Part{
						wrongHull,
						enginePart,
					}, nil)
			},
			expected: expected{
				err: errs.ErrPartTypeMismatch,
			},
		},
		{
			name: "repository list fails",
			args: args{
				slots: valueobject.ShipSlots{
					HullUUID:   hullID,
					EngineUUID: engineID,
				},
			},
			setupMock: func(repo *mocks.Repository, checker *mocks.CompatibilityChecker) {
				repo.EXPECT().
					List(ctx, mock.Anything).
					Return(nil, errors.New("db error"))
			},
			expected: expected{
				err: errors.New("fetch parts"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := mocks.NewRepository(t)
			checker := mocks.NewCompatibilityChecker(t)
			txManager := mocks.NewTxManager(t)

			svc := partService.NewService(txManager, repo, checker)

			tc.setupMock(repo, checker)

			err := svc.ValidateCompatibility(ctx, tc.args.slots)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())
				return
			}

			require.NoError(t, err)
		})
	}
}
