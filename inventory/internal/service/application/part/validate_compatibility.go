package part

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model/entity"
	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
	"github.com/vixart/rocket-factory/inventory/internal/service/input"
)

type slotRequest struct {
	name     string
	uuid     uuid.UUID
	partType valueobject.PartType
	required bool
}

func (s *service) ValidateCompatibility(ctx context.Context, slots valueobject.ShipSlots) error {
	resolved, err := s.resolveShipSlots(ctx, slots)
	if err != nil {
		return err
	}

	return s.compatibilityChecker.Check(resolved)
}

func (s *service) resolveShipSlots(ctx context.Context, slots valueobject.ShipSlots) (entity.ResolvedShipSlots, error) {
	requests := []slotRequest{
		{name: "hull", uuid: slots.HullUUID, partType: valueobject.PartTypeHull, required: true},
		{name: "engine", uuid: slots.EngineUUID, partType: valueobject.PartTypeEngine, required: true},
		{name: "shield", uuid: slots.ShieldUUID, partType: valueobject.PartTypeShield, required: false},
		{name: "weapon", uuid: slots.WeaponUUID, partType: valueobject.PartTypeWeapon, required: false},
	}

	uuids := make([]uuid.UUID, 0, len(requests))
	for _, request := range requests {
		if request.uuid == uuid.Nil {
			if request.required {
				return entity.ResolvedShipSlots{}, fmt.Errorf("%s_uuid обязателен: %w", request.name, errs.ErrInvalidUUID)
			}
			continue
		}
		uuids = append(uuids, request.uuid)
	}

	if err := validateDuplicateUUIDs(requests); err != nil {
		return entity.ResolvedShipSlots{}, err
	}

	parts, err := s.partRepo.List(ctx, input.PartFilter{UUIDs: uuids})
	if err != nil {
		slog.ErrorContext(ctx, "не удалось получить детали для проверки совместимости", "part_uuids", uuids, "error", err)
		return entity.ResolvedShipSlots{}, fmt.Errorf("получить детали: %w", err)
	}

	byUUID := make(map[uuid.UUID]entity.Part, len(parts))
	for _, part := range parts {
		byUUID[part.UUID()] = part
	}

	var resolved entity.ResolvedShipSlots

	for _, request := range requests {
		if request.uuid == uuid.Nil {
			continue
		}

		part, ok := byUUID[request.uuid]
		if !ok {
			return entity.ResolvedShipSlots{}, fmt.Errorf(
				"деталь %q для слота %s: %w",
				request.uuid.String(),
				request.name,
				errs.ErrPartNotFound,
			)
		}

		if part.PartType() != request.partType {
			return entity.ResolvedShipSlots{}, fmt.Errorf(
				"в слот %s передана деталь типа %s, ожидается %s: %w",
				request.name,
				part.PartType(),
				request.partType,
				errs.ErrPartTypeMismatch,
			)
		}

		switch request.partType {
		case valueobject.PartTypeHull:
			resolved.Hull = part
		case valueobject.PartTypeEngine:
			resolved.Engine = part
		case valueobject.PartTypeShield:
			resolved.Shield = &part
		case valueobject.PartTypeWeapon:
			resolved.Weapon = &part
		}
	}

	return resolved, nil
}

func validateDuplicateUUIDs(requests []slotRequest) error {
	seen := make(map[uuid.UUID]string, len(requests))

	for _, r := range requests {
		if r.uuid == uuid.Nil {
			continue
		}

		if prev, ok := seen[r.uuid]; ok {
			return fmt.Errorf(
				"uuid %s задублирован в слотах %s и %s: %w",
				r.uuid,
				prev,
				r.name,
				errs.ErrInvalidUUID,
			)
		}

		seen[r.uuid] = r.name
	}

	return nil
}
