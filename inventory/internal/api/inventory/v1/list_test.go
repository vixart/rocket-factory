package v1

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vixart/rocket-factory/inventory/internal/api/inventory/v1/mocks"
	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
)

func TestListParts(t *testing.T) {
	t.Parallel()

	type args struct {
		req *inventoryv1.ListPartsRequest
	}

	type expected struct {
		errCode  codes.Code
		hasError bool
	}

	ctx := context.Background()

	hullUUID := uuid.New()
	engineUUID := uuid.New()

	hullPart := model.Part{
		UUID:          hullUUID,
		Name:          "Hull",
		Price:         100000,
		StockQuantity: 10,
	}

	enginePart := model.Part{
		UUID:          engineUUID,
		Name:          "Engine",
		Price:         50000,
		StockQuantity: 5,
	}

	tests := []struct {
		name      string
		args      args
		setupMock func(svc *mocks.InventoryService)
		expected  expected
	}{
		{
			name: "успешное получение списка деталей",
			args: args{
				req: &inventoryv1.ListPartsRequest{
					Uuids:    []string{hullUUID.String(), engineUUID.String()},
					PartType: inventoryv1.PartType_PART_TYPE_HULL,
				},
			},
			setupMock: func(svc *mocks.InventoryService) {
				svc.EXPECT().
					List(ctx, []uuid.UUID{hullUUID, engineUUID}, model.PartTypeHull).
					Return([]model.Part{hullPart, enginePart}, nil)
			},
			expected: expected{},
		},
		{
			name: "пустой список uuid",
			args: args{
				req: &inventoryv1.ListPartsRequest{
					Uuids:    []string{},
					PartType: inventoryv1.PartType_PART_TYPE_HULL,
				},
			},
			setupMock: func(svc *mocks.InventoryService) {
				svc.EXPECT().
					List(ctx, []uuid.UUID{}, model.PartTypeHull).
					Return([]model.Part{}, nil)
			},
			expected: expected{},
		},
		{
			name: "неверный формат uuid",
			args: args{
				req: &inventoryv1.ListPartsRequest{
					Uuids:    []string{"not-a-uuid"},
					PartType: inventoryv1.PartType_PART_TYPE_HULL,
				},
			},
			setupMock: func(svc *mocks.InventoryService) {
				// не должен вызываться сервис вообще
			},
			expected: expected{
				hasError: true,
				errCode:  codes.InvalidArgument,
			},
		},
		{
			name: "деталь не найдена",
			args: args{
				req: &inventoryv1.ListPartsRequest{
					Uuids:    []string{hullUUID.String()},
					PartType: inventoryv1.PartType_PART_TYPE_HULL,
				},
			},
			setupMock: func(svc *mocks.InventoryService) {
				svc.EXPECT().
					List(ctx, []uuid.UUID{hullUUID}, model.PartTypeHull).
					Return(nil, errs.ErrPartNotFound)
			},
			expected: expected{
				hasError: true,
				errCode:  codes.NotFound,
			},
		},
		{
			name: "внутренняя ошибка сервиса",
			args: args{
				req: &inventoryv1.ListPartsRequest{
					Uuids:    []string{hullUUID.String()},
					PartType: inventoryv1.PartType_PART_TYPE_HULL,
				},
			},
			setupMock: func(svc *mocks.InventoryService) {
				svc.EXPECT().
					List(ctx, []uuid.UUID{hullUUID}, model.PartTypeHull).
					Return(nil, errors.New("db error"))
			},
			expected: expected{
				hasError: true,
				errCode:  codes.Internal,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := mocks.NewInventoryService(t)
			api := &api{
				inventoryService: svc,
			}

			tc.setupMock(svc)

			res, err := api.ListParts(ctx, tc.args.req)

			if tc.expected.hasError {
				require.Error(t, err)

				st, ok := status.FromError(err)
				require.True(t, ok)

				assert.Equal(t, tc.expected.errCode, st.Code())
				assert.Nil(t, res)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)
		})
	}
}
