package part

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
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

	builder := r.buildListQuery(partFilter)

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

	return r.mapAndOrder(records, partFilter)
}

func (r *repository) buildListQuery(
	partFilter model.PartFilter,
) sq.SelectBuilder {

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

	if len(partFilter.Uuids) > 0 {
		return builder.Where(sq.Eq{
			"uuid": partFilter.Uuids,
		})
	}

	if partFilter.PartType != model.PartTypeUnspecified {
		builder = builder.Where(sq.Eq{
			"part_type": partFilter.PartType,
		})
	}

	return builder.OrderBy("name")
}

func (r *repository) mapAndOrder(
	records []record.Part,
	partFilter model.PartFilter,
) ([]model.Part, error) {

	if len(partFilter.Uuids) == 0 {
		parts := make([]model.Part, len(records))
		for i, rec := range records {
			parts[i] = converter.PartRecordToModel(rec)
		}
		return parts, nil
	}

	recordByUUID := make(map[uuid.UUID]record.Part, len(records))

	for _, rec := range records {
		recordByUUID[rec.UUID] = rec
	}

	if len(recordByUUID) != len(partFilter.Uuids) {
		return nil, fmt.Errorf("найти детали: %w", errs.ErrPartNotFound)
	}

	ordered := make([]model.Part, 0, len(partFilter.Uuids))

	for _, id := range partFilter.Uuids {
		rec, ok := recordByUUID[id]
		if !ok {
			return nil, fmt.Errorf("найти детали: %w", errs.ErrPartNotFound)
		}

		ordered = append(ordered, converter.PartRecordToModel(rec))
	}

	return ordered, nil
}
