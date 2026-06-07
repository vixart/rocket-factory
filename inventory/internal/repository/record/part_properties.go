package record

// PartPropertiesRecord — структура для десериализации JSONB.
// Живёт в repository-слое: json-теги не принадлежат доменной модели.
type PartPropertiesRecord struct {
	Hull   *HullPropertiesRecord   `json:"hull,omitempty"`
	Engine *EnginePropertiesRecord `json:"engine,omitempty"`
	Shield *ShieldPropertiesRecord `json:"shield,omitempty"`
	Weapon *WeaponPropertiesRecord `json:"weapon,omitempty"`
}

type HullPropertiesRecord struct {
	Strength int `json:"strength"`
}

type EnginePropertiesRecord struct {
	Class            string `json:"class"`
	RequiredStrength int    `json:"required_strength"`
}

type ShieldPropertiesRecord struct {
	ShieldType string `json:"shield_type"`
}

type WeaponPropertiesRecord struct {
	WeaponType string `json:"weapon_type"`
}
