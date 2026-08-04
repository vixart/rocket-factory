-- +goose Up

-- Raise stock_quantity for the seeded parts.
-- With reservations in place (Reserve/Release) every created order lowers the available
-- stock (stock_quantity - reserved). With the original values (3–10) the stock runs out
-- after a handful of orders, which breaks the API tests.
-- The plasma hull (440007) keeps stock_quantity = 0 for the out-of-stock scenario.
UPDATE parts SET stock_quantity = 100 WHERE uuid IN (
    '550e8400-e29b-41d4-a716-446655440001',
    '550e8400-e29b-41d4-a716-446655440002',
    '550e8400-e29b-41d4-a716-446655440003',
    '550e8400-e29b-41d4-a716-446655440004',
    '550e8400-e29b-41d4-a716-446655440005',
    '550e8400-e29b-41d4-a716-446655440006'
);

-- +goose Down

-- Rollback: restore the original values from migration 00002.
UPDATE parts SET stock_quantity = 10 WHERE uuid = '550e8400-e29b-41d4-a716-446655440001';
UPDATE parts SET stock_quantity = 5  WHERE uuid = '550e8400-e29b-41d4-a716-446655440002';
UPDATE parts SET stock_quantity = 8  WHERE uuid = '550e8400-e29b-41d4-a716-446655440003';
UPDATE parts SET stock_quantity = 3  WHERE uuid = '550e8400-e29b-41d4-a716-446655440004';
UPDATE parts SET stock_quantity = 6  WHERE uuid = '550e8400-e29b-41d4-a716-446655440005';
UPDATE parts SET stock_quantity = 7  WHERE uuid = '550e8400-e29b-41d4-a716-446655440006';