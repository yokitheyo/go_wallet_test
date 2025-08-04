-- +goose Up
CREATE TABLE IF NOT EXISTS wallets (
    wallet_id UUID PRIMARY KEY,
    balance BIGINT NOT NULL
);
-- +goose Down
DROP TABLE IF EXISTS wallets;