-- +goose Up
CREATE TABLE users (
    uuid UUID PRIMARY KEY,
    login VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP
);

CREATE UNIQUE INDEX idx_users_login ON users(login);

-- +goose Down
DROP TABLE IF EXISTS users;