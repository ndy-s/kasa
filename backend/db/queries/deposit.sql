-- name: GetAccountForUpdate :one
SELECT id, customer_id, product_id, currency, status, ledger_account_id
FROM accounts WHERE id = $1 FOR UPDATE;
