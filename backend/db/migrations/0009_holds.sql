-- +goose Up
CREATE TABLE hold (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id   UUID NOT NULL REFERENCES accounts(id),
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    currency     TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'active'
                 CHECK (status IN ('active', 'captured', 'released', 'expired')),
    expires_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE hold;
