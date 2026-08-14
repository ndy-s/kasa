-- +goose Up
CREATE TABLE interest_accrual (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id   UUID NOT NULL REFERENCES accounts(id),
    accrual_date DATE NOT NULL,
    amount_minor BIGINT NOT NULL,
    currency     TEXT NOT NULL,
    capitalized  BOOLEAN NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE interest_accrual;
