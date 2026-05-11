package v1

import orderv1 "github.com/vixart/rocket-factory/shared/pkg/openapi/order/v1"

type api struct {
	orderv1.UnimplementedHandler
	orderService OrderService
}

func NewApi(service OrderService) *api {
	return &api{
		orderService: service,
	}
}
