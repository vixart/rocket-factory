package v1

import inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"

type api struct {
	inventoryv1.UnimplementedInventoryServiceServer
	inventoryService InventoryService
}

func NewApi(service InventoryService) *api {
	return &api{
		inventoryService: service,
	}
}
