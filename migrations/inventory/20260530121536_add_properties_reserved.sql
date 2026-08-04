-- +goose Up

-- properties is a JSONB column holding the type-specific properties of a part.
-- Each type has its own fields: strength for a hull, class and required_strength for an
-- engine, shield_type for a shield, weapon_type for a weapon. JSONB keeps them all in one
-- table instead of a table per type.
-- DEFAULT '{}' gives new parts without properties an empty object rather than NULL.
--
-- reserved is how many units of the part are already reserved for orders.
-- Available for new orders: stock_quantity - reserved.
ALTER TABLE parts
    ADD COLUMN properties JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN reserved INT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE parts
    DROP COLUMN IF EXISTS reserved,
    DROP COLUMN IF EXISTS properties;