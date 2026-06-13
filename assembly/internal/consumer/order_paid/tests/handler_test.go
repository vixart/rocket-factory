package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	orderPaidConsumer "github.com/vixart/rocket-factory/assembly/internal/consumer/order_paid"
	"github.com/vixart/rocket-factory/assembly/internal/consumer/order_paid/mocks"
	"github.com/vixart/rocket-factory/platform/pkg/kafka"
	eventsv1 "github.com/vixart/rocket-factory/shared/pkg/proto/events/v1"
)

func TestOrderPaidHandler(t *testing.T) {
	t.Parallel()

	type expected struct {
		err error
	}

	var (
		ctx = context.Background()

		orderUUID = uuid.New()
		userUUID  = uuid.New()
	)

	tests := []struct {
		name      string
		message   kafka.Message
		setupMock func(assembleService *mocks.ShipAssembleService)
		expected  expected
	}{
		{
			name: "успешная обработка сообщения",
			message: kafka.Message{
				Value: mustMarshalOrderPaid(
					t,
					orderUUID.String(),
					userUUID.String(),
				),
			},
			setupMock: func(assembleService *mocks.ShipAssembleService) {
				assembleService.EXPECT().
					Assemble(ctx, orderUUID, userUUID).
					Return(nil)
			},
		},
		{
			name: "не удалось декодировать сообщение",
			message: kafka.Message{
				Value: []byte("invalid protobuf"),
			},
			setupMock: func(_ *mocks.ShipAssembleService) {},
		},
		{
			name: "невалидный OrderUUID",
			message: kafka.Message{
				Value: mustMarshalOrderPaid(
					t,
					"invalid-order-uuid",
					userUUID.String(),
				),
			},
			setupMock: func(_ *mocks.ShipAssembleService) {},
		},
		{
			name: "невалидный UserUUID",
			message: kafka.Message{
				Value: mustMarshalOrderPaid(
					t,
					orderUUID.String(),
					"invalid-user-uuid",
				),
			},
			setupMock: func(_ *mocks.ShipAssembleService) {},
		},
		{
			name: "ошибка сборки корабля",
			message: kafka.Message{
				Value: mustMarshalOrderPaid(
					t,
					orderUUID.String(),
					userUUID.String(),
				),
			},
			setupMock: func(assembleService *mocks.ShipAssembleService) {
				assembleService.EXPECT().
					Assemble(ctx, orderUUID, userUUID).
					Return(errors.New("assemble error"))
			},
			expected: expected{
				err: errors.New("assemble error"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			consumer := mocks.NewConsumer(t)
			assembleService := mocks.NewShipAssembleService(t)

			tc.setupMock(assembleService)

			svc := orderPaidConsumer.NewService(
				consumer,
				assembleService,
			)

			err := svc.OrderPaidHandler(ctx, tc.message)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())

				return
			}

			require.NoError(t, err)
		})
	}
}

func mustMarshalOrderPaid(
	t *testing.T,
	orderUUID string,
	userUUID string,
) []byte {
	t.Helper()

	data, err := proto.Marshal(&eventsv1.OrderPaid{
		EventUuid: uuid.NewString(),
		OrderUuid: orderUUID,
		UserUuid:  userUUID,
	})
	require.NoError(t, err)

	return data
}
