-- name: CreateCustomer :one
INSERT INTO customers (name, email)
VALUES ($1, $2)
RETURNING *;

-- name: GetCustomerByID :one
SELECT * FROM customers WHERE id = $1;

-- name: GetCustomerByEmail :one
SELECT * FROM customers WHERE email = $1;

-- name: CreateCredential :exec
INSERT INTO credentials (customer_id, password_hash)
VALUES ($1, $2);

-- name: GetCredentialByEmail :one
SELECT c.id, cr.password_hash
FROM customers c
JOIN credentials cr ON cr.customer_id = c.id
WHERE c.email = $1;
