package part

import (
	"context"

	"github.com/vixart/rocket-factory/inventory/internal/model/entity"
)

func (r *repository) UpdateReservedBatch(
	ctx context.Context,
	parts []entity.Part,
) error {
	const query = `
		UPDATE parts AS p
		SET reserved   = batch.reserved
		FROM unnest($1::uuid[], $2::int[]) AS batch(uuid, reserved)
		WHERE p.uuid = batch.uuid
	`

	uuids := make([]string, len(parts))
	reservedVals := make([]int, len(parts))

	for i, p := range parts {
		uuids[i] = p.UUID().String()
		reservedVals[i] = p.Reserved()
	}

	_, err := r.getter.DefaultTrOrDB(ctx, r.pool).Exec(ctx, query, uuids, reservedVals)
	if err != nil {
		return err
	}
	return nil
}
