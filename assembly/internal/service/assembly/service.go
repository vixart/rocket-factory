package assembly

import "time"

type service struct {
	shipAssembledProducer ShipAssembledProducer
	minBuildTime          time.Duration
	maxBuildTime          time.Duration
}

func NewService(shipAssembledProducer ShipAssembledProducer, minBuildTime, maxBuildTime time.Duration) *service {
	return &service{
		shipAssembledProducer: shipAssembledProducer,
		minBuildTime:          minBuildTime,
		maxBuildTime:          maxBuildTime,
	}
}
