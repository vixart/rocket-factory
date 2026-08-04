package entity

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
)

type Part struct {
	uuid          uuid.UUID
	name          string
	description   string
	partType      valueobject.PartType
	price         int64
	stockQuantity int
	reserved      int
	properties    valueobject.PartProperties
	createdAt     time.Time
}

func (p *Part) UUID() uuid.UUID {
	return p.uuid
}

func (p *Part) Name() string {
	return p.name
}

func (p *Part) Description() string {
	return p.description
}

func (p *Part) PartType() valueobject.PartType {
	return p.partType
}

func (p *Part) Price() int64 {
	return p.price
}

func (p *Part) StockQuantity() int {
	return p.stockQuantity
}

func (p *Part) Reserved() int {
	return p.reserved
}

func (p *Part) Properties() valueobject.PartProperties {
	return p.properties
}

func (p *Part) CreatedAt() time.Time {
	return p.createdAt
}

func (p *Part) Reserve() error {
	slog.Debug("PART",
		"reserved", p.reserved,
		"stock", p.stockQuantity,
	)
	if p.stockQuantity-p.reserved <= 0 {
		return fmt.Errorf("cannot reserve part with id: \"%s\" : %w", p.uuid, errs.ErrOutOfStock)
	}

	p.reserved += 1

	return nil
}

func (p *Part) Release() error {
	if p.reserved <= 0 {
		return fmt.Errorf("cannot release part with id: \"%s\" : %w", p.uuid, errs.ErrNothingToRelease)
	}

	p.reserved -= 1

	return nil
}

func (p *Part) Commit() error {
	if p.reserved <= 0 {
		return fmt.Errorf("cannot commit part with id: \"%s\", it is not reserved: %w", p.uuid, errs.ErrNothingToCommit)
	}

	if p.stockQuantity <= 0 {
		return fmt.Errorf("cannot commit part with id: \"%s\", it is out of stock: %w", p.uuid, errs.ErrNothingToCommit)
	}

	p.reserved -= 1
	p.stockQuantity -= 1

	return nil
}

// RestorePart rebuilds the entity from the database (no validation: the data was already checked).
func RestorePart(partUUID uuid.UUID, name, description string, partType valueobject.PartType, price int64,
	stockQuantity, reserved int, properties valueobject.PartProperties, createdAt time.Time,
) Part {
	return Part{
		uuid:          partUUID,
		name:          name,
		description:   description,
		partType:      partType,
		price:         price,
		stockQuantity: stockQuantity,
		reserved:      reserved,
		properties:    properties,
		createdAt:     createdAt,
	}
}
