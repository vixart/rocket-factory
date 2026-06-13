package order

type service struct {
	orderRepository   Repository
	orderPaidProducer OrderPaidProducer
	inventoryClient   InventoryClient
	paymentClient     PaymentClient
	txManager         TxManager
}

func NewService(
	orderRepository Repository,
	orderPaidProducer OrderPaidProducer,
	inventoryClient InventoryClient,
	paymentClient PaymentClient,
	txManager TxManager,
) *service {
	return &service{
		orderRepository:   orderRepository,
		orderPaidProducer: orderPaidProducer,
		inventoryClient:   inventoryClient,
		paymentClient:     paymentClient,
		txManager:         txManager,
	}
}
