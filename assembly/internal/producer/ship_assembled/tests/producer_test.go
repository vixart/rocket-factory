package tests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/vixart/rocket-factory/assembly/internal/model"
	shipAssembledProducer "github.com/vixart/rocket-factory/assembly/internal/producer/ship_assembled"
	"github.com/vixart/rocket-factory/assembly/internal/producer/ship_assembled/mocks"
	"github.com/vixart/rocket-factory/platform/pkg/kafka"
	eventsv1 "github.com/vixart/rocket-factory/shared/pkg/proto/events/v1"
)

func TestProduceShipAssembled(t *testing.T) {
	t.Parallel()

	type expected struct {
		err error
	}

	var (
		ctx = context.Background()

		event = model.ShipAssembledEvent{
			UUID:         uuid.NewString(),
			OrderUUID:    uuid.NewString(),
			UserUUID:     uuid.NewString(),
			BuildTimeSec: 3600,
			AssembledAt:  time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC),
		}
	)

	tests := []struct {
		name      string
		setupMock func(producer *mocks.KafkaProducer)
		expected  expected
	}{
		{
			name: "успешная отправка события",
			setupMock: func(producer *mocks.KafkaProducer) {
				producer.On(
					"Send",
					ctx,
					mock.MatchedBy(func(msg *kafka.Message) bool {
						var pb eventsv1.ShipAssembled

						err := proto.Unmarshal(msg.Value, &pb)
						require.NoError(t, err)

						if string(msg.Key) != event.OrderUUID {
							return false
						}

						assert.Equal(t, event.UUID, pb.EventUuid)
						assert.Equal(t, event.OrderUUID, pb.OrderUuid)
						assert.Equal(t, event.UserUUID, pb.UserUuid)
						assert.Equal(t, event.BuildTimeSec, pb.BuildTimeSec)

						require.NotNil(t, pb.AssembledAt)
						assert.True(
							t,
							event.AssembledAt.Equal(pb.AssembledAt.AsTime()),
						)

						return true
					}),
				).
					Return(nil)
			},
		},
		{
			name: "ошибка отправки события",
			setupMock: func(producer *mocks.KafkaProducer) {
				producer.On(
					"Send",
					ctx,
					mock.Anything,
				).
					Return(assert.AnError)
			},
			expected: expected{
				err: assert.AnError,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			producer := mocks.NewKafkaProducer(t)

			tc.setupMock(producer)

			svc := shipAssembledProducer.NewService(producer)

			err := svc.ProduceShipAssembled(ctx, event)

			if tc.expected.err != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.expected.err)
				return
			}

			require.NoError(t, err)

			producer.AssertExpectations(t)
		})
	}
}
