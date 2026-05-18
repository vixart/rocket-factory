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

func TestCreateOrder(t *testing.T) {
	t.Parallel()

	type args struct {
		req *orderv1.CreateOrderRequest
	}

	type expected struct {
		resType any
	}

	var (
		ctx = context.Background()

		orderUUID  = uuid.New()
		engineUUID = uuid.New()
		hullUUID   = uuid.New()
		shieldUUID = uuid.New()
		weaponUUID = uuid.New()

		internalErr = errors.New("internal error")
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(service *mocks.OrderService)
		expected  expected
	}{
		{
			name: "успешное создание заказа",
			args: args{
				req: &orderv1.CreateOrderRequest{
					EngineUUID: engineUUID,
					HullUUID:   hullUUID,
				},
			},
			setupMock: func(service *mocks.OrderService) {
				service.EXPECT().
					Create(ctx, model.OrderParts{
						EngineUUID: engineUUID,
						HullUUID:   hullUUID,
					}).
					Return(&model.Order{
						OrderUUID:  orderUUID,
						TotalPrice: 150000,
					}, nil)
			},
			expected: expected{
				resType: &orderv1.CreateOrderResponse{},
			},
		},
		{
			name: "успешное создание заказа со всеми деталями",
			args: args{
				req: &orderv1.CreateOrderRequest{
					EngineUUID: engineUUID,
					HullUUID:   hullUUID,
					ShieldUUID: orderv1.NewOptNilUUID(shieldUUID),
					WeaponUUID: orderv1.NewOptNilUUID(weaponUUID),
				},
			},
			setupMock: func(service *mocks.OrderService) {
				service.EXPECT().
					Create(ctx, model.OrderParts{
						EngineUUID: engineUUID,
						HullUUID:   hullUUID,
						ShieldUUID: &shieldUUID,
						WeaponUUID: &weaponUUID,
					}).
					Return(&model.Order{
						OrderUUID:  orderUUID,
						TotalPrice: 205000,
					}, nil)
			},
			expected: expected{
				resType: &orderv1.CreateOrderResponse{},
			},
		},
		{
			name: "деталь не найдена",
			args: args{
				req: &orderv1.CreateOrderRequest{
					EngineUUID: engineUUID,
					HullUUID:   hullUUID,
				},
			},
			setupMock: func(service *mocks.OrderService) {
				service.EXPECT().
					Create(ctx, model.OrderParts{
						EngineUUID: engineUUID,
						HullUUID:   hullUUID,
					}).
					Return(nil, errs.ErrPartNotFound)
			},
			expected: expected{
				resType: &orderv1.CreateOrderNotFound{},
			},
		},
		{
			name: "деталь не найдена в inventory",
			args: args{
				req: &orderv1.CreateOrderRequest{
					EngineUUID: engineUUID,
					HullUUID:   hullUUID,
				},
			},
			setupMock: func(service *mocks.OrderService) {
				service.EXPECT().
					Create(ctx, model.OrderParts{
						EngineUUID: engineUUID,
						HullUUID:   hullUUID,
					}).
					Return(nil, errs.ErrInventoryPartNotFound)
			},
			expected: expected{
				resType: &orderv1.CreateOrderNotFound{},
			},
		},
		{
			name: "невалидный uuid",
			args: args{
				req: &orderv1.CreateOrderRequest{
					EngineUUID: engineUUID,
					HullUUID:   hullUUID,
				},
			},
			setupMock: func(service *mocks.OrderService) {
				service.EXPECT().
					Create(ctx, model.OrderParts{
						EngineUUID: engineUUID,
						HullUUID:   hullUUID,
					}).
					Return(nil, errs.ErrInvalidUUID)
			},
			expected: expected{
				resType: &orderv1.CreateOrderBadRequest{},
			},
		},
		{
			name: "заказ уже существует",
			args: args{
				req: &orderv1.CreateOrderRequest{
					EngineUUID: engineUUID,
					HullUUID:   hullUUID,
				},
			},
			setupMock: func(service *mocks.OrderService) {
				service.EXPECT().
					Create(ctx, model.OrderParts{
						EngineUUID: engineUUID,
						HullUUID:   hullUUID,
					}).
					Return(nil, errs.ErrOrderAlreadyExists)
			},
			expected: expected{
				resType: &orderv1.CreateOrderBadRequest{},
			},
		},
		{
			name: "деталь отсутствует на складе",
			args: args{
				req: &orderv1.CreateOrderRequest{
					EngineUUID: engineUUID,
					HullUUID:   hullUUID,
				},
			},
			setupMock: func(service *mocks.OrderService) {
				service.EXPECT().
					Create(ctx, model.OrderParts{
						EngineUUID: engineUUID,
						HullUUID:   hullUUID,
					}).
					Return(nil, errs.ErrPartInsufficientStock)
			},
			expected: expected{
				resType: &orderv1.CreateOrderConflict{},
			},
		},
		{
			name: "внутренняя ошибка",
			args: args{
				req: &orderv1.CreateOrderRequest{
					EngineUUID: engineUUID,
					HullUUID:   hullUUID,
				},
			},
			setupMock: func(service *mocks.OrderService) {
				service.EXPECT().
					Create(ctx, model.OrderParts{
						EngineUUID: engineUUID,
						HullUUID:   hullUUID,
					}).
					Return(nil, internalErr)
			},
			expected: expected{
				resType: &orderv1.CreateOrderInternalServerError{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			orderService := mocks.NewOrderService(t)

			tc.setupMock(orderService)

			api := NewApi(orderService)

			res, err := api.CreateOrder(ctx, tc.args.req)

			require.NoError(t, err)
			require.NotNil(t, res)

			assert.IsType(t, tc.expected.resType, res)

			switch response := res.(type) {
			case *orderv1.CreateOrderResponse:
				assert.Equal(t, orderUUID, response.OrderUUID)
				assert.Positive(t, response.TotalPrice)

			case *orderv1.CreateOrderNotFound:
				assert.Equal(t, 404, response.Code)

			case *orderv1.CreateOrderBadRequest:
				assert.Equal(t, 400, response.Code)

			case *orderv1.CreateOrderConflict:
				assert.Equal(t, 409, response.Code)

			case *orderv1.CreateOrderInternalServerError:
				assert.Equal(t, 500, response.Code)
			}
		})
	}
}
