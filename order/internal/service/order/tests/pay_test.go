package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
	orderService "github.com/vixart/rocket-factory/order/internal/service/order"
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

	var (
		ctx = context.Background()

		orderUUID       = uuid.New()
		transactionUUID = uuid.New()
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(
			repo *mocks.Repository,
			payment *mocks.PaymentClient,
			tx *mocks.TxManager,
		)
		expected expected
	}{
		{
			name: "успешная оплата заказа",
			args: args{
				orderUUID:     orderUUID,
				paymentMethod: model.PaymentMethodCard,
			},
			setupMock: func(repo *mocks.Repository, payment *mocks.PaymentClient, tx *mocks.TxManager) {
				order := model.Order{
					UUID:   orderUUID,
					Status: model.OrderStatusPendingPayment,
				}

				tx.EXPECT().
					Do(ctx, mock.AnythingOfType("func(context.Context) error")).
					RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
						repo.EXPECT().
							Get(ctx, orderUUID).
							Return(order, nil)

						payment.EXPECT().
							PayOrder(ctx, orderUUID, model.PaymentMethodCard).
							Return(transactionUUID, nil)

						repo.EXPECT().
							Update(ctx, mock.MatchedBy(func(o model.Order) bool {
								return o.UUID == orderUUID &&
									o.Status == model.OrderStatusPaid &&
									o.TransactionUUID != nil &&
									o.PaymentMethod != nil
							})).
							Return(nil)

						return fn(ctx)
					})
			},
		},
		{
			name: "невалидный payment method",
			args: args{
				orderUUID:     orderUUID,
				paymentMethod: model.PaymentMethodUnspecified,
			},
			setupMock: func(repo *mocks.Repository, payment *mocks.PaymentClient, tx *mocks.TxManager) {
				// tx не должен вызываться
			},
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
			setupMock: func(repo *mocks.Repository, payment *mocks.PaymentClient, tx *mocks.TxManager) {
				tx.EXPECT().
					Do(ctx, mock.AnythingOfType("func(context.Context) error")).
					RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
						repo.EXPECT().
							Get(ctx, orderUUID).
							Return(model.Order{}, errs.ErrOrderNotFound)

						return fn(ctx)
					})
			},
			expected: expected{
				err: errs.ErrOrderNotFound,
			},
		},
		{
			name: "неверный статус заказа",
			args: args{
				orderUUID:     orderUUID,
				paymentMethod: model.PaymentMethodCard,
			},
			setupMock: func(repo *mocks.Repository, payment *mocks.PaymentClient, tx *mocks.TxManager) {
				tx.EXPECT().
					Do(ctx, mock.AnythingOfType("func(context.Context) error")).
					RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
						repo.EXPECT().
							Get(ctx, orderUUID).
							Return(model.Order{
								UUID:   orderUUID,
								Status: model.OrderStatusPaid,
							}, nil)

						return fn(ctx)
					})
			},
			expected: expected{
				err: errs.ErrInvalidOrderStatus,
			},
		},
		{
			name: "ошибка платежа",
			args: args{
				orderUUID:     orderUUID,
				paymentMethod: model.PaymentMethodCard,
			},
			setupMock: func(repo *mocks.Repository, payment *mocks.PaymentClient, tx *mocks.TxManager) {
				tx.EXPECT().
					Do(ctx, mock.AnythingOfType("func(context.Context) error")).
					RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
						repo.EXPECT().
							Get(ctx, orderUUID).
							Return(model.Order{
								UUID:   orderUUID,
								Status: model.OrderStatusPendingPayment,
							}, nil)

						payment.EXPECT().
							PayOrder(ctx, orderUUID, model.PaymentMethodCard).
							Return(uuid.Nil, errors.New("payment failed"))

						return fn(ctx)
					})
			},
			expected: expected{
				err: errors.New("payment failed"),
			},
		},
		{
			name: "ошибка обновления заказа",
			args: args{
				orderUUID:     orderUUID,
				paymentMethod: model.PaymentMethodCard,
			},
			setupMock: func(repo *mocks.Repository, payment *mocks.PaymentClient, tx *mocks.TxManager) {
				tx.EXPECT().
					Do(ctx, mock.AnythingOfType("func(context.Context) error")).
					RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
						repo.EXPECT().
							Get(ctx, orderUUID).
							Return(model.Order{
								UUID:   orderUUID,
								Status: model.OrderStatusPendingPayment,
							}, nil)

						payment.EXPECT().
							PayOrder(ctx, orderUUID, model.PaymentMethodCard).
							Return(transactionUUID, nil)

						repo.EXPECT().
							Update(ctx, mock.Anything).
							Return(errors.New("update failed"))

						return fn(ctx)
					})
			},
			expected: expected{
				err: errors.New("update failed"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := mocks.NewRepository(t)
			payment := mocks.NewPaymentClient(t)
			tx := mocks.NewTxManager(t)

			tc.setupMock(repo, payment, tx)

			svc := orderService.NewService(repo, nil, payment, tx)

			got, err := svc.Pay(ctx, tc.args.orderUUID, tc.args.paymentMethod)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expected.err.Error())
				assert.Equal(t, uuid.Nil, got)
				return
			}

			require.NoError(t, err)
			assert.NotEqual(t, uuid.Nil, got)
		})
	}
}
