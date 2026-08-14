-- name: ListActiveDepositAccounts :many
SELECT a.id, a.ledger_account_id, a.currency, p.interest_rate_bps
FROM accounts a
JOIN product p ON p.id = a.product_id
WHERE a.status = 'active' AND p.kind = 'deposit';

-- name: CreateAccrual :exec
INSERT INTO interest_accrual (account_id, accrual_date, amount_minor, currency)
VALUES ($1, $2, $3, $4);

-- name: SumUncapitalizedAccruals :one
SELECT COALESCE(SUM(amount_minor), 0)::bigint
FROM interest_accrual WHERE account_id = $1 AND capitalized = false;

-- name: MarkAccrualsCapitalized :exec
UPDATE interest_accrual SET capitalized = true WHERE account_id = $1 AND capitalized = false;
