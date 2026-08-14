-- name: CreateLedgerAccount :one
INSERT INTO chart_of_accounts (code, name, type, currency)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: CreateAccount :one
INSERT INTO accounts (customer_id, product_id, currency, status, ledger_account_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetAccountByID :one
SELECT * FROM accounts WHERE id = $1;

-- name: ListAccountsByCustomer :many
SELECT * FROM accounts WHERE customer_id = $1 ORDER BY created_at;

-- name: UpdateAccountStatus :exec
UPDATE accounts SET status = $2, updated_at = now() WHERE id = $1;

-- name: GetProductByCode :one
SELECT * FROM product WHERE code = $1;
