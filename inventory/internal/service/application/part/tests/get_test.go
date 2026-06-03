package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model/entity"
	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
	partService "github.com/vixart/rocket-factory/inventory/internal/service/application/part"
	"github.com/vixart/rocket-factory/inventory/internal/service/application/part/mocks"
)

func TestGet(t *testing.T) {
	t.Parallel()

	type args struct {
		uuid uuid.UUID
	}

	type expected struct {
		part entity.Part
		err  error
	}

	ctx := context.Background()

	partUUID := uuid.New()

	properties, _ := valueobject.NewEngineProperties(
		valueobject.EngineClassA,
		100,
	)

	part := entity.RestorePart(
		partUUID,
		"Engine",
		"desc",
		valueobject.PartTypeEngine,
		5000,
		3,
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
			name: "успешное получение детали",
			args: args{
				uuid: partUUID,
			},
			setupMock: func(repo *mocks.Repository, checker *mocks.CompatibilityChecker) {
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
			setupMock: func(repo *mocks.Repository, checker *mocks.CompatibilityChecker) {
				repo.EXPECT().
					Get(ctx, partUUID).
					Return(entity.Part{}, errs.ErrPartNotFound)
			},
			expected: expected{
				part: entity.Part{},
				err:  errs.ErrPartNotFound,
			},
		},
		{
			name: "ошибка репозитория",
			args: args{
				uuid: partUUID,
			},
			setupMock: func(repo *mocks.Repository, checker *mocks.CompatibilityChecker) {
				repo.EXPECT().
					Get(ctx, partUUID).
					Return(entity.Part{}, errors.New("db error"))
			},
			expected: expected{
				part: entity.Part{},
				err:  errors.New("db error"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := mocks.NewRepository(t)
			checker := mocks.NewCompatibilityChecker(t)

			svc := partService.NewService(repo, checker)

			tc.setupMock(repo, checker)

			res, err := svc.Get(ctx, tc.args.uuid)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())
				assert.Equal(t, entity.Part{}, res)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expected.part, res)
		})
	}
}
