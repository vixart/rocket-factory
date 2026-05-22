package part

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model"
	"github.com/vixart/rocket-factory/inventory/internal/repository/converter"
	"github.com/vixart/rocket-factory/inventory/internal/repository/record"
)

func (r *repository) List(
	ctx context.Context,
	partFilter model.PartFilter,
) ([]model.Part, error) {
	builder := sq.
		Select(
			"uuid",
			"name",
			"description",
			"part_type",
			"price",
			"stock_quantity",
			"created_at",
		).
		From("parts").
		PlaceholderFormat(sq.Dollar)

	if len(partFilter.Uuids) == 0 {
		builder = builder.OrderBy("name")
	}

	if len(partFilter.Uuids) > 0 {
		builder = builder.
			Where(sq.Eq{
				"uuid": partFilter.Uuids,
			}).
			OrderByClause(
				sq.Expr(
					"array_position(?::uuid[], uuid)",
					partFilter.Uuids,
				),
			)
	} else if partFilter.PartType != model.PartTypeUnspecified {
		builder = builder.Where(sq.Eq{
			"part_type": partFilter.PartType,
		})
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("не удалось сформировать запрос: %w", err)
	}

	rows, err := r.getter.DefaultTrOrDB(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("получить список деталей: %w", err)
	}

	defer rows.Close()

	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[record.Part])
	if err != nil {
		return nil, fmt.Errorf("считать строки: %w", err)
	}

	if len(partFilter.Uuids) > 0 && len(partFilter.Uuids) != len(records) {
		return nil, fmt.Errorf("найти детали: %w", errs.ErrPartNotFound)
	}

	parts := make([]model.Part, 0, len(records))

	for _, rec := range records {
		parts = append(parts, converter.PartRecordToModel(rec))
	}

	return parts, nil
}
