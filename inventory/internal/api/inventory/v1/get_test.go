package v1

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	var (
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
		check     func(t *testing.T, res *inventoryv1.GetPartResponse, err error)
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
					Get(context.Background(), validUUID).
					Return(validPart, nil)
			},
			check: func(t *testing.T, res *inventoryv1.GetPartResponse, err error) {
				require.NoError(t, err)
				require.NotNil(t, res)

				assert.Equal(t, validUUID.String(), res.Part.Uuid)
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
			check: func(t *testing.T, res *inventoryv1.GetPartResponse, err error) {
				require.Error(t, err)
				require.Nil(t, res)
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
			check: func(t *testing.T, res *inventoryv1.GetPartResponse, err error) {
				require.Error(t, err)
				require.Nil(t, res)
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
					Get(context.Background(), validUUID).
					Return(model.Part{}, errs.ErrPartNotFound)
			},
			check: func(t *testing.T, res *inventoryv1.GetPartResponse, err error) {
				require.Error(t, err)
				require.Nil(t, res)
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
					Get(context.Background(), validUUID).
					Return(model.Part{}, repoErr)
			},
			check: func(t *testing.T, res *inventoryv1.GetPartResponse, err error) {
				require.Error(t, err)
				require.Nil(t, res)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockSvc := mocks.NewInventoryService(t)
			api := NewApi(mockSvc)

			tc.setupMock(mockSvc)

			res, err := api.GetPart(context.Background(), tc.args.req)

			tc.check(t, res, err)
		})
	}
}
