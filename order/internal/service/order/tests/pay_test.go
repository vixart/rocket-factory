package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	orderService "github.com/vixart/rocket-factory/order/internal/service/order"

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

	var (
		ctx = context.Background()

		orderUUID       = uuid.New()
		userUUID        = uuid.New()
		transactionUUID = uuid.New()

		repositoryErr = errors.New("repository error")
		paymentErr    = errors.New("payment error")
		producerErr   = errors.New("producer error")
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(
			repo *mocks.Repository,
			txManager *mocks.TxManager,
			payment *mocks.PaymentClient,
			producer *mocks.OrderPaidProducer,
		)
		expected expected
	}{
		{
			name: "invalid payment method",
			args: args{
				orderUUID:     orderUUID,
				paymentMethod: model.PaymentMethodUnspecified,
			},
			expected: expected{
				err: errs.ErrInvalidPaymentMethod,
			},
		},
		{
			name: "get order error",
			args: args{
				orderUUID:     orderUUID,
				paymentMethod: model.PaymentMethodCard,
			},
			setupMock: func(
				repo *mocks.Repository,
				txManager *mocks.TxManager,
				payment *mocks.PaymentClient,
				producer *mocks.OrderPaidProducer,
			) {
				txManager.EXPECT().
					Do(ctx, mock.Anything).
					RunAndReturn(func(
						ctx context.Context,
						fn func(context.Context) error,
					) error {
						return fn(ctx)
					})

				repo.EXPECT().
					GetForUpdate(ctx, orderUUID).
					Return(model.Order{}, repositoryErr)
			},
			expected: expected{
				err: repositoryErr,
			},
		},
		{
			name: "invalid order status",
			args: args{
				orderUUID:     orderUUID,
				paymentMethod: model.PaymentMethodCard,
			},
			setupMock: func(
				repo *mocks.Repository,
				txManager *mocks.TxManager,
				payment *mocks.PaymentClient,
				producer *mocks.OrderPaidProducer,
			) {
				txManager.EXPECT().
					Do(ctx, mock.Anything).
					RunAndReturn(func(
						ctx context.Context,
						fn func(context.Context) error,
					) error {
						return fn(ctx)
					})

				repo.EXPECT().
					GetForUpdate(ctx, orderUUID).
					Return(
						model.Order{
							UUID:   orderUUID,
							Status: model.OrderStatusCancelled,
						},
						nil,
					)
			},
			expected: expected{
				err: errs.ErrInvalidOrderStatus,
			},
		},
		{
			name: "payment error",
			args: args{
				orderUUID:     orderUUID,
				paymentMethod: model.PaymentMethodCard,
			},
			setupMock: func(
				repo *mocks.Repository,
				txManager *mocks.TxManager,
				payment *mocks.PaymentClient,
				producer *mocks.OrderPaidProducer,
			) {
				txManager.EXPECT().
					Do(ctx, mock.Anything).
					RunAndReturn(func(
						ctx context.Context,
						fn func(context.Context) error,
					) error {
						return fn(ctx)
					})

				repo.EXPECT().
					GetForUpdate(ctx, orderUUID).
					Return(
						model.Order{
							UUID:   orderUUID,
							Status: model.OrderStatusPendingPayment,
						},
						nil,
					)

				payment.EXPECT().
					PayOrder(
						ctx,
						orderUUID,
						model.PaymentMethodCard,
					).
					Return(uuid.Nil, paymentErr)
			},
			expected: expected{
				err: paymentErr,
			},
		},
		{
			name: "update error",
			args: args{
				orderUUID:     orderUUID,
				paymentMethod: model.PaymentMethodCard,
			},
			setupMock: func(
				repo *mocks.Repository,
				txManager *mocks.TxManager,
				payment *mocks.PaymentClient,
				producer *mocks.OrderPaidProducer,
			) {
				txManager.EXPECT().
					Do(ctx, mock.Anything).
					RunAndReturn(func(
						ctx context.Context,
						fn func(context.Context) error,
					) error {
						return fn(ctx)
					})

				repo.EXPECT().
					GetForUpdate(ctx, orderUUID).
					Return(
						model.Order{
							UUID:   orderUUID,
							Status: model.OrderStatusPendingPayment,
						},
						nil,
					)

				payment.EXPECT().
					PayOrder(
						ctx,
						orderUUID,
						model.PaymentMethodCard,
					).
					Return(transactionUUID, nil)

				repo.EXPECT().
					Update(
						ctx,
						mock.MatchedBy(func(order model.Order) bool {
							return order.Status == model.OrderStatusPaid
						}),
					).
					Return(repositoryErr)
			},
			expected: expected{
				err: repositoryErr,
			},
		},
		{
			name: "produce event error",
			args: args{
				orderUUID:     orderUUID,
				paymentMethod: model.PaymentMethodCard,
			},
			setupMock: func(
				repo *mocks.Repository,
				txManager *mocks.TxManager,
				payment *mocks.PaymentClient,
				producer *mocks.OrderPaidProducer,
			) {
				txManager.EXPECT().
					Do(ctx, mock.Anything).
					RunAndReturn(func(
						ctx context.Context,
						fn func(context.Context) error,
					) error {
						return fn(ctx)
					})

				order := model.Order{
					UUID:     orderUUID,
					UserUUID: userUUID,
					Status:   model.OrderStatusPendingPayment,
				}

				repo.EXPECT().
					GetForUpdate(ctx, orderUUID).
					Return(order, nil)

				payment.EXPECT().
					PayOrder(
						ctx,
						orderUUID,
						model.PaymentMethodCard,
					).
					Return(transactionUUID, nil)

				repo.EXPECT().
					Update(ctx, mock.Anything).
					Return(nil)

				producer.EXPECT().
					ProduceOrderPaid(
						ctx,
						model.OrderPaidEvent{
							UUID:      orderUUID.String(),
							OrderUUID: orderUUID.String(),
							UserUUID:  userUUID.String(),
						},
					).
					Return(producerErr)
			},
			expected: expected{
				err: producerErr,
			},
		},
		{
			name: "success",
			args: args{
				orderUUID:     orderUUID,
				paymentMethod: model.PaymentMethodCard,
			},
			setupMock: func(
				repo *mocks.Repository,
				txManager *mocks.TxManager,
				payment *mocks.PaymentClient,
				producer *mocks.OrderPaidProducer,
			) {
				txManager.EXPECT().
					Do(ctx, mock.Anything).
					RunAndReturn(func(
						ctx context.Context,
						fn func(context.Context) error,
					) error {
						return fn(ctx)
					})

				order := model.Order{
					UUID:     orderUUID,
					UserUUID: userUUID,
					Status:   model.OrderStatusPendingPayment,
				}

				repo.EXPECT().
					GetForUpdate(ctx, orderUUID).
					Return(order, nil)

				payment.EXPECT().
					PayOrder(
						ctx,
						orderUUID,
						model.PaymentMethodCard,
					).
					Return(transactionUUID, nil)

				repo.EXPECT().
					Update(
						ctx,
						mock.MatchedBy(func(order model.Order) bool {
							return order.Status == model.OrderStatusPaid &&
								order.TransactionUUID != nil &&
								*order.TransactionUUID == transactionUUID &&
								order.PaymentMethod != nil &&
								*order.PaymentMethod == model.PaymentMethodCard
						}),
					).
					Return(nil)

				producer.EXPECT().
					ProduceOrderPaid(
						ctx,
						model.OrderPaidEvent{
							UUID:      orderUUID.String(),
							OrderUUID: orderUUID.String(),
							UserUUID:  userUUID.String(),
						},
					).
					Return(nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := mocks.NewRepository(t)
			inventory := mocks.NewInventoryClient(t)
			payment := mocks.NewPaymentClient(t)
			txManager := mocks.NewTxManager(t)
			producer := mocks.NewOrderPaidProducer(t)

			if tc.setupMock != nil {
				tc.setupMock(
					repo,
					txManager,
					payment,
					producer,
				)
			}

			sut := orderService.NewService(
				repo,
				producer,
				inventory,
				payment,
				txManager,
			)

			txUUID, err := sut.Pay(
				ctx,
				tc.args.orderUUID,
				tc.args.paymentMethod,
			)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.expected.err)
				assert.Equal(t, uuid.Nil, txUUID)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, transactionUUID, txUUID)
		})
	}
}
