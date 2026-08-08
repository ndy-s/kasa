-- +goose Up
CREATE TABLE credentials (
    customer_id UUID PRIMARY KEY REFERENCES customers(id) ON DELETE CASCADE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE credentials;
