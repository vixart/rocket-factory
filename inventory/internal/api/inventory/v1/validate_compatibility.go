package v1

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	errs "github.com/vixart/rocket-factory/inventory/internal/errors"
	"github.com/vixart/rocket-factory/inventory/internal/model/valueobject"
	inventoryv1 "github.com/vixart/rocket-factory/shared/pkg/proto/inventory/v1"
)

func (a *api) ValidateCompatibility(
	ctx context.Context,
	req *inventoryv1.ValidateCompatibilityRequest,
) (*inventoryv1.ValidateCompatibilityResponse, error) {
	shipSlots, err := composeShipSlots(req)
	if err != nil {
		return nil, err
	}

	err = a.inventoryService.ValidateCompatibility(ctx, shipSlots)
	if err != nil {
		return nil, err
	}
	return &inventoryv1.ValidateCompatibilityResponse{}, nil
}

func composeShipSlots(req *inventoryv1.ValidateCompatibilityRequest) (valueobject.ShipSlots, error) {
	hullUUID, err := parseUUID(req.GetHullUuid())
	if err != nil {
		return valueobject.ShipSlots{}, err
	}

	engineUUID, err := parseUUID(req.GetEngineUuid())
	if err != nil {
		return valueobject.ShipSlots{}, err
	}

	shipSlots := valueobject.ShipSlots{
		HullUUID:   hullUUID,
		EngineUUID: engineUUID,
	}

	if req.GetShieldUuid() != "" {
		shipSlots.ShieldUUID, err = parseUUID(req.GetShieldUuid())
		if err != nil {
			return valueobject.ShipSlots{}, err
		}
	}

	if req.GetWeaponUuid() != "" {
		shipSlots.WeaponUUID, err = parseUUID(req.GetWeaponUuid())
		if err != nil {
			return valueobject.ShipSlots{}, err
		}
	}

	return shipSlots, nil
}

func parseUUID(raw string) (uuid.UUID, error) {
	parsedUUID, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf(
			"invalid uuid format: %s, %w",
			raw,
			errs.ErrInvalidUUID,
		)
	}

	return parsedUUID, nil
}
