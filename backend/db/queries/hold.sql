-- name: CreateHold :one
INSERT INTO hold (account_id, amount_minor, currency) VALUES ($1, $2, $3) RETURNING id;

-- name: ReleaseHold :exec
UPDATE hold SET status = 'released' WHERE id = $1 AND status = 'active';

-- name: SumActiveHolds :one
SELECT COALESCE(SUM(amount_minor), 0)::bigint FROM hold
WHERE account_id = $1 AND status = 'active';
