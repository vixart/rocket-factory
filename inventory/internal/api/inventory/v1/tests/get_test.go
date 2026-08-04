package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vixart/rocket-factory/inventory/internal/api/inventory/v1"
	"github.com/vixart/rocket-factory/inventory/internal/api/inventory/v1/mocks"
	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model/entity"
	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
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
		ctx       = context.Background()
		validUUID = uuid.New()

		validPart = entity.RestorePart(
			validUUID,
			"Engine X",
			"desc",
			valueobject.PartTypeEngine,
			5000,
			3,
			0,
			valueobject.PartProperties{},
			time.Now(),
		)

		repoErr = errors.New("db error")
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(svc *mocks.InventoryService)
		expected  expected
	}{
		{
			name: "part fetched successfully",
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
			name: "empty uuid",
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
			name: "invalid uuid",
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
			name: "part not found",
			args: args{
				req: &inventoryv1.GetPartRequest{
					Uuid: validUUID.String(),
				},
			},
			setupMock: func(svc *mocks.InventoryService) {
				svc.EXPECT().
					Get(ctx, validUUID).
					Return(entity.Part{}, errs.ErrPartNotFound)
			},
			expected: expected{
				err: errs.ErrPartNotFound,
			},
		},
		{
			name: "internal service error",
			args: args{
				req: &inventoryv1.GetPartRequest{
					Uuid: validUUID.String(),
				},
			},
			setupMock: func(svc *mocks.InventoryService) {
				svc.EXPECT().
					Get(ctx, validUUID).
					Return(entity.Part{}, repoErr)
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
			require.NotNil(t, res.Part)

			assert.Equal(t, validPart.UUID().String(), res.Part.Uuid)
			assert.Equal(t, validPart.Name(), res.Part.Name)
			assert.Equal(t, validPart.Description(), res.Part.Description)
			assert.Equal(t, validPart.Price(), res.Part.Price)
			assert.Equal(t, int64(validPart.StockQuantity()), res.Part.StockQuantity)
		})
	}
}
