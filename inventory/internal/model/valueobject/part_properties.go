package valueobject

// PartProperties — типоспецифичные свойства детали.
// Ровно одно поле non-nil — определяется типом детали.
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
