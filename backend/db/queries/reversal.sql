-- name: CreateReversingEntry :one
INSERT INTO journal_entry (transaction_type, description, booking_date, value_date, reverses_entry_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;
