package assembly

type service struct {
	shipAssembledProducer ShipAssembledProducer
}

func NewService(shipAssembledProducer ShipAssembledProducer) *service {
	return &service{
		shipAssembledProducer: shipAssembledProducer,
	}
}
