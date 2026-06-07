package valueobject

import "github.com/google/uuid"

type ShipSlots struct {
	HullUUID   uuid.UUID
	EngineUUID uuid.UUID
	ShieldUUID uuid.UUID
	WeaponUUID uuid.UUID
}
