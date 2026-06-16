package order_paid

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/vixart/rocket-factory/assembly/internal/model"
	eventsv1 "github.com/vixart/rocket-factory/shared/pkg/proto/events/v1"
)

func decodeOrderPaid(data []byte) (model.OrderPaidEvent, error) {
	var pb eventsv1.OrderPaid
	if err := proto.Unmarshal(data, &pb); err != nil {
		return model.OrderPaidEvent{}, fmt.Errorf("не удалось десериализовать protobuf: %w", err)
	}

	return model.OrderPaidEvent{
		UUID:      pb.EventUuid,
		OrderUUID: pb.OrderUuid,
		UserUUID:  pb.UserUuid,
	}, nil
}
