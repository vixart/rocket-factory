-- +goose Up
CREATE TABLE orders (
    uuid UUID PRIMARY KEY,
    status VARCHAR(50) NOT NULL,
    transaction_uuid UUID,
    payment_method VARCHAR(50),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS orders;