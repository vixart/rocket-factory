package v1

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vixart/rocket-factory/order/internal/api/order/v1/mocks"
	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
	orderv1 "github.com/vixart/rocket-factory/shared/pkg/openapi/order/v1"
)

func TestGetOrder(t *testing.T) {
	t.Parallel()

	type args struct {
		params orderv1.GetOrderParams
	}

	type expected struct {
		resType any
	}

	var (
		ctx = context.Background()

		orderUUID       = uuid.New()
		hullUUID        = uuid.New()
		engineUUID      = uuid.New()
		shieldUUID      = uuid.New()
		weaponUUID      = uuid.New()
		transactionUUID = uuid.New()

		createdAt = time.Now()

		internalErr = errors.New("internal error")

		paymentMethod = model.PaymentMethodCard
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(service *mocks.OrderService)
		expected  expected
	}{
		{
			name: "успешное получение заказа",
			args: args{
				params: orderv1.GetOrderParams{
					OrderUUID: orderUUID,
				},
			},
			setupMock: func(service *mocks.OrderService) {
				service.EXPECT().
					Get(ctx, orderUUID).
					Return(model.Order{
						OrderUUID:       orderUUID,
						HullUUID:        hullUUID,
						EngineUUID:      engineUUID,
						ShieldUUID:      &shieldUUID,
						WeaponUUID:      &weaponUUID,
						TotalPrice:      250000,
						TransactionUUID: &transactionUUID,
						PaymentMethod:   &paymentMethod,
						Status:          model.OrderStatusPaid,
						CreatedAt:       createdAt,
					}, nil)
			},
			expected: expected{
				resType: &orderv1.OrderDto{},
			},
		},
		{
			name: "заказ не найден",
			args: args{
				params: orderv1.GetOrderParams{
					OrderUUID: orderUUID,
				},
			},
			setupMock: func(service *mocks.OrderService) {
				service.EXPECT().
					Get(ctx, orderUUID).
					Return(model.Order{}, errs.ErrOrderNotFound)
			},
			expected: expected{
				resType: &orderv1.GetOrderNotFound{},
			},
		},
		{
			name: "внутренняя ошибка",
			args: args{
				params: orderv1.GetOrderParams{
					OrderUUID: orderUUID,
				},
			},
			setupMock: func(service *mocks.OrderService) {
				service.EXPECT().
					Get(ctx, orderUUID).
					Return(model.Order{}, internalErr)
			},
			expected: expected{
				resType: &orderv1.GetOrderInternalServerError{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			orderService := mocks.NewOrderService(t)

			tc.setupMock(orderService)

			api := NewApi(orderService)

			res, err := api.GetOrder(ctx, tc.args.params)

			require.NoError(t, err)
			require.NotNil(t, res)

			assert.IsType(t, tc.expected.resType, res)

			switch response := res.(type) {
			case *orderv1.OrderDto:
				assert.Equal(t, orderUUID, response.OrderUUID)
				assert.Equal(t, hullUUID, response.HullUUID)
				assert.Equal(t, engineUUID, response.EngineUUID)
				assert.Equal(t, shieldUUID, response.ShieldUUID.Value)
				assert.Equal(t, weaponUUID, response.WeaponUUID.Value)
				assert.Equal(t, transactionUUID, response.TransactionUUID.Value)
				assert.Equal(t, orderv1.PaymentMethodCARD, response.PaymentMethod.Value)
				assert.Equal(t, orderv1.OrderStatusPAID, response.Status)
				assert.Equal(t, int64(250000), response.TotalPrice)
				assert.Equal(t, createdAt, response.CreatedAt)

			case *orderv1.GetOrderNotFound:
				assert.Equal(t, 404, response.Code)
				assert.Equal(t, "заказ не найден", response.Message)

			case *orderv1.GetOrderInternalServerError:
				assert.Equal(t, 500, response.Code)
				assert.Equal(t, "что-то пошло не так", response.Message)
			}
		})
	}
}
