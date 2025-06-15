-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY, -- FOR THE FUTURE MAY BE CHANGED TO UUID
    telegram_id BIGINT UNIQUE NOT NULL,
    username TEXT,
    created_at TIMESTAMP DEFAULT now()
);

CREATE TABLE IF NOT EXISTS portfolios (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT now()
);


CREATE TABLE IF NOT EXISTS transactions (
    id BIGSERIAL PRIMARY KEY,
    portfolio_id BIGINT NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN ('buy', 'sell', 'transfer')),
    pair TEXT NOT NULL,
    asset_amount NUMERIC(18,8) NOT NULL,
    asset_price NUMERIC(18,8), NOT NULL,
    amount_usd NUMERIC(12,2) NOT NULL,
    transaction_date TIMESTAMP NOT NULL,
    note TEXT,
    created_at TIMESTAMP DEFAULT now()
);


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS portfolios;
DROP TABLE IF EXISTS users;

-- +goose StatementEnd
