-- name: ListEntriesByAccount :many
SELECT je.id, je.transaction_type, je.description, je.booking_date, je.created_at
FROM journal_entry je
JOIN journal_line jl ON jl.journal_entry_id = je.id
WHERE jl.ledger_account_id = $1
GROUP BY je.id
ORDER BY je.created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListLinesByEntry :many
SELECT ledger_account_id, direction, amount_minor, currency
FROM journal_line
WHERE journal_entry_id = $1;

-- name: GetEntry :one
SELECT id, transaction_type, description, booking_date
FROM journal_entry WHERE id = $1;
