package v1

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vixart/rocket-factory/order/internal/api/order/v1/mocks"
	errs "github.com/vixart/rocket-factory/order/internal/errors"
	orderv1 "github.com/vixart/rocket-factory/shared/pkg/openapi/order/v1"
)

func TestCancelOrder(t *testing.T) {
	t.Parallel()

	type args struct {
		params orderv1.CancelOrderParams
	}

	type expected struct {
		resType any
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
			name: "успешная отмена заказа",
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
			expected: expected{
				resType: &orderv1.CancelOrderResponse{},
			},
		},
		{
			name: "заказ не найден",
			args: args{
				params: orderv1.CancelOrderParams{
					OrderUUID: orderUUID,
				},
			},
			setupMock: func(service *mocks.OrderService) {
				service.EXPECT().
					Cancel(ctx, orderUUID).
					Return(errs.ErrOrderNotFound)
			},
			expected: expected{
				resType: &orderv1.CancelOrderNotFound{},
			},
		},
		{
			name: "неверный статус заказа",
			args: args{
				params: orderv1.CancelOrderParams{
					OrderUUID: orderUUID,
				},
			},
			setupMock: func(service *mocks.OrderService) {
				service.EXPECT().
					Cancel(ctx, orderUUID).
					Return(errs.ErrInvalidOrderStatus)
			},
			expected: expected{
				resType: &orderv1.CancelOrderConflict{},
			},
		},
		{
			name: "внутренняя ошибка",
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
				resType: &orderv1.CancelOrderInternalServerError{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			orderService := mocks.NewOrderService(t)

			tc.setupMock(orderService)

			api := NewApi(orderService)

			res, err := api.CancelOrder(ctx, tc.args.params)

			require.NoError(t, err)
			require.NotNil(t, res)

			assert.IsType(t, tc.expected.resType, res)

			switch response := res.(type) {
			case *orderv1.CancelOrderResponse:
				assert.NotNil(t, response)

			case *orderv1.CancelOrderNotFound:
				assert.Equal(t, 404, response.Code)

			case *orderv1.CancelOrderConflict:
				assert.Equal(t, 409, response.Code)

			case *orderv1.CancelOrderInternalServerError:
				assert.Equal(t, 500, response.Code)
			}
		})
	}
}
