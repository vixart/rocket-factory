package converter

import (
	"encoding/json"
	"fmt"

	"github.com/vixart/rocket-factory/inventory/internal/model/entity"
	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
	"github.com/vixart/rocket-factory/inventory/internal/repository/record"
)

func PartRecordToModel(rec record.PartRecord) (entity.Part, error) {
	var propsRec record.PartPropertiesRecord
	if err := json.Unmarshal(rec.Properties, &propsRec); err != nil {
		return entity.Part{}, fmt.Errorf("десериализовать свойства: %w", err)
	}

	props, err := partPropertiesFromRecord(propsRec)
	if err != nil {
		return entity.Part{}, fmt.Errorf("конвертировать свойства: %w", err)
	}

	partType, err := valueobject.NewPartType(rec.PartType)
	if err != nil {
		return entity.Part{}, fmt.Errorf("конвертировать тип детали: %w", err)
	}

	return entity.RestorePart(
		rec.UUID,
		rec.Name,
		rec.Description,
		partType,
		rec.Price,
		rec.StockQuantity,
		rec.Reserved,
		props,
		rec.CreatedAt,
	), nil
}

func partPropertiesFromRecord(rec record.PartPropertiesRecord) (valueobject.PartProperties, error) {
	switch {
	case rec.Hull != nil:
		return valueobject.NewHullProperties(rec.Hull.Strength)
	case rec.Engine != nil:
		return valueobject.NewEngineProperties(valueobject.EngineClass(rec.Engine.Class), rec.Engine.RequiredStrength)
	case rec.Shield != nil:
		return valueobject.NewShieldProperties(valueobject.ShieldType(rec.Shield.ShieldType))
	case rec.Weapon != nil:
		return valueobject.NewWeaponProperties(valueobject.WeaponType(rec.Weapon.WeaponType))
	default:
		return valueobject.PartProperties{}, nil
	}
}
