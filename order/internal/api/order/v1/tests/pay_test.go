package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/vixart/rocket-factory/order/internal/api/order/v1"
	"github.com/vixart/rocket-factory/order/internal/api/order/v1/mocks"
	"github.com/vixart/rocket-factory/order/internal/model"
	orderv1 "github.com/vixart/rocket-factory/shared/pkg/openapi/order/v1"
)

func TestPayOrder(t *testing.T) {
	t.Parallel()

	type args struct {
		req    *orderv1.PayOrderRequest
		params orderv1.PayOrderParams
	}

	type expected struct {
		err error
	}

	var (
		ctx = context.Background()

		orderUUID       = uuid.New()
		transactionUUID = uuid.New()

		internalErr = errors.New("internal error")
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(service *mocks.OrderService)
		expected  expected
	}{
		{
			name: "order paid successfully",
			args: args{
				req: &orderv1.PayOrderRequest{
					PaymentMethod: orderv1.PaymentMethodCARD,
				},
				params: orderv1.PayOrderParams{
					OrderUUID: orderUUID,
				},
			},
			setupMock: func(service *mocks.OrderService) {
				service.EXPECT().
					Pay(ctx, orderUUID, model.PaymentMethodCard).
					Return(transactionUUID, nil)
			},
		},
		{
			name: "order payment fails",
			args: args{
				req: &orderv1.PayOrderRequest{
					PaymentMethod: orderv1.PaymentMethodCARD,
				},
				params: orderv1.PayOrderParams{
					OrderUUID: orderUUID,
				},
			},
			setupMock: func(service *mocks.OrderService) {
				service.EXPECT().
					Pay(ctx, orderUUID, model.PaymentMethodCard).
					Return(uuid.UUID{}, internalErr)
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

			res, err := api.PayOrder(ctx, tc.args.req, tc.args.params)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())
				assert.Nil(t, res)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)

			response, ok := res.(*orderv1.PayOrderResponse)
			require.True(t, ok)

			assert.Equal(t, transactionUUID, response.TransactionUUID)
		})
	}
}
