-- +goose Up
CREATE TABLE idempotency_key (
    key           TEXT PRIMARY KEY,
    request_hash  TEXT NOT NULL,
    response_body JSONB NOT NULL,
    status_code   INT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE idempotency_key;
