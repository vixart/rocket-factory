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

func TestReleaseParts(t *testing.T) {
	t.Parallel()

	type args struct {
		req *inventoryv1.ReleasePartsRequest
	}

	type expected struct {
		err error
	}

	var (
		ctx     = context.Background()
		uuid1   = uuid.New()
		uuid2   = uuid.New()
		repoErr = errors.New("db error")
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(svc *mocks.InventoryService)
		expected  expected
	}{
		{
			name: "parts released successfully",
			args: args{
				req: &inventoryv1.ReleasePartsRequest{
					Uuids: []string{
						uuid1.String(),
						uuid2.String(),
					},
				},
			},
			setupMock: func(svc *mocks.InventoryService) {
				svc.EXPECT().
					Release(
						ctx,
						[]uuid.UUID{uuid1, uuid2},
					).
					Return(nil)
			},
		},
		{
			name: "invalid uuid",
			args: args{
				req: &inventoryv1.ReleasePartsRequest{
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
				req: &inventoryv1.ReleasePartsRequest{
					Uuids: []string{
						uuid1.String(),
					},
				},
			},
			setupMock: func(svc *mocks.InventoryService) {
				svc.EXPECT().
					Release(
						ctx,
						[]uuid.UUID{uuid1},
					).
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

			res, err := api.ReleaseParts(ctx, tc.args.req)

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
