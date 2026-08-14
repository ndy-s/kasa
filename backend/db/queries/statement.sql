-- name: BalanceAsOf :one
SELECT COALESCE(SUM(
  CASE WHEN jl.direction = 'credit' THEN jl.amount_minor ELSE -jl.amount_minor END
), 0)::bigint
FROM journal_line jl
JOIN journal_entry je ON je.id = jl.journal_entry_id
WHERE jl.ledger_account_id = $1 AND je.booking_date < sqlc.arg(before)::date;

-- name: LinesInPeriod :many
SELECT je.booking_date, je.transaction_type, jl.direction, jl.amount_minor, jl.currency
FROM journal_line jl
JOIN journal_entry je ON je.id = jl.journal_entry_id
WHERE jl.ledger_account_id = $1
  AND je.booking_date >= sqlc.arg(from_date)::date
  AND je.booking_date < sqlc.arg(to_date)::date
ORDER BY je.booking_date;
