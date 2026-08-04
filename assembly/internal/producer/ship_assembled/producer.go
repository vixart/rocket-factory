package ship_assembled

import (
	"context"
	"log/slog"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/vixart/rocket-factory/assembly/internal/model"
	"github.com/vixart/rocket-factory/platform/pkg/kafka"
	kafkamw "github.com/vixart/rocket-factory/platform/pkg/middleware/kafka"
	eventsv1 "github.com/vixart/rocket-factory/shared/pkg/proto/events/v1"
)

type service struct {
	shipAssembledProducer KafkaProducer
}

func NewService(shipAssembledProducer KafkaProducer) *service {
	return &service{
		shipAssembledProducer: shipAssembledProducer,
	}
}

func (p *service) ProduceShipAssembled(ctx context.Context, event model.ShipAssembledEvent) error {
	msg := &eventsv1.ShipAssembled{
		EventUuid:    event.UUID,
		OrderUuid:    event.OrderUUID,
		UserUuid:     event.UserUUID,
		BuildTimeSec: event.BuildTimeSec,
		AssembledAt:  timestamppb.New(event.AssembledAt),
	}

	payload, err := proto.Marshal(msg)
	if err != nil {
		slog.ErrorContext(ctx, "failed to marshal ShipAssembled", "error", err)
		return err
	}

	return p.shipAssembledProducer.Send(ctx, &kafka.Message{
		Key:     []byte(event.OrderUUID),
		Value:   payload,
		Headers: kafkamw.ProducerSessionHeaders(ctx),
	})
}
