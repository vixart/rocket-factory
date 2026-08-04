-- +goose Up
INSERT INTO parts (uuid, name, description, part_type, price, stock_quantity) VALUES
-- Hulls (price in kopecks)
('550e8400-e29b-41d4-a716-446655440001', 'Aluminium hull', 'Lightweight hull for small ships', 'HULL', 500000, 10),
('550e8400-e29b-41d4-a716-446655440002', 'Titanium hull', 'Durable hull for medium ships', 'HULL', 1500000, 5),
-- Engines
('550e8400-e29b-41d4-a716-446655440003', 'Ion engine C', 'Basic class C ion engine', 'ENGINE', 300000, 8),
('550e8400-e29b-41d4-a716-446655440004', 'Ion engine B', 'Improved class B ion engine', 'ENGINE', 800000, 3),
-- Shields
('550e8400-e29b-41d4-a716-446655440005', 'Energy shield', 'Standard energy shield', 'SHIELD', 400000, 6),
-- Weapons
('550e8400-e29b-41d4-a716-446655440006', 'Laser cannon', 'Precise laser cannon', 'WEAPON', 250000, 7),
-- Plasma hull (out of stock, used in tests)
('550e8400-e29b-41d4-a716-446655440007', 'Plasma hull', 'Experimental hull (out of stock)', 'HULL', 2000000, 0);

-- +goose Down
DELETE FROM parts WHERE uuid IN (
    '550e8400-e29b-41d4-a716-446655440001',
    '550e8400-e29b-41d4-a716-446655440002',
    '550e8400-e29b-41d4-a716-446655440003',
    '550e8400-e29b-41d4-a716-446655440004',
    '550e8400-e29b-41d4-a716-446655440005',
    '550e8400-e29b-41d4-a716-446655440006',
    '550e8400-e29b-41d4-a716-446655440007'
);