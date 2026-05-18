package order

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
	"github.com/vixart/rocket-factory/order/internal/service/order/mocks"
)

func TestPay(t *testing.T) {
	t.Parallel()

	type args struct {
		orderUUID     uuid.UUID
		paymentMethod model.PaymentMethod
	}

	type expected struct {
		err error
	}

	ctx := context.Background()

	orderUUID := uuid.New()
	txUUID := uuid.New()

	validOrder := model.Order{
		OrderUUID: orderUUID,
		Status:    model.OrderStatusPendingPayment,
	}

	tests := []struct {
		name      string
		args      args
		setupMock func(
			orderRepo *mocks.Repository,
			paymentClient *mocks.PaymentClient,
		)
		expected expected
	}{
		{
			name: "не указан метод оплаты",
			args: args{
				orderUUID:     orderUUID,
				paymentMethod: model.PaymentMethodUnspecified,
			},
			setupMock: func(orderRepo *mocks.Repository, paymentClient *mocks.PaymentClient) {},
			expected: expected{
				err: errs.ErrInvalidPaymentMethod,
			},
		},
		{
			name: "заказ не найден",
			args: args{
				orderUUID:     orderUUID,
				paymentMethod: model.PaymentMethodCard,
			},
			setupMock: func(orderRepo *mocks.Repository, paymentClient *mocks.PaymentClient) {
				orderRepo.EXPECT().
					Get(ctx, orderUUID).
					Return(model.Order{}, errs.ErrOrderNotFound)
			},
			expected: expected{
				err: errs.ErrOrderNotFound,
			},
		},
		{
			name: "невалидный статус заказа",
			args: args{
				orderUUID:     orderUUID,
				paymentMethod: model.PaymentMethodCard,
			},
			setupMock: func(orderRepo *mocks.Repository, paymentClient *mocks.PaymentClient) {
				orderRepo.EXPECT().
					Get(ctx, orderUUID).
					Return(model.Order{
						OrderUUID: orderUUID,
						Status:    model.OrderStatusPaid,
					}, nil)
			},
			expected: expected{
				err: errs.ErrInvalidOrderStatus,
			},
		},
		{
			name: "ошибка платежного клиента",
			args: args{
				orderUUID:     orderUUID,
				paymentMethod: model.PaymentMethodCard,
			},
			setupMock: func(orderRepo *mocks.Repository, paymentClient *mocks.PaymentClient) {
				orderRepo.EXPECT().
					Get(ctx, orderUUID).
					Return(validOrder, nil)

				paymentClient.EXPECT().
					PayOrder(ctx, orderUUID, model.PaymentMethodCard).
					Return((*uuid.UUID)(nil), errs.ErrPaymentFailed)
			},
			expected: expected{
				err: errs.ErrPaymentFailed,
			},
		},
		{
			name: "успешная оплата",
			args: args{
				orderUUID:     orderUUID,
				paymentMethod: model.PaymentMethodCard,
			},
			setupMock: func(orderRepo *mocks.Repository, paymentClient *mocks.PaymentClient) {
				orderRepo.EXPECT().
					Get(ctx, orderUUID).
					Return(validOrder, nil)

				paymentClient.EXPECT().
					PayOrder(ctx, orderUUID, model.PaymentMethodCard).
					Return(&txUUID, nil)

				orderRepo.EXPECT().
					Update(ctx, mock.MatchedBy(func(o model.Order) bool {
						return o.OrderUUID == orderUUID &&
							o.Status == model.OrderStatusPaid &&
							o.PaymentMethod != nil &&
							*o.PaymentMethod == model.PaymentMethodCard &&
							o.TransactionUUID != nil &&
							*o.TransactionUUID == txUUID
					})).
					Return(nil)
			},
			expected: expected{
				err: nil,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			orderRepo := mocks.NewRepository(t)
			paymentClient := mocks.NewPaymentClient(t)

			tc.setupMock(orderRepo, paymentClient)

			svc := &service{
				orderRepository: orderRepo,
				paymentClient:   paymentClient,
			}

			res, err := svc.Pay(ctx, tc.args.orderUUID, tc.args.paymentMethod)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.expected.err)
				assert.Nil(t, res)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, res)
		})
	}
}
