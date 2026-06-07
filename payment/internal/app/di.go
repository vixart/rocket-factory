package app

import (
	"context"

	paymentApiV1 "github.com/vixart/rocket-factory/payment/internal/api/payment/v1"
	paymentService "github.com/vixart/rocket-factory/payment/internal/service/payment"
	paymentv1 "github.com/vixart/rocket-factory/shared/pkg/proto/payment/v1"
)

type diContainer struct {
	// Сервисы
	paymentSvc paymentApiV1.PaymentService

	// API-обработчики
	paymentv1Handler paymentv1.PaymentServiceServer
}

func (d *diContainer) PaymentService(_ context.Context) paymentApiV1.PaymentService {
	if d.paymentSvc == nil {
		d.paymentSvc = paymentService.NewService()
	}

	return d.paymentSvc
}

func (d *diContainer) PaymentV1API(ctx context.Context) paymentv1.PaymentServiceServer {
	if d.paymentv1Handler == nil {
		d.paymentv1Handler = paymentApiV1.NewApi(d.PaymentService(ctx))
	}

	return d.paymentv1Handler
}
