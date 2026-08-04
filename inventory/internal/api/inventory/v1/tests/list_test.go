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

func TestListParts(t *testing.T) {
	t.Parallel()

	type args struct {
		req *inventoryv1.ListPartsRequest
	}

	type expected struct {
		err error
	}

	var (
		ctx   = context.Background()
		uuid1 = uuid.New()
		uuid2 = uuid.New()

		part1 = entity.RestorePart(
			uuid1,
			"Engine X",
			"engine description",
			valueobject.PartTypeEngine,
			5000,
			3,
			0,
			valueobject.PartProperties{},
			time.Now(),
		)

		part2 = entity.RestorePart(
			uuid2,
			"Engine Y",
			"another engine",
			valueobject.PartTypeEngine,
			7000,
			5,
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
			name: "part list fetched successfully",
			args: args{
				req: &inventoryv1.ListPartsRequest{
					Uuids: []string{
						uuid1.String(),
						uuid2.String(),
					},
					PartType: inventoryv1.PartType_PART_TYPE_ENGINE,
				},
			},
			setupMock: func(svc *mocks.InventoryService) {
				svc.EXPECT().
					List(
						ctx,
						[]uuid.UUID{uuid1, uuid2},
						valueobject.PartTypeEngine,
					).
					Return([]entity.Part{part1, part2}, nil)
			},
		},
		{
			name: "invalid uuid",
			args: args{
				req: &inventoryv1.ListPartsRequest{
					Uuids: []string{
						"not-a-uuid",
					},
				},
			},
			setupMock: func(svc *mocks.InventoryService) {},
			expected: expected{
				err: errs.ErrInvalidUUID,
			},
		},
		{
			name: "service fails",
			args: args{
				req: &inventoryv1.ListPartsRequest{
					Uuids: []string{
						uuid1.String(),
					},
					PartType: inventoryv1.PartType_PART_TYPE_ENGINE,
				},
			},
			setupMock: func(svc *mocks.InventoryService) {
				svc.EXPECT().
					List(
						ctx,
						[]uuid.UUID{uuid1},
						valueobject.PartTypeEngine,
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

			assert.Equal(t, part1.UUID().String(), res.Parts[0].Uuid)
			assert.Equal(t, part1.Name(), res.Parts[0].Name)
			assert.Equal(t, part1.Description(), res.Parts[0].Description)
			assert.Equal(t, part1.Price(), res.Parts[0].Price)
			assert.Equal(t, int64(part1.StockQuantity()), res.Parts[0].StockQuantity)

			assert.Equal(t, part2.UUID().String(), res.Parts[1].Uuid)
			assert.Equal(t, part2.Name(), res.Parts[1].Name)
			assert.Equal(t, part2.Description(), res.Parts[1].Description)
			assert.Equal(t, part2.Price(), res.Parts[1].Price)
			assert.Equal(t, int64(part2.StockQuantity()), res.Parts[1].StockQuantity)
		})
	}
}
