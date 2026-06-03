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
	"github.com/vixart/rocket-factory/order/internal/service/order"
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
		partUUID1 = uuid.New()
		partUUID2 = uuid.New()

		repositoryErr = errors.New("repository error")
		inventoryErr  = errors.New("inventory error")
	)

	tests := []struct {
		name      string
		args      args
		setupMock func(
			repo *mocks.Repository,
			inventory *mocks.InventoryClient,
		)
		expected expected
	}{
		{
			name: "успешная отмена заказа",
			args: args{
				orderUUID: orderUUID,
			},
			setupMock: func(
				repo *mocks.Repository,
				inventory *mocks.InventoryClient,
			) {
				orderRes := model.Order{
					UUID:   orderUUID,
					Status: model.OrderStatusPendingPayment,
					Items: []model.OrderItem{
						{UUID: partUUID1},
						{UUID: partUUID2},
					},
				}

				repo.EXPECT().
					Get(ctx, orderUUID).
					Return(orderRes, nil)

				inventory.EXPECT().
					ReleaseParts(ctx, []uuid.UUID{
						partUUID1,
						partUUID2,
					}).
					Return(nil)

				repo.EXPECT().
					Update(ctx, mock.MatchedBy(func(order model.Order) bool {
						return order.UUID == orderUUID &&
							order.Status == model.OrderStatusCancelled
					})).
					Return(nil)
			},
		},
		{
			name: "заказ не найден",
			args: args{
				orderUUID: orderUUID,
			},
			setupMock: func(
				repo *mocks.Repository,
				inventory *mocks.InventoryClient,
			) {
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
			setupMock: func(
				repo *mocks.Repository,
				inventory *mocks.InventoryClient,
			) {
				orderRes := model.Order{
					UUID:   orderUUID,
					Status: model.OrderStatusPaid,
				}

				repo.EXPECT().
					Get(ctx, orderUUID).
					Return(orderRes, nil)
			},
			expected: expected{
				err: errs.ErrInvalidOrderStatus,
			},
		},
		{
			name: "ошибка освобождения деталей",
			args: args{
				orderUUID: orderUUID,
			},
			setupMock: func(
				repo *mocks.Repository,
				inventory *mocks.InventoryClient,
			) {
				orderRes := model.Order{
					UUID:   orderUUID,
					Status: model.OrderStatusPendingPayment,
					Items: []model.OrderItem{
						{UUID: partUUID1},
					},
				}

				repo.EXPECT().
					Get(ctx, orderUUID).
					Return(orderRes, nil)

				inventory.EXPECT().
					ReleaseParts(ctx, []uuid.UUID{
						partUUID1,
					}).
					Return(inventoryErr)
			},
			expected: expected{
				err: inventoryErr,
			},
		},
		{
			name: "ошибка обновления заказа",
			args: args{
				orderUUID: orderUUID,
			},
			setupMock: func(
				repo *mocks.Repository,
				inventory *mocks.InventoryClient,
			) {
				orderRes := model.Order{
					UUID:   orderUUID,
					Status: model.OrderStatusPendingPayment,
					Items: []model.OrderItem{
						{UUID: partUUID1},
					},
				}

				repo.EXPECT().
					Get(ctx, orderUUID).
					Return(orderRes, nil)

				inventory.EXPECT().
					ReleaseParts(ctx, []uuid.UUID{
						partUUID1,
					}).
					Return(nil)

				repo.EXPECT().
					Update(ctx, mock.MatchedBy(func(order model.Order) bool {
						return order.UUID == orderUUID &&
							order.Status == model.OrderStatusCancelled
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
			txManager := mocks.NewTxManager(t)

			tc.setupMock(orderRepo, inventoryClient)

			svc := order.NewService(orderRepo, inventoryClient, paymentClient, txManager)

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
