-- +goose Up

-- properties — JSONB-колонка для типоспецифичных свойств детали.
-- У каждого типа свой набор полей: у корпуса — strength, у двигателя — class и required_strength,
-- у щита — shield_type, у оружия — weapon_type. JSONB позволяет хранить их в одной таблице
-- без создания отдельных таблиц на каждый тип.
-- DEFAULT '{}' — новые детали без свойств получают пустой объект, а не NULL.
--
-- reserved — сколько единиц этой детали уже зарезервировано под заказы.
-- Доступно для новых заказов: stock_quantity - reserved.
ALTER TABLE parts
    ADD COLUMN properties JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN reserved INT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE parts
    DROP COLUMN IF EXISTS reserved,
    DROP COLUMN IF EXISTS properties;