package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model"
	partService "github.com/vixart/rocket-factory/inventory/internal/service/part"
	"github.com/vixart/rocket-factory/inventory/internal/service/part/mocks"
)

func TestList(t *testing.T) {
	t.Parallel()

	type args struct {
		uuids    []uuid.UUID
		partType model.PartType
	}

	type expected struct {
		parts []model.Part
		err   error
	}

	ctx := context.Background()

	hullUUID := uuid.New()
	engineUUID := uuid.New()

	hullPart := model.Part{
		UUID:          hullUUID,
		Name:          "Hull",
		Price:         100000,
		StockQuantity: 5,
	}

	enginePart := model.Part{
		UUID:          engineUUID,
		Name:          "Engine",
		Price:         50000,
		StockQuantity: 3,
	}

	tests := []struct {
		name      string
		args      args
		setupMock func(repo *mocks.Repository)
		expected  expected
	}{
		{
			name: "успешное получение списка деталей",
			args: args{
				uuids:    []uuid.UUID{hullUUID, engineUUID},
				partType: model.PartTypeHull,
			},
			setupMock: func(repo *mocks.Repository) {
				repo.EXPECT().
					List(ctx, model.PartFilter{
						Uuids:    []uuid.UUID{hullUUID, engineUUID},
						PartType: model.PartTypeHull,
					}).
					Return([]model.Part{hullPart, enginePart}, nil)
			},
			expected: expected{
				parts: []model.Part{hullPart, enginePart},
				err:   nil,
			},
		},
		{
			name: "пустой список uuid",
			args: args{
				uuids:    []uuid.UUID{},
				partType: model.PartTypeEngine,
			},
			setupMock: func(repo *mocks.Repository) {
				repo.EXPECT().
					List(ctx, model.PartFilter{
						Uuids:    []uuid.UUID{},
						PartType: model.PartTypeEngine,
					}).
					Return([]model.Part{}, nil)
			},
			expected: expected{
				parts: []model.Part{},
				err:   nil,
			},
		},
		{
			name: "детали не найдены",
			args: args{
				uuids:    []uuid.UUID{hullUUID},
				partType: model.PartTypeHull,
			},
			setupMock: func(repo *mocks.Repository) {
				repo.EXPECT().
					List(ctx, mock.Anything).
					Return(nil, errs.ErrPartNotFound)
			},
			expected: expected{
				parts: []model.Part{},
				err:   errs.ErrPartNotFound,
			},
		},
		{
			name: "ошибка репозитория",
			args: args{
				uuids:    []uuid.UUID{hullUUID},
				partType: model.PartTypeHull,
			},
			setupMock: func(repo *mocks.Repository) {
				repo.EXPECT().
					List(ctx, mock.Anything).
					Return(nil, errors.New("db error"))
			},
			expected: expected{
				parts: []model.Part{},
				err:   errors.New("db error"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := mocks.NewRepository(t)
			svc := partService.NewService(repo)

			tc.setupMock(repo)

			res, err := svc.List(ctx, tc.args.uuids, tc.args.partType)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())
				assert.Equal(t, tc.expected.parts, res)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expected.parts, res)
		})
	}
}
