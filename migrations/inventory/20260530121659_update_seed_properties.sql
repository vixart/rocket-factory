-- +goose Up

-- Заполняем properties для seed-данных из миграции 00002.
-- Каждый тип детали получает свой набор свойств, используемых в ValidateCompatibility.

-- Формат JSONB: {"<тип>": {<свойства>}} — Pointer Union паттерн.
-- Ключ верхнего уровня (hull/engine/shield/weapon) определяет тип свойств.
-- json.Unmarshal в Go автоматически заполнит ровно одно поле в PartProperties.

-- Корпуса — strength определяет, какой класс двигателя выдержит корпус.
-- Алюминиевый (strength=50): потянет только класс C (требует ≥30).
-- Титановый (strength=150): потянет любой двигатель, включая класс A (требует ≥100).
-- Плазменный (strength=120, stock=0): тоже потянет любой двигатель, но используется в тестах out-of-stock.
UPDATE parts SET properties = '{"hull": {"strength": 50}}'  WHERE uuid = '550e8400-e29b-41d4-a716-446655440001';
UPDATE parts SET properties = '{"hull": {"strength": 150}}' WHERE uuid = '550e8400-e29b-41d4-a716-446655440002';
UPDATE parts SET properties = '{"hull": {"strength": 120}}' WHERE uuid = '550e8400-e29b-41d4-a716-446655440007';

-- Двигатели — class (A/B/C) и required_strength (минимальная прочность корпуса).
-- Класс C (required_strength=30) — лёгкий, подходит для любого корпуса.
-- Класс B (required_strength=70) — средний, алюминиевый корпус (50) не выдержит.
UPDATE parts SET properties = '{"engine": {"class": "C", "required_strength": 30}}'  WHERE uuid = '550e8400-e29b-41d4-a716-446655440003';
UPDATE parts SET properties = '{"engine": {"class": "B", "required_strength": 70}}'  WHERE uuid = '550e8400-e29b-41d4-a716-446655440004';

-- Щиты — shield_type: "energy" или "plasma".
-- Энергетический щит совместим с любым оружием.
-- Плазменный щит (не в seed, но возможен) конфликтует с лазерным оружием.
UPDATE parts SET properties = '{"shield": {"shield_type": "energy"}}' WHERE uuid = '550e8400-e29b-41d4-a716-446655440005';

-- Оружие — weapon_type: "laser" или "missile".
-- Лазер несовместим с плазменным щитом (электромагнитные помехи).
UPDATE parts SET properties = '{"weapon": {"weapon_type": "laser"}}' WHERE uuid = '550e8400-e29b-41d4-a716-446655440006';

-- +goose Down

-- Откат: сбрасываем properties обратно в пустой объект.
UPDATE parts SET properties = '{}' WHERE uuid IN (
    '550e8400-e29b-41d4-a716-446655440001',
    '550e8400-e29b-41d4-a716-446655440002',
    '550e8400-e29b-41d4-a716-446655440003',
    '550e8400-e29b-41d4-a716-446655440004',
    '550e8400-e29b-41d4-a716-446655440005',
    '550e8400-e29b-41d4-a716-446655440006',
    '550e8400-e29b-41d4-a716-446655440007'
);