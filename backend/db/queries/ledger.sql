-- name: GetAccountByCode :one
SELECT id, code, name, type, currency FROM chart_of_accounts WHERE code = $1;

-- name: CreateJournalEntry :one
INSERT INTO journal_entry (transaction_type, description, booking_date, value_date)
VALUES ($1, $2, $3, $4)
RETURNING id;

-- name: CreateJournalLine :exec
INSERT INTO journal_line (journal_entry_id, ledger_account_id, direction, amount_minor, currency)
VALUES ($1, $2, $3, $4, $5);

-- name: LedgerBalance :one
SELECT COALESCE(SUM(
  CASE WHEN direction = 'credit' THEN amount_minor ELSE -amount_minor END
), 0)::bigint AS balance
FROM journal_line
WHERE ledger_account_id = $1;
