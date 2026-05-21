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
	"github.com/vixart/rocket-factory/order/internal/model"
	orderv1 "github.com/vixart/rocket-factory/shared/pkg/openapi/order/v1"
)

func TestCreateOrder(t *testing.T) {
	t.Parallel()

	type args struct {
		req *orderv1.CreateOrderRequest
	}

	type expected struct {
		err error
	}

	var (
		ctx = context.Background()

		orderUUID  = uuid.New()
		hullUUID   = uuid.New()
		engineUUID = uuid.New()
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
					HullUUID:   hullUUID,
					EngineUUID: engineUUID,
					ShieldUUID: orderv1.NewOptNilUUID(shieldUUID),
					WeaponUUID: orderv1.NewOptNilUUID(weaponUUID),
				},
			},
			setupMock: func(service *mocks.OrderService) {
				service.EXPECT().
					Create(ctx, model.OrderParts{
						HullUUID:   hullUUID,
						EngineUUID: engineUUID,
						ShieldUUID: &shieldUUID,
						WeaponUUID: &weaponUUID,
					}).
					Return(&model.Order{
						OrderUUID:  orderUUID,
						TotalPrice: 205000,
					}, nil)
			},
		},
		{
			name: "ошибка создания заказа",
			args: args{
				req: &orderv1.CreateOrderRequest{
					HullUUID:   hullUUID,
					EngineUUID: engineUUID,
				},
			},
			setupMock: func(service *mocks.OrderService) {
				service.EXPECT().
					Create(ctx, model.OrderParts{
						HullUUID:   hullUUID,
						EngineUUID: engineUUID,
					}).
					Return(nil, internalErr)
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

			res, err := api.CreateOrder(ctx, tc.args.req)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())
				assert.Nil(t, res)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)

			response, ok := res.(*orderv1.CreateOrderResponse)
			require.True(t, ok)

			assert.Equal(t, orderUUID, response.OrderUUID)
			assert.Equal(t, int64(205000), response.TotalPrice)
		})
	}
}
