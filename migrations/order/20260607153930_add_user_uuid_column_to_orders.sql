-- +goose Up
ALTER TABLE orders ADD COLUMN user_uuid UUID;

-- +goose Down
ALTER TABLE orders DROP COLUMN user_uuid;