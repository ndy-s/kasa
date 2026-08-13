-- +goose Up
CREATE TABLE chart_of_accounts (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code     TEXT NOT NULL UNIQUE,
    name     TEXT NOT NULL,
    type     TEXT NOT NULL CHECK (type IN ('asset', 'liability', 'equity', 'income', 'expense')),
    currency TEXT NOT NULL
);

CREATE TABLE journal_entry (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_type  TEXT NOT NULL,
    description       TEXT NOT NULL DEFAULT '',
    idempotency_key   TEXT,
    booking_date      DATE NOT NULL,
    value_date        DATE NOT NULL,
    reference_no      TEXT,
    reverses_entry_id UUID REFERENCES journal_entry(id),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE journal_line (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    journal_entry_id  UUID NOT NULL REFERENCES journal_entry(id) ON DELETE CASCADE,
    ledger_account_id UUID NOT NULL REFERENCES chart_of_accounts(id),
    direction         TEXT NOT NULL CHECK (direction IN ('debit', 'credit')),
    amount_minor      BIGINT NOT NULL CHECK (amount_minor > 0),
    currency          TEXT NOT NULL
);

CREATE INDEX idx_journal_line_account ON journal_line(ledger_account_id);
CREATE INDEX idx_journal_line_entry ON journal_line(journal_entry_id);

-- +goose Down
DROP TABLE journal_line;
DROP TABLE journal_entry;
DROP TABLE chart_of_accounts;
