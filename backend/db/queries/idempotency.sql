-- name: GetIdempotencyKey :one
SELECT key, request_hash, response_body, status_code FROM idempotency_key WHERE key = $1;

-- name: CreateIdempotencyKey :exec
INSERT INTO idempotency_key (key, request_hash, response_body, status_code)
VALUES ($1, $2, $3, $4);
