package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/vixart/rocket-factory/platform/pkg/auth"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
	"github.com/vixart/rocket-factory/order/internal/service/input"
	orderService "github.com/vixart/rocket-factory/order/internal/service/order"
	"github.com/vixart/rocket-factory/order/internal/service/order/mocks"
)

func ctxWithUser(userUUID uuid.UUID) context.Context {
	return auth.WithUserUUID(context.Background(), userUUID)
}

func TestCreate(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx        context.Context
		orderParts input.OrderParts
	}

	type expected struct {
		err error
	}

	var (
		userUUID = uuid.New()

		hullUUID   = uuid.New()
		engineUUID = uuid.New()
		shieldUUID = uuid.New()
		weaponUUID = uuid.New()

		inventoryErr     = errors.New("inventory unavailable")
		compatibilityErr = errors.New("compatibility error")
		reserveErr       = errors.New("reserve error")
		repositoryErr    = errors.New("repository error")
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
			name: "не авторизован",
			args: args{
				ctx: context.Background(),
				orderParts: input.OrderParts{
					HullUUID:   hullUUID,
					EngineUUID: engineUUID,
				},
			},
			setupMock: func(repo *mocks.Repository, inventory *mocks.InventoryClient) {},
			expected: expected{
				err: errs.ErrUnauthorized,
			},
		},
		{
			name: "успешное создание заказа только с обязательными деталями",
			args: args{
				ctx: ctxWithUser(userUUID),
				orderParts: input.OrderParts{
					HullUUID:   hullUUID,
					EngineUUID: engineUUID,
				},
			},
			setupMock: func(
				repo *mocks.Repository,
				inventory *mocks.InventoryClient,
			) {
				parts := []model.Part{
					{
						UUID:          hullUUID,
						Price:         100000,
						StockQuantity: 5,
						PartType:      model.PartTypeHull,
					},
					{
						UUID:          engineUUID,
						Price:         50000,
						StockQuantity: 3,
						PartType:      model.PartTypeEngine,
					},
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
					Create(
						mock.Anything,
						mock.MatchedBy(func(order model.Order) bool {
							return order.UserUUID == userUUID &&
								order.Status == model.OrderStatusPendingPayment &&
								len(order.Items) == 2 &&
								order.TotalPrice() == 150000
						}),
					).
					Return(nil)
			},
		},
		{
			name: "успешное создание заказа со всеми деталями",
			args: args{
				ctx: ctxWithUser(userUUID),
				orderParts: input.OrderParts{
					HullUUID:   hullUUID,
					EngineUUID: engineUUID,
					ShieldUUID: &shieldUUID,
					WeaponUUID: &weaponUUID,
				},
			},
			setupMock: func(
				repo *mocks.Repository,
				inventory *mocks.InventoryClient,
			) {
				parts := []model.Part{
					{
						UUID:          hullUUID,
						Price:         100000,
						StockQuantity: 5,
						PartType:      model.PartTypeHull,
					},
					{
						UUID:          engineUUID,
						Price:         50000,
						StockQuantity: 3,
						PartType:      model.PartTypeEngine,
					},
					{
						UUID:          shieldUUID,
						Price:         25000,
						StockQuantity: 2,
						PartType:      model.PartTypeShield,
					},
					{
						UUID:          weaponUUID,
						Price:         30000,
						StockQuantity: 1,
						PartType:      model.PartTypeWeapon,
					},
				}

				partUUIDs := []uuid.UUID{hullUUID, engineUUID, shieldUUID, weaponUUID}

				inventory.EXPECT().
					ListParts(mock.Anything, partUUIDs).
					Return(parts, nil)

				inventory.EXPECT().
					ValidateCompatibility(mock.Anything, mock.Anything).
					Return(nil)

				inventory.EXPECT().
					ReserveParts(mock.Anything, partUUIDs).
					Return(nil)

				repo.EXPECT().
					Create(
						mock.Anything,
						mock.MatchedBy(func(order model.Order) bool {
							return order.UserUUID == userUUID &&
								len(order.Items) == 4 &&
								order.TotalPrice() == 205000
						}),
					).
					Return(nil)
			},
		},
		{
			name: "duplicate uuids",
			args: args{
				ctx: ctxWithUser(userUUID),
				orderParts: input.OrderParts{
					HullUUID:   hullUUID,
					EngineUUID: hullUUID,
				},
			},
			setupMock: func(repo *mocks.Repository, inventory *mocks.InventoryClient) {},
			expected: expected{
				err: errs.ErrInvalidUUID,
			},
		},
		{
			name: "ошибка inventory ListParts",
			args: args{
				ctx: ctxWithUser(userUUID),
				orderParts: input.OrderParts{
					HullUUID:   hullUUID,
					EngineUUID: engineUUID,
				},
			},
			setupMock: func(
				repo *mocks.Repository,
				inventory *mocks.InventoryClient,
			) {
				inventory.EXPECT().
					ListParts(mock.Anything, mock.Anything).
					Return(nil, inventoryErr)
			},
			expected: expected{
				err: inventoryErr,
			},
		},
		{
			name: "деталь не найдена",
			args: args{
				ctx: ctxWithUser(userUUID),
				orderParts: input.OrderParts{
					HullUUID:   hullUUID,
					EngineUUID: engineUUID,
				},
			},
			setupMock: func(
				repo *mocks.Repository,
				inventory *mocks.InventoryClient,
			) {
				inventory.EXPECT().
					ListParts(mock.Anything, mock.Anything).
					Return([]model.Part{
						{UUID: hullUUID, Price: 100000, StockQuantity: 5},
					}, nil)
			},
			expected: expected{
				err: errs.ErrPartNotFound,
			},
		},
		{
			name: "нет товара на складе",
			args: args{
				ctx: ctxWithUser(userUUID),
				orderParts: input.OrderParts{
					HullUUID:   hullUUID,
					EngineUUID: engineUUID,
				},
			},
			setupMock: func(
				repo *mocks.Repository,
				inventory *mocks.InventoryClient,
			) {
				inventory.EXPECT().
					ListParts(mock.Anything, mock.Anything).
					Return([]model.Part{
						{UUID: hullUUID, Price: 100000, StockQuantity: 0},
						{UUID: engineUUID, Price: 50000, StockQuantity: 3},
					}, nil)
			},
			expected: expected{
				err: errs.ErrOutOfStock,
			},
		},
		{
			name: "ошибка проверки совместимости",
			args: args{
				ctx: ctxWithUser(userUUID),
				orderParts: input.OrderParts{
					HullUUID:   hullUUID,
					EngineUUID: engineUUID,
				},
			},
			setupMock: func(
				repo *mocks.Repository,
				inventory *mocks.InventoryClient,
			) {
				inventory.EXPECT().
					ListParts(mock.Anything, mock.Anything).
					Return([]model.Part{
						{UUID: hullUUID, Price: 100000, StockQuantity: 5},
						{UUID: engineUUID, Price: 50000, StockQuantity: 3},
					}, nil)

				inventory.EXPECT().
					ValidateCompatibility(mock.Anything, mock.Anything).
					Return(compatibilityErr)
			},
			expected: expected{
				err: compatibilityErr,
			},
		},
		{
			name: "ошибка резервирования деталей",
			args: args{
				ctx: ctxWithUser(userUUID),
				orderParts: input.OrderParts{
					HullUUID:   hullUUID,
					EngineUUID: engineUUID,
				},
			},
			setupMock: func(
				repo *mocks.Repository,
				inventory *mocks.InventoryClient,
			) {
				inventory.EXPECT().
					ListParts(mock.Anything, mock.Anything).
					Return([]model.Part{
						{UUID: hullUUID, Price: 100000, StockQuantity: 5},
						{UUID: engineUUID, Price: 50000, StockQuantity: 3},
					}, nil)

				inventory.EXPECT().
					ValidateCompatibility(mock.Anything, mock.Anything).
					Return(nil)

				inventory.EXPECT().
					ReserveParts(mock.Anything, mock.Anything).
					Return(reserveErr)
			},
			expected: expected{
				err: reserveErr,
			},
		},
		{
			name: "ошибка сохранения заказа",
			args: args{
				ctx: ctxWithUser(userUUID),
				orderParts: input.OrderParts{
					HullUUID:   hullUUID,
					EngineUUID: engineUUID,
				},
			},
			setupMock: func(
				repo *mocks.Repository,
				inventory *mocks.InventoryClient,
			) {
				inventory.EXPECT().
					ListParts(mock.Anything, mock.Anything).
					Return([]model.Part{
						{UUID: hullUUID, Price: 100000, StockQuantity: 5},
						{UUID: engineUUID, Price: 50000, StockQuantity: 3},
					}, nil)

				inventory.EXPECT().
					ValidateCompatibility(mock.Anything, mock.Anything).
					Return(nil)

				inventory.EXPECT().
					ReserveParts(mock.Anything, mock.Anything).
					Return(nil)

				repo.EXPECT().
					Create(mock.Anything, mock.Anything).
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

			repo := mocks.NewRepository(t)
			inventory := mocks.NewInventoryClient(t)
			payment := mocks.NewPaymentClient(t)
			txManager := mocks.NewTxManager(t)
			orderPaidProducer := mocks.NewOrderPaidProducer(t)

			if tc.setupMock != nil {
				tc.setupMock(repo, inventory)
			}

			sut := orderService.NewService(
				repo,
				orderPaidProducer,
				inventory,
				payment,
				txManager,
			)

			order, err := sut.Create(tc.args.ctx, tc.args.orderParts)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.expected.err)
				assert.Nil(t, order)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, order)

			assert.NotEqual(t, uuid.Nil, order.UUID)
			assert.Equal(t, userUUID, order.UserUUID)
			assert.Equal(t, model.OrderStatusPendingPayment, order.Status)
		})
	}
}
