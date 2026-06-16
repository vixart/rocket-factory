package assembly_consumer

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/vixart/rocket-factory/order/internal/model"
	eventsv1 "github.com/vixart/rocket-factory/shared/pkg/proto/events/v1"
)

func decodeShipAssembled(data []byte) (model.ShipAssembledEvent, error) {
	var pb eventsv1.ShipAssembled
	if err := proto.Unmarshal(data, &pb); err != nil {
		return model.ShipAssembledEvent{}, fmt.Errorf("не удалось десериализовать protobuf: %w", err)
	}

	return model.ShipAssembledEvent{
		UUID:         pb.GetEventUuid(),
		OrderUUID:    pb.GetOrderUuid(),
		UserUUID:     pb.GetUserUuid(),
		BuildTimeSec: pb.GetBuildTimeSec(),
		AssembledAt:  pb.GetAssembledAt().AsTime(),
	}, nil
}
