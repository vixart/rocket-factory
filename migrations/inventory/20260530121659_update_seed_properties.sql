-- +goose Up

-- Fill in properties for the seed data of migration 00002.
-- Each part type gets the property set used by ValidateCompatibility.

-- JSONB format: {"<type>": {<properties>}} — the pointer union pattern.
-- The top-level key (hull/engine/shield/weapon) selects the property type.
-- json.Unmarshal in Go fills exactly one field of PartProperties.

-- Hulls: strength decides which engine class the hull can carry.
-- Aluminium (strength=50) only supports class C, which requires ≥30.
-- Titanium (strength=150) supports every engine, including class A, which requires ≥100.
-- Plasma (strength=120, stock=0) would support any engine too, but serves the out-of-stock tests.
UPDATE parts SET properties = '{"hull": {"strength": 50}}'  WHERE uuid = '550e8400-e29b-41d4-a716-446655440001';
UPDATE parts SET properties = '{"hull": {"strength": 150}}' WHERE uuid = '550e8400-e29b-41d4-a716-446655440002';
UPDATE parts SET properties = '{"hull": {"strength": 120}}' WHERE uuid = '550e8400-e29b-41d4-a716-446655440007';

-- Engines: class (A/B/C) and required_strength (the minimum hull strength).
-- Class C (required_strength=30) is light and fits any hull.
-- Class B (required_strength=70) is medium; the aluminium hull (50) cannot carry it.
UPDATE parts SET properties = '{"engine": {"class": "C", "required_strength": 30}}'  WHERE uuid = '550e8400-e29b-41d4-a716-446655440003';
UPDATE parts SET properties = '{"engine": {"class": "B", "required_strength": 70}}'  WHERE uuid = '550e8400-e29b-41d4-a716-446655440004';

-- Shields: shield_type is "energy" or "plasma".
-- The energy shield is compatible with every weapon.
-- A plasma shield (not seeded, but possible) conflicts with laser weapons.
UPDATE parts SET properties = '{"shield": {"shield_type": "energy"}}' WHERE uuid = '550e8400-e29b-41d4-a716-446655440005';

-- Weapons: weapon_type is "laser" or "missile".
-- A laser is incompatible with a plasma shield (electromagnetic interference).
UPDATE parts SET properties = '{"weapon": {"weapon_type": "laser"}}' WHERE uuid = '550e8400-e29b-41d4-a716-446655440006';

-- +goose Down

-- Rollback: reset properties back to an empty object.
UPDATE parts SET properties = '{}' WHERE uuid IN (
    '550e8400-e29b-41d4-a716-446655440001',
    '550e8400-e29b-41d4-a716-446655440002',
    '550e8400-e29b-41d4-a716-446655440003',
    '550e8400-e29b-41d4-a716-446655440004',
    '550e8400-e29b-41d4-a716-446655440005',
    '550e8400-e29b-41d4-a716-446655440006',
    '550e8400-e29b-41d4-a716-446655440007'
);