package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vixart/rocket-factory/inventory/internal/model/entity"
	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
	partService "github.com/vixart/rocket-factory/inventory/internal/service/application/part"
	"github.com/vixart/rocket-factory/inventory/internal/service/application/part/mocks"
	"github.com/vixart/rocket-factory/inventory/internal/service/input"
)

func TestList(t *testing.T) {
	t.Parallel()

	type args struct {
		uuids    []uuid.UUID
		partType valueobject.PartType
	}

	type expected struct {
		parts []entity.Part
		err   error
	}

	ctx := context.Background()

	uuid1 := uuid.New()
	uuid2 := uuid.New()

	properties, _ := valueobject.NewEngineProperties(
		valueobject.EngineClassA,
		100,
	)

	part1 := entity.RestorePart(
		uuid1,
		"Engine A",
		"desc",
		valueobject.PartTypeEngine,
		5000,
		3,
		0,
		properties,
		time.Time{},
	)

	part2 := entity.RestorePart(
		uuid2,
		"Engine B",
		"desc",
		valueobject.PartTypeEngine,
		7000,
		5,
		0,
		properties,
		time.Time{},
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(repo *mocks.Repository, checker *mocks.CompatibilityChecker)
		expected  expected
	}{
		{
			name: "part list fetched successfully",
			args: args{
				uuids:    []uuid.UUID{uuid1, uuid2},
				partType: valueobject.PartTypeEngine,
			},
			setupMock: func(repo *mocks.Repository, checker *mocks.CompatibilityChecker) {
				repo.EXPECT().
					List(
						ctx,
						input.PartFilter{
							UUIDs:    []uuid.UUID{uuid1, uuid2},
							PartType: valueobject.PartTypeEngine,
						},
					).
					Return([]entity.Part{part1, part2}, nil)
			},
			expected: expected{
				parts: []entity.Part{part1, part2},
				err:   nil,
			},
		},
		{
			name: "repository fails",
			args: args{
				uuids:    []uuid.UUID{uuid1},
				partType: valueobject.PartTypeEngine,
			},
			setupMock: func(repo *mocks.Repository, checker *mocks.CompatibilityChecker) {
				repo.EXPECT().
					List(
						ctx,
						input.PartFilter{
							UUIDs:    []uuid.UUID{uuid1},
							PartType: valueobject.PartTypeEngine,
						},
					).
					Return(nil, errors.New("db error"))
			},
			expected: expected{
				parts: nil,
				err:   errors.New("db error"),
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

			res, err := svc.List(ctx, tc.args.uuids, tc.args.partType)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())
				assert.Empty(t, res)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expected.parts, res)
		})
	}
}
