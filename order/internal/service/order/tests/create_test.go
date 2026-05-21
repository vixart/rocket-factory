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

func TestCreate(t *testing.T) {
	t.Parallel()

	type args struct {
		orderParts model.OrderParts
	}

	type expected struct {
		err error
	}

	ctx := context.Background()

	hullUUID := uuid.New()
	engineUUID := uuid.New()
	shieldUUID := uuid.New()
	weaponUUID := uuid.New()

	tests := []struct {
		name      string
		args      args
		setupMock func(
			orderRepo *mocks.Repository,
			inventoryClient *mocks.InventoryClient,
		)
		expected expected
	}{
		{
			name: "успешное создание заказа только с обязательными деталями",
			args: args{
				orderParts: model.OrderParts{
					HullUUID:   hullUUID,
					EngineUUID: engineUUID,
				},
			},
			setupMock: func(
				orderRepo *mocks.Repository,
				inventoryClient *mocks.InventoryClient,
			) {
				parts := []model.Part{
					{
						UUID:          hullUUID,
						Price:         100000,
						StockQuantity: 5,
					},
					{
						UUID:          engineUUID,
						Price:         50000,
						StockQuantity: 3,
					},
				}

				inventoryClient.EXPECT().
					ListParts(mock.Anything, []uuid.UUID{
						hullUUID,
						engineUUID,
					}).
					Return(parts, nil)

				orderRepo.EXPECT().
					Create(ctx, mock.MatchedBy(func(order model.Order) bool {
						return order.HullUUID == hullUUID &&
							order.EngineUUID == engineUUID &&
							order.TotalPrice == 150000 &&
							order.Status == model.OrderStatusPendingPayment &&
							order.ShieldUUID == nil &&
							order.WeaponUUID == nil
					})).
					Return(nil)
			},
			expected: expected{},
		},
		{
			name: "успешное создание заказа со всеми деталями",
			args: args{
				orderParts: model.OrderParts{
					HullUUID:   hullUUID,
					EngineUUID: engineUUID,
					ShieldUUID: &shieldUUID,
					WeaponUUID: &weaponUUID,
				},
			},
			setupMock: func(
				orderRepo *mocks.Repository,
				inventoryClient *mocks.InventoryClient,
			) {
				parts := []model.Part{
					{
						UUID:          hullUUID,
						Price:         100000,
						StockQuantity: 5,
					},
					{
						UUID:          engineUUID,
						Price:         50000,
						StockQuantity: 3,
					},
					{
						UUID:          shieldUUID,
						Price:         25000,
						StockQuantity: 2,
					},
					{
						UUID:          weaponUUID,
						Price:         30000,
						StockQuantity: 1,
					},
				}

				inventoryClient.EXPECT().
					ListParts(mock.Anything, []uuid.UUID{
						hullUUID,
						engineUUID,
						shieldUUID,
						weaponUUID,
					}).
					Return(parts, nil)

				orderRepo.EXPECT().
					Create(ctx, mock.MatchedBy(func(order model.Order) bool {
						return order.TotalPrice == 205000 &&
							order.ShieldUUID != nil &&
							order.WeaponUUID != nil &&
							*order.ShieldUUID == shieldUUID &&
							*order.WeaponUUID == weaponUUID
					})).
					Return(nil)
			},
			expected: expected{},
		},
		{
			name: "ошибка inventory клиента",
			args: args{
				orderParts: model.OrderParts{
					HullUUID:   hullUUID,
					EngineUUID: engineUUID,
				},
			},
			setupMock: func(
				orderRepo *mocks.Repository,
				inventoryClient *mocks.InventoryClient,
			) {
				inventoryErr := errors.New("inventory unavailable")

				inventoryClient.EXPECT().
					ListParts(mock.Anything, mock.Anything).
					Return(nil, inventoryErr)
			},
			expected: expected{
				err: errors.New("inventory unavailable"),
			},
		},
		{
			name: "деталь не найдена",
			args: args{
				orderParts: model.OrderParts{
					HullUUID:   hullUUID,
					EngineUUID: engineUUID,
				},
			},
			setupMock: func(
				orderRepo *mocks.Repository,
				inventoryClient *mocks.InventoryClient,
			) {
				parts := []model.Part{
					{
						UUID:          hullUUID,
						Price:         100000,
						StockQuantity: 5,
					},
				}

				inventoryClient.EXPECT().
					ListParts(mock.Anything, mock.Anything).
					Return(parts, nil)
			},
			expected: expected{
				err: errs.ErrPartNotFound,
			},
		},
		{
			name: "деталь отсутствует на складе",
			args: args{
				orderParts: model.OrderParts{
					HullUUID:   hullUUID,
					EngineUUID: engineUUID,
				},
			},
			setupMock: func(
				orderRepo *mocks.Repository,
				inventoryClient *mocks.InventoryClient,
			) {
				parts := []model.Part{
					{
						UUID:          hullUUID,
						Price:         100000,
						StockQuantity: 0,
					},
					{
						UUID:          engineUUID,
						Price:         50000,
						StockQuantity: 3,
					},
				}

				inventoryClient.EXPECT().
					ListParts(mock.Anything, mock.Anything).
					Return(parts, nil)
			},
			expected: expected{
				err: errs.ErrPartInsufficientStock,
			},
		},
		{
			name: "ошибка сохранения заказа",
			args: args{
				orderParts: model.OrderParts{
					HullUUID:   hullUUID,
					EngineUUID: engineUUID,
				},
			},
			setupMock: func(
				orderRepo *mocks.Repository,
				inventoryClient *mocks.InventoryClient,
			) {
				repositoryErr := errors.New("repository error")

				parts := []model.Part{
					{
						UUID:          hullUUID,
						Price:         100000,
						StockQuantity: 5,
					},
					{
						UUID:          engineUUID,
						Price:         50000,
						StockQuantity: 3,
					},
				}

				inventoryClient.EXPECT().
					ListParts(mock.Anything, mock.Anything).
					Return(parts, nil)

				orderRepo.EXPECT().
					Create(ctx, mock.Anything).
					Return(repositoryErr)
			},
			expected: expected{
				err: errors.New("repository error"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			orderRepo := mocks.NewRepository(t)
			inventoryClient := mocks.NewInventoryClient(t)
			paymentClient := mocks.NewPaymentClient(t)

			tc.setupMock(orderRepo, inventoryClient)

			svc := orderService.NewService(orderRepo, inventoryClient, paymentClient)

			order, err := svc.Create(ctx, tc.args.orderParts)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())
				assert.Nil(t, order)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, order)

			assert.NotEqual(t, uuid.Nil, order.OrderUUID)
		})
	}
}
