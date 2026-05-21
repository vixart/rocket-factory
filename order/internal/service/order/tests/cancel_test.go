package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vixart/rocket-factory/order/internal/service/order"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
	"github.com/vixart/rocket-factory/order/internal/service/order/mocks"
)

func TestCancel(t *testing.T) {
	t.Parallel()

	type args struct {
		orderUUID uuid.UUID
	}

	type expected struct {
		err error
	}

	var (
		ctx = context.Background()

		orderUUID = uuid.New()

		repositoryErr = errors.New("repository error")
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(repo *mocks.Repository)
		expected  expected
	}{
		{
			name: "успешная отмена заказа",
			args: args{
				orderUUID: orderUUID,
			},
			setupMock: func(repo *mocks.Repository) {
				order := model.Order{
					OrderUUID: orderUUID,
					Status:    model.OrderStatusPendingPayment,
				}

				repo.EXPECT().
					Get(ctx, orderUUID).
					Return(order, nil)

				repo.EXPECT().
					Update(ctx, mock.MatchedBy(func(order model.Order) bool {
						return order.OrderUUID == orderUUID &&
							order.Status == model.OrderStatusCanceled
					})).
					Return(nil)
			},
			expected: expected{},
		},
		{
			name: "заказ не найден",
			args: args{
				orderUUID: orderUUID,
			},
			setupMock: func(repo *mocks.Repository) {
				repo.EXPECT().
					Get(ctx, orderUUID).
					Return(model.Order{}, errs.ErrOrderNotFound)
			},
			expected: expected{
				err: errs.ErrOrderNotFound,
			},
		},
		{
			name: "нельзя отменить заказ с неверным статусом",
			args: args{
				orderUUID: orderUUID,
			},
			setupMock: func(repo *mocks.Repository) {
				order := model.Order{
					OrderUUID: orderUUID,
					Status:    model.OrderStatusPaid,
				}

				repo.EXPECT().
					Get(ctx, orderUUID).
					Return(order, nil)
			},
			expected: expected{
				err: errs.ErrInvalidOrderStatus,
			},
		},
		{
			name: "ошибка обновления заказа",
			args: args{
				orderUUID: orderUUID,
			},
			setupMock: func(repo *mocks.Repository) {
				order := model.Order{
					OrderUUID: orderUUID,
					Status:    model.OrderStatusPendingPayment,
				}

				repo.EXPECT().
					Get(ctx, orderUUID).
					Return(order, nil)

				repo.EXPECT().
					Update(ctx, mock.MatchedBy(func(order model.Order) bool {
						return order.OrderUUID == orderUUID &&
							order.Status == model.OrderStatusCanceled
					})).
					Return(repositoryErr)
			},
			expected: expected{
				err: repositoryErr,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			orderRepo := mocks.NewRepository(t)
			inventoryClient := mocks.NewInventoryClient(t)
			paymentClient := mocks.NewPaymentClient(t)

			tc.setupMock(orderRepo)

			svc := order.NewService(orderRepo, inventoryClient, paymentClient)

			err := svc.Cancel(ctx, tc.args.orderUUID)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.expected.err)

				return
			}

			require.NoError(t, err)
		})
	}
}
