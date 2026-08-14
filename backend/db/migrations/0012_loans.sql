-- +goose Up
INSERT INTO product (code, name, kind, currency, interest_rate_bps)
VALUES ('PL', 'Personal Loan', 'loan', 'IDR', 1800);

CREATE TABLE loans (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id        UUID NOT NULL REFERENCES customers(id),
    deposit_account_id UUID NOT NULL REFERENCES accounts(id),
    principal_minor    BIGINT NOT NULL CHECK (principal_minor > 0),
    term_months        INT NOT NULL CHECK (term_months > 0),
    interest_rate_bps  INT NOT NULL,
    currency           TEXT NOT NULL,
    status             TEXT NOT NULL DEFAULT 'active'
                       CHECK (status IN ('active', 'closed')),
    disbursed_at       TIMESTAMPTZ NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE loan_installments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    loan_id         UUID NOT NULL REFERENCES loans(id),
    installment_no  INT NOT NULL,
    due_date        DATE NOT NULL,
    principal_minor BIGINT NOT NULL,
    interest_minor  BIGINT NOT NULL,
    balance_minor   BIGINT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'due' CHECK (status IN ('due', 'paid')),
    paid_at         TIMESTAMPTZ,
    UNIQUE (loan_id, installment_no)
);

-- +goose Down
DROP TABLE loan_installments;
DROP TABLE loans;
DELETE FROM product WHERE code = 'PL';
