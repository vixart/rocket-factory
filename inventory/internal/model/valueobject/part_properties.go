package valueobject

// PartProperties holds the type-specific properties of a part.
// Exactly one field is non-nil, decided by the part type.
type PartProperties struct {
	hull   *HullProperties
	engine *EngineProperties
	shield *ShieldProperties
	weapon *WeaponProperties
}

func (p PartProperties) Hull() *HullProperties     { return p.hull }
func (p PartProperties) Engine() *EngineProperties { return p.engine }
func (p PartProperties) Shield() *ShieldProperties { return p.shield }
func (p PartProperties) Weapon() *WeaponProperties { return p.weapon }
