package part

import (
	"context"
	"fmt"

	"github.com/go-faster/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model"
	"github.com/vixart/rocket-factory/inventory/internal/repository/converter"
	"github.com/vixart/rocket-factory/inventory/internal/repository/record"
)

func (r *repository) Get(ctx context.Context, uuid uuid.UUID) (model.Part, error) {
	const query = `SELECT uuid, name, description, part_type, price, stock_quantity, created_at 
					FROM parts WHERE uuid = $1`

	var part record.Part

	err := r.getter.DefaultTrOrDB(ctx, r.pool).QueryRow(ctx, query, uuid).Scan(
		&part.UUID, &part.Name, &part.Description, &part.PartType, &part.Price, &part.StockQuantity, &part.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Part{}, fmt.Errorf("деталь не найдена в репозитории: %w", errs.ErrPartNotFound)
		}
		return model.Part{}, err
	}

	return converter.PartRecordToModel(part), nil
}
