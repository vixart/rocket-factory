package order

type service struct {
	orderRepository Repository
	inventoryClient InventoryClient
	paymentClient   PaymentClient
	txManager       TxManager
}

func NewService(
	orderRepository Repository,
	inventoryClient InventoryClient,
	paymentClient PaymentClient,
	txManager TxManager,
) *service {
	return &service{
		orderRepository: orderRepository,
		inventoryClient: inventoryClient,
		paymentClient:   paymentClient,
		txManager:       txManager,
	}
}
