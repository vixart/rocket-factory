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

func TestListParts(t *testing.T) {
	t.Parallel()

	type args struct {
		req *inventoryv1.ListPartsRequest
	}

	type expected struct {
		err error
	}

	var (
		ctx = context.Background()

		partUUID1 = uuid.New()
		partUUID2 = uuid.New()

		parts = []model.Part{
			{
				UUID:          partUUID1,
				Name:          "Engine X",
				Description:   "engine",
				Price:         5000,
				PartType:      model.PartTypeEngine,
				StockQuantity: 3,
			},
			{
				UUID:          partUUID2,
				Name:          "Hull Z",
				Description:   "hull",
				Price:         10000,
				PartType:      model.PartTypeHull,
				StockQuantity: 5,
			},
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
			name: "успешное получение списка деталей",
			args: args{
				req: &inventoryv1.ListPartsRequest{
					Uuids: []string{
						partUUID1.String(),
						partUUID2.String(),
					},
					PartType: inventoryv1.PartType_PART_TYPE_UNSPECIFIED,
				},
			},
			setupMock: func(svc *mocks.InventoryService) {
				svc.EXPECT().
					List(
						ctx,
						[]uuid.UUID{partUUID1, partUUID2},
						model.PartTypeUnspecified,
					).
					Return(parts, nil)
			},
		},
		{
			name: "невалидный uuid",
			args: args{
				req: &inventoryv1.ListPartsRequest{
					Uuids: []string{
						"invalid-uuid",
					},
				},
			},
			setupMock: func(svc *mocks.InventoryService) {},
			expected: expected{
				err: errs.ErrInvalidUUID,
			},
		},
		{
			name: "ошибка сервиса",
			args: args{
				req: &inventoryv1.ListPartsRequest{
					Uuids: []string{
						partUUID1.String(),
					},
				},
			},
			setupMock: func(svc *mocks.InventoryService) {
				svc.EXPECT().
					List(
						ctx,
						[]uuid.UUID{partUUID1},
						model.PartTypeUnspecified,
					).
					Return(nil, repoErr)
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

			res, err := api.ListParts(ctx, tc.args.req)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())
				assert.Nil(t, res)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)

			require.Len(t, res.Parts, 2)

			assert.Equal(t, partUUID1.String(), res.Parts[0].Uuid)
			assert.Equal(t, "Engine X", res.Parts[0].Name)

			assert.Equal(t, partUUID2.String(), res.Parts[1].Uuid)
			assert.Equal(t, "Hull Z", res.Parts[1].Name)
		})
	}
}
