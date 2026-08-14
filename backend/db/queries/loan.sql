-- name: GetLoanProductByCode :one
SELECT * FROM product WHERE code = $1 AND kind = 'loan';

-- name: CreateLoan :one
INSERT INTO loans (customer_id, deposit_account_id, principal_minor, term_months, interest_rate_bps, currency, disbursed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetLoanByID :one
SELECT * FROM loans WHERE id = $1;

-- name: UpdateLoanStatus :exec
UPDATE loans SET status = $2 WHERE id = $1;

-- name: CreateLoanInstallment :exec
INSERT INTO loan_installments (loan_id, installment_no, due_date, principal_minor, interest_minor, balance_minor)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListLoanInstallments :many
SELECT * FROM loan_installments WHERE loan_id = $1 ORDER BY installment_no;

-- name: NextDueInstallment :one
SELECT * FROM loan_installments WHERE loan_id = $1 AND status = 'due' ORDER BY installment_no LIMIT 1;

-- name: MarkInstallmentPaid :exec
UPDATE loan_installments SET status = 'paid', paid_at = now() WHERE id = $1;

-- name: CountDueInstallments :one
SELECT COUNT(*) FROM loan_installments WHERE loan_id = $1 AND status = 'due';
