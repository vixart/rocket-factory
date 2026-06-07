package part

import (
	"context"
	"fmt"

	"github.com/go-faster/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model/entity"
	"github.com/vixart/rocket-factory/inventory/internal/repository/converter"
	"github.com/vixart/rocket-factory/inventory/internal/repository/record"
)

func (r *repository) Get(ctx context.Context, uuid uuid.UUID) (entity.Part, error) {
	const query = `SELECT uuid, name, description, part_type, price, stock_quantity, properties, reserved, created_at
					FROM parts WHERE uuid = $1`

	var part record.PartRecord

	pgxRow := r.getter.DefaultTrOrDB(ctx, r.pool).QueryRow(ctx, query, uuid)
	err := pgxRow.Scan(
		&part.UUID, &part.Name, &part.Description, &part.PartType, &part.Price, &part.StockQuantity, &part.Properties, &part.Reserved, &part.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Part{}, fmt.Errorf("деталь не найдена в репозитории: %w", errs.ErrPartNotFound)
		}
		return entity.Part{}, err
	}

	return converter.PartRecordToModel(part)
}
