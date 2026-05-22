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
	"github.com/vixart/rocket-factory/inventory/internal/model"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
)

func TestGetPart(t *testing.T) {
	t.Parallel()

	type args struct {
		req *inventoryv1.GetPartRequest
	}

	type expected struct {
		err error
	}

	var (
		ctx = context.Background()

		validUUID = uuid.New()

		validPart = model.Part{
			UUID:          validUUID,
			Name:          "Engine X",
			Description:   "desc",
			Price:         5000,
			PartType:      model.PartTypeEngine,
			StockQuantity: 3,
		}

		repoErr = errors.New("db error")
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(svc *mocks.InventoryService)
		expected  expected
	}{
		{
			name: "успешное получение детали",
			args: args{
				req: &inventoryv1.GetPartRequest{
					Uuid: validUUID.String(),
				},
			},
			setupMock: func(svc *mocks.InventoryService) {
				svc.EXPECT().
					Get(ctx, validUUID).
					Return(validPart, nil)
			},
		},
		{
			name: "пустой uuid",
			args: args{
				req: &inventoryv1.GetPartRequest{
					Uuid: "",
				},
			},
			setupMock: func(svc *mocks.InventoryService) {},
			expected: expected{
				err: errs.ErrInvalidUUID,
			},
		},
		{
			name: "невалидный uuid",
			args: args{
				req: &inventoryv1.GetPartRequest{
					Uuid: "not-a-uuid",
				},
			},
			setupMock: func(svc *mocks.InventoryService) {},
			expected: expected{
				err: errs.ErrInvalidUUID,
			},
		},
		{
			name: "деталь не найдена",
			args: args{
				req: &inventoryv1.GetPartRequest{
					Uuid: validUUID.String(),
				},
			},
			setupMock: func(svc *mocks.InventoryService) {
				svc.EXPECT().
					Get(ctx, validUUID).
					Return(model.Part{}, errs.ErrPartNotFound)
			},
			expected: expected{
				err: errs.ErrPartNotFound,
			},
		},
		{
			name: "внутренняя ошибка сервиса",
			args: args{
				req: &inventoryv1.GetPartRequest{
					Uuid: validUUID.String(),
				},
			},
			setupMock: func(svc *mocks.InventoryService) {
				svc.EXPECT().
					Get(ctx, validUUID).
					Return(model.Part{}, repoErr)
			},
			expected: expected{
				err: repoErr,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockSvc := mocks.NewInventoryService(t)
			api := v1.NewApi(mockSvc)

			tc.setupMock(mockSvc)

			res, err := api.GetPart(ctx, tc.args.req)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())
				assert.Nil(t, res)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)

			assert.Equal(t, validUUID.String(), res.Part.Uuid)
			assert.Equal(t, validPart.Name, res.Part.Name)
			assert.Equal(t, validPart.Description, res.Part.Description)
			assert.Equal(t, validPart.Price, res.Part.Price)
			assert.Equal(t, validPart.StockQuantity, res.Part.StockQuantity)
		})
	}
}
