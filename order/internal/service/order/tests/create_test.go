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
	"github.com/vixart/rocket-factory/order/internal/service/input"
	orderService "github.com/vixart/rocket-factory/order/internal/service/order"
	"github.com/vixart/rocket-factory/order/internal/service/order/mocks"
)

func TestCreate(t *testing.T) {
	t.Parallel()

	type args struct {
		orderParts input.OrderParts
	}

	type expected struct {
		err error
	}

	var (
		ctx = context.Background()

		hullUUID   = uuid.New()
		engineUUID = uuid.New()
		shieldUUID = uuid.New()
		weaponUUID = uuid.New()
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
			name: "успешное создание заказа только с обязательными деталями",
			args: args{
				orderParts: input.OrderParts{
					HullUUID:   hullUUID,
					EngineUUID: engineUUID,
				},
			},
			setupMock: func(repo *mocks.Repository, inventory *mocks.InventoryClient) {
				parts := []model.Part{
					{UUID: hullUUID, Price: 100000, StockQuantity: 5, PartType: model.PartTypeHull},
					{UUID: engineUUID, Price: 50000, StockQuantity: 3, PartType: model.PartTypeEngine},
				}

				inventory.EXPECT().
					ListParts(mock.Anything, []uuid.UUID{hullUUID, engineUUID}).
					Return(parts, nil)

				inventory.EXPECT().
					ValidateCompatibility(mock.Anything, mock.Anything).
					Return(nil)

				inventory.EXPECT().
					ReserveParts(mock.Anything, []uuid.UUID{hullUUID, engineUUID}).
					Return(nil)

				repo.EXPECT().
					Create(mock.Anything, mock.MatchedBy(func(order model.Order) bool {
						return order.Status == model.OrderStatusPendingPayment &&
							len(order.Items) == 2 &&
							order.TotalPrice() == 150000
					})).
					Return(nil)
			},
		},
		{
			name: "успешное создание заказа со всеми деталями",
			args: args{
				orderParts: input.OrderParts{
					HullUUID:   hullUUID,
					EngineUUID: engineUUID,
					ShieldUUID: &shieldUUID,
					WeaponUUID: &weaponUUID,
				},
			},
			setupMock: func(repo *mocks.Repository, inventory *mocks.InventoryClient) {
				parts := []model.Part{
					{UUID: hullUUID, Price: 100000, StockQuantity: 5, PartType: model.PartTypeHull},
					{UUID: engineUUID, Price: 50000, StockQuantity: 3, PartType: model.PartTypeEngine},
					{UUID: shieldUUID, Price: 25000, StockQuantity: 2, PartType: model.PartTypeShield},
					{UUID: weaponUUID, Price: 30000, StockQuantity: 1, PartType: model.PartTypeWeapon},
				}

				inventory.EXPECT().
					ListParts(mock.Anything, []uuid.UUID{
						hullUUID, engineUUID, shieldUUID, weaponUUID,
					}).
					Return(parts, nil)

				inventory.EXPECT().
					ValidateCompatibility(mock.Anything, mock.Anything).
					Return(nil)

				inventory.EXPECT().
					ReserveParts(mock.Anything, []uuid.UUID{
						hullUUID, engineUUID, shieldUUID, weaponUUID,
					}).
					Return(nil)

				repo.EXPECT().
					Create(mock.Anything, mock.MatchedBy(func(order model.Order) bool {
						return len(order.Items) == 4 &&
							order.TotalPrice() == 205000
					})).
					Return(nil)
			},
		},
		{
			name: "ошибка inventory ListParts",
			args: args{
				orderParts: input.OrderParts{
					HullUUID:   hullUUID,
					EngineUUID: engineUUID,
				},
			},
			setupMock: func(repo *mocks.Repository, inventory *mocks.InventoryClient) {
				inventory.EXPECT().
					ListParts(mock.Anything, mock.Anything).
					Return(nil, errors.New("inventory unavailable"))
			},
			expected: expected{
				err: errors.New("inventory unavailable"),
			},
		},
		{
			name: "деталь не найдена",
			args: args{
				orderParts: input.OrderParts{
					HullUUID:   hullUUID,
					EngineUUID: engineUUID,
				},
			},
			setupMock: func(repo *mocks.Repository, inventory *mocks.InventoryClient) {
				parts := []model.Part{
					{UUID: hullUUID, Price: 100000, StockQuantity: 5},
				}

				inventory.EXPECT().
					ListParts(mock.Anything, mock.Anything).
					Return(parts, nil)
			},
			expected: expected{
				err: errs.ErrPartNotFound,
			},
		},
		{
			name: "нет товара на складе",
			args: args{
				orderParts: input.OrderParts{
					HullUUID:   hullUUID,
					EngineUUID: engineUUID,
				},
			},
			setupMock: func(repo *mocks.Repository, inventory *mocks.InventoryClient) {
				parts := []model.Part{
					{UUID: hullUUID, Price: 100000, StockQuantity: 0},
					{UUID: engineUUID, Price: 50000, StockQuantity: 3},
				}

				inventory.EXPECT().
					ListParts(mock.Anything, mock.Anything).
					Return(parts, nil)
			},
			expected: expected{
				err: errs.ErrOutOfStock,
			},
		},
		{
			name: "ошибка сохранения заказа",
			args: args{
				orderParts: input.OrderParts{
					HullUUID:   hullUUID,
					EngineUUID: engineUUID,
				},
			},
			setupMock: func(repo *mocks.Repository, inventory *mocks.InventoryClient) {
				parts := []model.Part{
					{UUID: hullUUID, Price: 100000, StockQuantity: 5},
					{UUID: engineUUID, Price: 50000, StockQuantity: 3},
				}

				inventory.EXPECT().
					ListParts(mock.Anything, mock.Anything).
					Return(parts, nil)

				inventory.EXPECT().
					ValidateCompatibility(mock.Anything, mock.Anything).
					Return(nil)

				inventory.EXPECT().
					ReserveParts(mock.Anything, mock.Anything).
					Return(nil)

				repo.EXPECT().
					Create(mock.Anything, mock.Anything).
					Return(errors.New("repository error"))
			},
			expected: expected{
				err: errors.New("repository error"),
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

			tc.setupMock(repo, inventory)

			svc := orderService.NewService(repo, inventory, payment, txManager)

			order, err := svc.Create(ctx, tc.args.orderParts)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expected.err.Error())
				assert.Nil(t, order)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, order)
			assert.NotEqual(t, uuid.Nil, order.UUID)
		})
	}
}
