-- +goose Up
CREATE TABLE product (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code              TEXT NOT NULL UNIQUE,
    name              TEXT NOT NULL,
    kind              TEXT NOT NULL CHECK (kind IN ('deposit', 'loan')),
    currency          TEXT NOT NULL,
    interest_rate_bps INT NOT NULL DEFAULT 0,
    config            JSONB NOT NULL DEFAULT '{}',
    active            BOOLEAN NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE accounts (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id       UUID NOT NULL REFERENCES customers(id),
    product_id        UUID NOT NULL REFERENCES product(id),
    currency          TEXT NOT NULL,
    status            TEXT NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending', 'active', 'frozen', 'closed')),
    ledger_account_id UUID NOT NULL REFERENCES chart_of_accounts(id),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE accounts;
DROP TABLE product;
