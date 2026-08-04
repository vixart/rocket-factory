package order

import (
	"context"
	"fmt"
	"time"

	errs "github.com/vixart/rocket-factory/order/internal/errors"
	"github.com/vixart/rocket-factory/order/internal/model"
)

func (r *repository) Update(ctx context.Context, order model.Order) error {
	const query = `
		UPDATE orders
		SET
			status = $1,
			transaction_uuid = $2,
			payment_method = $3,
			updated_at = $4
		WHERE uuid = $5
	`

	result, err := r.getter.DefaultTrOrDB(ctx, r.pool).Exec(
		ctx,
		query,
		order.Status,
		order.TransactionUUID,
		order.PaymentMethod,
		time.Now(),
		order.UUID,
	)
	if err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	if result.RowsAffected() == 0 {
		return errs.ErrOrderNotFound
	}

	return nil
}
