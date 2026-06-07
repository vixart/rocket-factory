-- +goose Up

-- Увеличиваем stock_quantity для seed-деталей.
-- С появлением резервирования (Reserve/Release) каждый Create-заказ уменьшает доступный
-- остаток (stock_quantity - reserved). При оригинальных значениях (3–10) запас исчерпывается
-- после нескольких заказов, что ломает API-тесты.
-- Плазменный корпус (440007) остаётся с stock_quantity = 0 — для тестирования out-of-stock.
UPDATE parts SET stock_quantity = 100 WHERE uuid IN (
    '550e8400-e29b-41d4-a716-446655440001',
    '550e8400-e29b-41d4-a716-446655440002',
    '550e8400-e29b-41d4-a716-446655440003',
    '550e8400-e29b-41d4-a716-446655440004',
    '550e8400-e29b-41d4-a716-446655440005',
    '550e8400-e29b-41d4-a716-446655440006'
);

-- +goose Down

-- Откат: возвращаем оригинальные значения из миграции 00002.
UPDATE parts SET stock_quantity = 10 WHERE uuid = '550e8400-e29b-41d4-a716-446655440001';
UPDATE parts SET stock_quantity = 5  WHERE uuid = '550e8400-e29b-41d4-a716-446655440002';
UPDATE parts SET stock_quantity = 8  WHERE uuid = '550e8400-e29b-41d4-a716-446655440003';
UPDATE parts SET stock_quantity = 3  WHERE uuid = '550e8400-e29b-41d4-a716-446655440004';
UPDATE parts SET stock_quantity = 6  WHERE uuid = '550e8400-e29b-41d4-a716-446655440005';
UPDATE parts SET stock_quantity = 7  WHERE uuid = '550e8400-e29b-41d4-a716-446655440006';