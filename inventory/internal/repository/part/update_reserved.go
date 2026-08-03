package part

import (
	"context"
	"log/slog"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model/entity"
)

func (r *repository) UpdateReservedBatch(
	ctx context.Context,
	parts []entity.Part,
) error {
	const query = `
		UPDATE parts AS p
		SET reserved = batch.reserved, stock_quantity = batch.stock_quantity
		FROM unnest($1::uuid[], $2::int[], $3::int[]) AS batch(uuid, reserved, stock_quantity)
		WHERE p.uuid = batch.uuid
	`

	uuids := make([]string, len(parts))
	reservedVals := make([]int, len(parts))
	stockQuantityVals := make([]int, len(parts))

	for i, p := range parts {
		uuids[i] = p.UUID().String()
		reservedVals[i] = p.Reserved()
		stockQuantityVals[i] = p.StockQuantity()
	}

	result, err := r.getter.DefaultTrOrDB(ctx, r.pool).Exec(ctx, query, uuids, reservedVals, stockQuantityVals)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return errs.ErrNoPartWereUpdated
	}

	slog.DebugContext(ctx, "резерв деталей обновлён в БД",
		"parts_count", len(parts), "rows_affected", result.RowsAffected())

	return nil
}
