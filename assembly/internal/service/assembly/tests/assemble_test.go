package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/vixart/rocket-factory/assembly/internal/model"
	assemblyService "github.com/vixart/rocket-factory/assembly/internal/service/assembly"
	"github.com/vixart/rocket-factory/assembly/internal/service/assembly/mocks"
)

func TestAssemble(t *testing.T) {
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
		setupMock func(producer *mocks.ShipAssembledProducer)
		expected  expected
	}{
		{
			name: "успешная отправка события",
			setupMock: func(producer *mocks.ShipAssembledProducer) {
				producer.EXPECT().
					ProduceShipAssembled(
						ctx,
						mock.MatchedBy(func(event model.ShipAssembledEvent) bool {
							return event.UUID == orderUUID.String() &&
								event.OrderUUID == orderUUID.String() &&
								event.UserUUID == userUUID.String() &&
								event.BuildTimeSec == 3 &&
								!event.AssembledAt.IsZero()
						}),
					).
					Return(nil)
			},
		},
		{
			name: "ошибка отправки события",
			setupMock: func(producer *mocks.ShipAssembledProducer) {
				producer.EXPECT().
					ProduceShipAssembled(
						ctx,
						mock.MatchedBy(func(event model.ShipAssembledEvent) bool {
							return event.UUID == orderUUID.String() &&
								event.OrderUUID == orderUUID.String() &&
								event.UserUUID == userUUID.String()
						}),
					).
					Return(errors.New("producer error"))
			},
			expected: expected{
				err: errors.New("producer error"),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			producer := mocks.NewShipAssembledProducer(t)

			tc.setupMock(producer)

			svc := assemblyService.NewService(producer, time.Duration(0), time.Duration(0))

			err := svc.Assemble(
				ctx,
				orderUUID,
				userUUID,
			)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorContains(t, err, tc.expected.err.Error())
				assert.ErrorContains(t, err, "не удалось отправить событие ShipAssembled")

				return
			}

			require.NoError(t, err)
		})
	}
}
