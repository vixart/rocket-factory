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

func TestService_Cancel(t *testing.T) {
	t.Parallel()

	var (
		ctx           = context.Background()
		orderUUID     = uuid.New()
		partUUID1     = uuid.New()
		partUUID2     = uuid.New()
		repositoryErr = errors.New("repository error")
		inventoryErr  = errors.New("inventory error")
	)

	tests := []struct {
		name      string
		setupMock func(
			repo *mocks.Repository,
			inventory *mocks.InventoryClient,
			txManager *mocks.TxManager,
		)
		expectedErr error
	}{
		{
			name: "успешная отмена заказа",
			setupMock: func(
				repo *mocks.Repository,
				inventory *mocks.InventoryClient,
				txManager *mocks.TxManager,
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
							Items: []model.OrderItem{
								{UUID: partUUID1},
								{UUID: partUUID2},
							},
						},
						nil,
					)

				inventory.EXPECT().
					ReleaseParts(ctx, []uuid.UUID{partUUID1, partUUID2}).
					Return(nil)

				repo.EXPECT().
					Update(
						ctx,
						mock.MatchedBy(func(order model.Order) bool {
							return order.UUID == orderUUID &&
								order.Status == model.OrderStatusCancelled
						}),
					).
					Return(nil)
			},
		},
		{
			name: "заказ не найден",
			setupMock: func(
				repo *mocks.Repository,
				inventory *mocks.InventoryClient,
				txManager *mocks.TxManager,
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
					Return(model.Order{}, errs.ErrOrderNotFound)
			},
			expectedErr: errs.ErrOrderNotFound,
		},
		{
			name: "неверный статус заказа",
			setupMock: func(
				repo *mocks.Repository,
				inventory *mocks.InventoryClient,
				txManager *mocks.TxManager,
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
							Status: model.OrderStatusPaid,
						},
						nil,
					)
			},
			expectedErr: errs.ErrInvalidOrderStatus,
		},
		{
			name: "ошибка release parts",
			setupMock: func(
				repo *mocks.Repository,
				inventory *mocks.InventoryClient,
				txManager *mocks.TxManager,
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
							Items: []model.OrderItem{
								{UUID: partUUID1},
							},
						},
						nil,
					)

				inventory.EXPECT().
					ReleaseParts(ctx, []uuid.UUID{partUUID1}).
					Return(inventoryErr)
			},
			expectedErr: inventoryErr,
		},
		{
			name: "ошибка обновления",
			setupMock: func(
				repo *mocks.Repository,
				inventory *mocks.InventoryClient,
				txManager *mocks.TxManager,
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
							Items: []model.OrderItem{
								{UUID: partUUID1},
							},
						},
						nil,
					)

				inventory.EXPECT().
					ReleaseParts(ctx, []uuid.UUID{partUUID1}).
					Return(nil)

				repo.EXPECT().
					Update(
						ctx,
						mock.MatchedBy(func(order model.Order) bool {
							return order.UUID == orderUUID &&
								order.Status == model.OrderStatusCancelled
						}),
					).
					Return(repositoryErr)
			},
			expectedErr: repositoryErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := mocks.NewRepository(t)
			inventory := mocks.NewInventoryClient(t)
			txManager := mocks.NewTxManager(t)

			tt.setupMock(repo, inventory, txManager)

			sut := order.NewService(
				repo,
				nil,
				inventory,
				nil,
				txManager,
			)

			err := sut.Cancel(ctx, orderUUID)

			if tt.expectedErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedErr)

				return
			}

			require.NoError(t, err)
		})
	}
}
