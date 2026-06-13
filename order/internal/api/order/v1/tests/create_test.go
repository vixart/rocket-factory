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
	"github.com/vixart/rocket-factory/order/internal/service/input"
	orderv1 "github.com/vixart/rocket-factory/shared/pkg/openapi/order/v1"
)

func TestCreateOrder(t *testing.T) {
	t.Parallel()

	var (
		ctx = context.Background()

		userUUID   = uuid.New()
		orderUUID  = uuid.New()
		hullUUID   = uuid.New()
		engineUUID = uuid.New()
		shieldUUID = uuid.New()
		weaponUUID = uuid.New()

		internalErr = errors.New("internal error")
	)

	tests := []struct {
		name      string
		req       *orderv1.CreateOrderRequest
		setupMock func(service *mocks.OrderService)
		check     func(t *testing.T, res orderv1.CreateOrderRes, err error)
	}{
		{
			name: "успешное создание заказа",
			req: &orderv1.CreateOrderRequest{
				UserUUID:   userUUID,
				HullUUID:   hullUUID,
				EngineUUID: engineUUID,
				ShieldUUID: orderv1.NewOptNilUUID(shieldUUID),
				WeaponUUID: orderv1.NewOptNilUUID(weaponUUID),
			},
			setupMock: func(service *mocks.OrderService) {
				service.EXPECT().
					Create(
						ctx,
						input.OrderParts{
							HullUUID:   hullUUID,
							EngineUUID: engineUUID,
							ShieldUUID: &shieldUUID,
							WeaponUUID: &weaponUUID,
						},
						userUUID,
					).
					Return(&model.Order{
						UUID: orderUUID,
						Items: []model.OrderItem{
							{
								UUID:  hullUUID,
								Price: 150000,
							},
							{
								UUID:  engineUUID,
								Price: 55000,
							},
						},
					}, nil)
			},
			check: func(t *testing.T, res orderv1.CreateOrderRes, err error) {
				t.Helper()

				require.NoError(t, err)

				response, ok := res.(*orderv1.CreateOrderResponse)
				require.True(t, ok)

				assert.Equal(t, &orderv1.CreateOrderResponse{
					OrderUUID:  orderUUID,
					TotalPrice: 205000,
				}, response)
			},
		},
		{
			name: "ошибка создания заказа",
			req: &orderv1.CreateOrderRequest{
				UserUUID:   userUUID,
				HullUUID:   hullUUID,
				EngineUUID: engineUUID,
				ShieldUUID: orderv1.OptNilUUID{},
				WeaponUUID: orderv1.OptNilUUID{},
			},
			setupMock: func(service *mocks.OrderService) {
				service.EXPECT().
					Create(
						ctx,
						input.OrderParts{
							HullUUID:   hullUUID,
							EngineUUID: engineUUID,
						},
						userUUID,
					).
					Return(nil, internalErr)
			},
			check: func(t *testing.T, res orderv1.CreateOrderRes, err error) {
				t.Helper()

				require.Error(t, err)
				assert.ErrorContains(t, err, internalErr.Error())
				assert.Nil(t, res)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			orderService := mocks.NewOrderService(t)

			tc.setupMock(orderService)

			api := v1.NewApi(orderService)

			res, err := api.CreateOrder(ctx, tc.req)

			tc.check(t, res, err)
		})
	}
}
