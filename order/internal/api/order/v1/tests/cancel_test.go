package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vixart/rocket-factory/order/internal/api/order/v1"
	"github.com/vixart/rocket-factory/order/internal/api/order/v1/mocks"
	orderv1 "github.com/vixart/rocket-factory/shared/pkg/openapi/order/v1"
)

func TestCancelOrder(t *testing.T) {
	t.Parallel()

	type args struct {
		params orderv1.CancelOrderParams
	}

	type expected struct {
		err error
	}

	var (
		ctx = context.Background()

		orderUUID = uuid.New()

		internalErr = errors.New("internal error")
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(service *mocks.OrderService)
		expected  expected
	}{
		{
			name: "order cancelled successfully",
			args: args{
				params: orderv1.CancelOrderParams{
					OrderUUID: orderUUID,
				},
			},
			setupMock: func(service *mocks.OrderService) {
				service.EXPECT().
					Cancel(ctx, orderUUID).
					Return(nil)
			},
		},
		{
			name: "order cancellation fails",
			args: args{
				params: orderv1.CancelOrderParams{
					OrderUUID: orderUUID,
				},
			},
			setupMock: func(service *mocks.OrderService) {
				service.EXPECT().
					Cancel(ctx, orderUUID).
					Return(internalErr)
			},
			expected: expected{
				err: internalErr,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			orderService := mocks.NewOrderService(t)

			tc.setupMock(orderService)

			api := v1.NewApi(orderService)

			res, err := api.CancelOrder(ctx, tc.args.params)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())
				assert.Nil(t, res)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)

			assert.IsType(t, &orderv1.CancelOrderResponse{}, res)
		})
	}
}
