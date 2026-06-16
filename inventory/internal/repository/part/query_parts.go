package part

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model/entity"
	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
	"github.com/vixart/rocket-factory/inventory/internal/repository/converter"
	"github.com/vixart/rocket-factory/inventory/internal/repository/record"
	"github.com/vixart/rocket-factory/inventory/internal/service/input"
)

func (r *repository) list(
	ctx context.Context,
	partFilter input.PartFilter,
	forUpdate bool,
) ([]entity.Part, error) {
	builder := r.buildListQuery(partFilter, forUpdate)

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("не удалось сформировать запрос: %w", err)
	}

	rows, err := r.getter.DefaultTrOrDB(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("получить список деталей: %w", err)
	}
	defer rows.Close()

	records, err := pgx.CollectRows(rows, pgx.RowToStructByName[record.PartRecord])
	if err != nil {
		return nil, fmt.Errorf("считать строки: %w", err)
	}

	return r.mapAndOrder(records, partFilter)
}

func (r *repository) buildListQuery(
	partFilter input.PartFilter,
	forUpdate bool,
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
			"properties",
			"reserved",
		).
		From("parts").
		PlaceholderFormat(sq.Dollar)

	if forUpdate {
		builder = builder.Suffix("FOR UPDATE")
	}

	if len(partFilter.UUIDs) > 0 {
		return builder.Where(sq.Eq{
			"uuid": partFilter.UUIDs,
		})
	}

	if partFilter.PartType != valueobject.PartTypeUnspecified {
		builder = builder.Where(sq.Eq{
			"part_type": partFilter.PartType,
		})
	}

	return builder.OrderBy("name")
}

func (r *repository) mapAndOrder(
	records []record.PartRecord,
	partFilter input.PartFilter,
) ([]entity.Part, error) {
	if len(partFilter.UUIDs) == 0 {
		parts := make([]entity.Part, len(records))
		for i, rec := range records {
			part, err := converter.PartRecordToModel(rec)
			if err != nil {
				return nil, err
			}
			parts[i] = part
		}
		return parts, nil
	}

	recordByUUID := make(map[uuid.UUID]record.PartRecord, len(records))

	for _, rec := range records {
		recordByUUID[rec.UUID] = rec
	}

	if len(recordByUUID) != len(partFilter.UUIDs) {
		return nil, fmt.Errorf("найти детали: %w", errs.ErrPartNotFound)
	}

	ordered := make([]entity.Part, 0, len(partFilter.UUIDs))

	for _, id := range partFilter.UUIDs {
		rec, ok := recordByUUID[id]
		if !ok {
			return nil, fmt.Errorf("найти детали: %w", errs.ErrPartNotFound)
		}
		part, err := converter.PartRecordToModel(rec)
		if err != nil {
			return nil, err
		}
		ordered = append(ordered, part)
	}

	return ordered, nil
}
