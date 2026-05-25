package order

type service struct {
	orderRepository Repository
	inventoryClient InventoryClient
	paymentClient   PaymentClient
}

func NewService(
	orderRepository Repository,
	inventoryClient InventoryClient,
	paymentClient PaymentClient,
) *service {
	return &service{
		orderRepository: orderRepository,
		inventoryClient: inventoryClient,
		paymentClient:   paymentClient,
	}
}
