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

	"github.com/vixart/rocket-factory/inventory/internal/model/entity"
	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
	partService "github.com/vixart/rocket-factory/inventory/internal/service/application/part"
	"github.com/vixart/rocket-factory/inventory/internal/service/application/part/mocks"
	"github.com/vixart/rocket-factory/inventory/internal/service/input"
)

func TestRelease(t *testing.T) {
	t.Parallel()

	type args struct {
		uuids []uuid.UUID
	}

	type expected struct {
		err error
	}

	ctx := context.Background()

	id := uuid.New()

	part := entity.RestorePart(
		id,
		"Engine",
		"desc",
		valueobject.PartTypeEngine,
		5000,
		3,
		1, // важно: reserved > 0 чтобы Release() прошёл
		valueobject.PartProperties{},
		time.Time{},
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(repo *mocks.Repository)
		expected  expected
	}{
		{
			name: "успешный release",
			args: args{
				uuids: []uuid.UUID{id},
			},
			setupMock: func(repo *mocks.Repository) {
				repo.EXPECT().
					List(
						ctx,
						input.PartFilter{
							UUIDs:    []uuid.UUID{id},
							PartType: valueobject.PartTypeUnspecified,
						},
					).
					Return([]entity.Part{part}, nil)

				repo.EXPECT().
					UpdateReservedBatch(
						ctx,
						mock.Anything,
					).
					Return(nil)
			},
			expected: expected{},
		},
		{
			name: "ошибка list",
			args: args{
				uuids: []uuid.UUID{id},
			},
			setupMock: func(repo *mocks.Repository) {
				repo.EXPECT().
					List(
						ctx,
						input.PartFilter{
							UUIDs:    []uuid.UUID{id},
							PartType: valueobject.PartTypeUnspecified,
						},
					).
					Return(nil, errors.New("db error"))
			},
			expected: expected{
				err: errors.New("не удалось получить детали"),
			},
		},
		{
			name: "ошибка domain Release",
			args: args{
				uuids: []uuid.UUID{id},
			},
			setupMock: func(repo *mocks.Repository) {
				repo.EXPECT().
					List(
						ctx,
						input.PartFilter{
							UUIDs:    []uuid.UUID{id},
							PartType: valueobject.PartTypeUnspecified,
						},
					).
					Return([]entity.Part{
						entity.RestorePart(
							id,
							"Engine",
							"desc",
							valueobject.PartTypeEngine,
							5000,
							3,
							0, // reserved = 0 → Release() упадёт
							valueobject.PartProperties{},
							time.Time{},
						),
					}, nil)
			},
			expected: expected{
				err: errors.New("не удалось освободить детали"),
			},
		},
		{
			name: "ошибка update batch",
			args: args{
				uuids: []uuid.UUID{id},
			},
			setupMock: func(repo *mocks.Repository) {
				repo.EXPECT().
					List(
						ctx,
						input.PartFilter{
							UUIDs:    []uuid.UUID{id},
							PartType: valueobject.PartTypeUnspecified,
						},
					).
					Return([]entity.Part{part}, nil)

				repo.EXPECT().
					UpdateReservedBatch(
						ctx,
						mock.Anything,
					).
					Return(errors.New("update error"))
			},
			expected: expected{
				err: errors.New("update error"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := mocks.NewRepository(t)
			checker := mocks.NewCompatibilityChecker(t)

			svc := partService.NewService(repo, checker)

			tc.setupMock(repo)

			err := svc.Release(ctx, tc.args.uuids)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())
				return
			}

			require.NoError(t, err)
		})
	}
}
