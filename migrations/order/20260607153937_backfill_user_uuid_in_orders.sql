-- +goose Up
UPDATE orders
SET user_uuid = '00000000-0000-0000-0000-000000000000'
WHERE user_uuid IS NULL;

-- +goose Down
UPDATE orders
SET user_uuid = NULL
WHERE user_uuid = '00000000-0000-0000-0000-000000000000';