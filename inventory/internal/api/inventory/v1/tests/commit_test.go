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
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
)

func TestCommitParts(t *testing.T) {
	t.Parallel()

	type args struct {
		req *inventoryv1.CommitPartsRequest
	}

	type expected struct {
		err error
	}

	var (
		ctx = context.Background()

		uuid1 = uuid.New()
		uuid2 = uuid.New()

		repoErr = errors.New("service error")
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(svc *mocks.InventoryService)
		expected  expected
	}{
		{
			name: "успешный commit",
			args: args{
				req: &inventoryv1.CommitPartsRequest{
					Uuids: []string{
						uuid1.String(),
						uuid2.String(),
					},
				},
			},
			setupMock: func(svc *mocks.InventoryService) {
				svc.EXPECT().
					Commit(ctx, []uuid.UUID{uuid1, uuid2}).
					Return(nil)
			},
		},
		{
			name: "пустой список uuid",
			args: args{
				req: &inventoryv1.CommitPartsRequest{
					Uuids: []string{},
				},
			},
			setupMock: func(svc *mocks.InventoryService) {
				svc.EXPECT().
					Commit(ctx, []uuid.UUID{}).
					Return(nil)
			},
		},
		{
			name: "невалидный uuid в списке",
			args: args{
				req: &inventoryv1.CommitPartsRequest{
					Uuids: []string{
						uuid1.String(),
						"invalid-uuid",
					},
				},
			},
			setupMock: func(_ *mocks.InventoryService) {},
			expected: expected{
				err: errs.ErrInvalidUUID,
			},
		},
		{
			name: "ошибка сервиса commit",
			args: args{
				req: &inventoryv1.CommitPartsRequest{
					Uuids: []string{
						uuid1.String(),
						uuid2.String(),
					},
				},
			},
			setupMock: func(svc *mocks.InventoryService) {
				svc.EXPECT().
					Commit(ctx, []uuid.UUID{uuid1, uuid2}).
					Return(repoErr)
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

			res, err := api.CommitParts(ctx, tc.args.req)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())
				assert.Nil(t, res)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)
		})
	}
}
