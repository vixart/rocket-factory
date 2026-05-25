-- +goose Up
CREATE TABLE order_items (
    order_uuid UUID NOT NULL REFERENCES orders(uuid),
    part_uuid UUID NOT NULL,
    part_type VARCHAR(20) NOT NULL,
    price BIGINT NOT NULL,
    PRIMARY KEY (order_uuid, part_uuid)
);

-- +goose Down
DROP TABLE IF EXISTS order_items;