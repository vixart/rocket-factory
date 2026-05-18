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
		resType any
	}

	var (
		ctx = context.Background()

		orderUUID = uuid.New()
		txUUID    = uuid.New()

		internalErr = errors.New("internal error")
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(service *mocks.OrderService)
		expected  expected
	}{
		{
			name: "успешная оплата заказа",
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
					Return(&txUUID, nil)
			},
			expected: expected{
				resType: &orderv1.PayOrderResponse{},
			},
		},
		{
			name: "заказ не найден",
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
					Return(nil, errs.ErrOrderNotFound)
			},
			expected: expected{
				resType: &orderv1.PayOrderNotFound{},
			},
		},
		{
			name: "невалидный метод оплаты",
			args: args{
				req: &orderv1.PayOrderRequest{
					PaymentMethod: "InvalidPayMethod",
				},
				params: orderv1.PayOrderParams{
					OrderUUID: orderUUID,
				},
			},
			setupMock: func(service *mocks.OrderService) {
				service.EXPECT().
					Pay(ctx, orderUUID, model.PaymentMethodUnspecified).
					Return(nil, errs.ErrInvalidPaymentMethod)
			},
			expected: expected{
				resType: &orderv1.PayOrderBadRequest{},
			},
		},
		{
			name: "недопустимый статус заказа",
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
					Return(nil, errs.ErrInvalidOrderStatus)
			},
			expected: expected{
				resType: &orderv1.PayOrderConflict{},
			},
		},
		{
			name: "ошибка платежного сервиса",
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
					Return(nil, errs.ErrPaymentFailed)
			},
			expected: expected{
				resType: &orderv1.PayOrderInternalServerError{},
			},
		},
		{
			name: "неизвестная ошибка",
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
					Return(nil, internalErr)
			},
			expected: expected{
				resType: &orderv1.PayOrderInternalServerError{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			orderService := mocks.NewOrderService(t)

			tc.setupMock(orderService)

			api := NewApi(orderService)

			res, err := api.PayOrder(ctx, tc.args.req, tc.args.params)

			require.NoError(t, err)
			require.NotNil(t, res)

			assert.IsType(t, tc.expected.resType, res)

			switch response := res.(type) {
			case *orderv1.PayOrderResponse:
				assert.Equal(t, txUUID, response.TransactionUUID)

			case *orderv1.PayOrderNotFound:
				assert.Equal(t, 404, response.Code)

			case *orderv1.PayOrderBadRequest:
				assert.Equal(t, 400, response.Code)

			case *orderv1.PayOrderConflict:
				assert.Equal(t, 409, response.Code)

			case *orderv1.PayOrderInternalServerError:
				assert.Equal(t, 500, response.Code)
			}
		})
	}
}
