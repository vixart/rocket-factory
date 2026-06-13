package order_producer

import (
	"context"
	"log/slog"

	"google.golang.org/protobuf/proto"

	"github.com/vixart/rocket-factory/order/internal/model"
	"github.com/vixart/rocket-factory/platform/pkg/kafka"
	eventsv1 "github.com/vixart/rocket-factory/shared/pkg/proto/events/v1"
)

type service struct {
	orderPaidProducer KafkaProducer
}

func NewService(orderPaidProducer KafkaProducer) *service {
	return &service{
		orderPaidProducer: orderPaidProducer,
	}
}

func (p *service) ProduceOrderPaid(ctx context.Context, event model.OrderPaidEvent) error {
	msg := &eventsv1.OrderPaid{
		EventUuid: event.UUID,
		OrderUuid: event.OrderUUID,
		UserUuid:  event.UserUUID,
	}

	payload, err := proto.Marshal(msg)
	if err != nil {
		slog.ErrorContext(ctx, "не удалось сериализовать OrderPaid", "error", err)
		return err
	}

	return p.orderPaidProducer.Send(ctx, &kafka.Message{
		Key:   []byte(event.UUID),
		Value: payload,
	})
}
