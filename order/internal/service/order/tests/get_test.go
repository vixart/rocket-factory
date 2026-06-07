package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vixart/rocket-factory/order/internal/model"
	orderService "github.com/vixart/rocket-factory/order/internal/service/order"
	"github.com/vixart/rocket-factory/order/internal/service/order/mocks"
)

func TestGet(t *testing.T) {
	t.Parallel()

	type args struct {
		orderUUID uuid.UUID
	}

	type expected struct {
		order model.Order
		err   error
	}

	ctx := context.Background()

	orderUUID := uuid.New()

	expectedOrder := model.Order{
		UUID:   orderUUID,
		Status: model.OrderStatusPendingPayment,
	}

	repositoryErr := errors.New("repository error")

	tests := []struct {
		name      string
		args      args
		setupMock func(orderRepo *mocks.Repository)
		expected  expected
	}{
		{
			name: "успешное получение заказа",
			args: args{
				orderUUID: orderUUID,
			},
			setupMock: func(orderRepo *mocks.Repository) {
				orderRepo.EXPECT().
					Get(ctx, orderUUID).
					Return(expectedOrder, nil)
			},
			expected: expected{
				order: expectedOrder,
			},
		},
		{
			name: "ошибка репозитория",
			args: args{
				orderUUID: orderUUID,
			},
			setupMock: func(orderRepo *mocks.Repository) {
				orderRepo.EXPECT().
					Get(ctx, orderUUID).
					Return(model.Order{}, repositoryErr)
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

			tc.setupMock(orderRepo)

			svc := orderService.NewService(orderRepo, inventoryClient, paymentClient, txManager)

			order, err := svc.Get(ctx, tc.args.orderUUID)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.expected.err)
				assert.Equal(t, model.Order{}, order)

				return
			}

			require.NoError(t, err)

			assert.Equal(t, tc.expected.order, order)
		})
	}
}
