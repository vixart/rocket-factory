package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model"
	partService "github.com/vixart/rocket-factory/inventory/internal/service/part"
	"github.com/vixart/rocket-factory/inventory/internal/service/part/mocks"
)

func TestGet(t *testing.T) {
	t.Parallel()

	type args struct {
		uuid uuid.UUID
	}

	type expected struct {
		part model.Part
		err  error
	}

	ctx := context.Background()

	partUUID := uuid.New()

	part := model.Part{
		UUID:          partUUID,
		Name:          "Engine",
		Price:         5000,
		StockQuantity: 3,
	}

	tests := []struct {
		name      string
		args      args
		setupMock func(repo *mocks.Repository)
		expected  expected
	}{
		{
			name: "успешное получение детали",
			args: args{
				uuid: partUUID,
			},
			setupMock: func(repo *mocks.Repository) {
				repo.EXPECT().
					Get(ctx, partUUID).
					Return(part, nil)
			},
			expected: expected{
				part: part,
				err:  nil,
			},
		},
		{
			name: "деталь не найдена",
			args: args{
				uuid: partUUID,
			},
			setupMock: func(repo *mocks.Repository) {
				repo.EXPECT().
					Get(ctx, partUUID).
					Return(model.Part{}, errs.ErrPartNotFound)
			},
			expected: expected{
				part: model.Part{},
				err:  errs.ErrPartNotFound,
			},
		},
		{
			name: "ошибка репозитория",
			args: args{
				uuid: partUUID,
			},
			setupMock: func(repo *mocks.Repository) {
				repo.EXPECT().
					Get(ctx, partUUID).
					Return(model.Part{}, errors.New("db error"))
			},
			expected: expected{
				part: model.Part{},
				err:  errors.New("db error"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := mocks.NewRepository(t)
			svc := partService.NewService(repo)

			tc.setupMock(repo)

			res, err := svc.Get(ctx, tc.args.uuid)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())
				assert.Equal(t, model.Part{}, res)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expected.part, res)
		})
	}
}
