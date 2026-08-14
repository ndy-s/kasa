package loan

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ndy-s/kasa/backend/internal/ledger"
	"github.com/ndy-s/kasa/backend/internal/platform/postgres"
	"github.com/ndy-s/kasa/backend/internal/shared/money"
)

var (
	ErrLoanNotFound      = errors.New("loan not found")
	ErrDepositNotActive  = errors.New("deposit account is not active")
	ErrNoInstallmentDue  = errors.New("loan has no installment due")
	ErrInsufficientFunds = errors.New("insufficient funds")
)

type Service struct {
	pool   *pgxpool.Pool
	q      *postgres.Queries
	ledger *ledger.Service
}

func NewService(pool *pgxpool.Pool, ledgerSvc *ledger.Service) *Service {
	return &Service{pool: pool, q: postgres.New(pool), ledger: ledgerSvc}
}

// Originate opens a loan for the customer, builds its amortization schedule, and disburses the
// principal into the given deposit account: Dr Loans Receivable / Cr the deposit. All in one transaction.
func (s *Service) Originate(
	ctx context.Context, customerID, productCode, depositAccountID string,
	principal money.Money, termMonths int,
) (*Loan, error) {
	custID, err := uuid.Parse(customerID)
	if err != nil {
		return nil, err
	}
	depID, err := uuid.Parse(depositAccountID)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful commit
	qtx := s.q.WithTx(tx)

	product, err := qtx.GetLoanProductByCode(ctx, productCode)
	if err != nil {
		return nil, err
	}

	deposit, err := qtx.GetAccountForUpdate(ctx, pgtype.UUID{Bytes: depID, Valid: true})
	if err != nil {
		return nil, err
	}
	if uuid.UUID(deposit.CustomerID.Bytes).String() != customerID || deposit.Status != "active" {
		return nil, ErrDepositNotActive
	}

	disbursedAt := time.Now()
	loanRow, err := qtx.CreateLoan(ctx, postgres.CreateLoanParams{
		CustomerID:       pgtype.UUID{Bytes: custID, Valid: true},
		DepositAccountID: pgtype.UUID{Bytes: depID, Valid: true},
		PrincipalMinor:   principal.Amount(),
		TermMonths:       int32(termMonths),
		InterestRateBps:  product.InterestRateBps,
		Currency:         product.Currency,
		DisbursedAt:      pgtype.Timestamptz{Time: disbursedAt, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	schedule, err := Schedule(principal, int64(product.InterestRateBps), termMonths, disbursedAt)
	if err != nil {
		return nil, err
	}
	for _, inst := range schedule {
		if err := qtx.CreateLoanInstallment(ctx, postgres.CreateLoanInstallmentParams{
			LoanID:         loanRow.ID,
			InstallmentNo:  int32(inst.Number),
			DueDate:        pgtype.Date{Time: inst.DueDate, Valid: true},
			PrincipalMinor: inst.Principal.Amount(),
			InterestMinor:  inst.Interest.Amount(),
			BalanceMinor:   inst.Balance.Amount(),
		}); err != nil {
			return nil, err
		}
	}

	receivable, err := qtx.GetAccountByCode(ctx, "1100") // Loans Receivable
	if err != nil {
		return nil, err
	}
	lines, err := ledger.LinesFor(ledger.Disbursement, ledger.PostingParams{
		Amount:        principal,
		CashAccountID: uuid.UUID(receivable.ID.Bytes).String(),
		ToAccountID:   uuid.UUID(deposit.LedgerAccountID.Bytes).String(),
	})
	if err != nil {
		return nil, err
	}
	if _, err := s.ledger.Post(ctx, tx, ledger.PostingRequest{
		Type: ledger.Disbursement, Description: "loan disbursement", BookingDate: disbursedAt, Lines: lines,
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return toDomain(loanRow), nil
}

func (s *Service) Get(ctx context.Context, loanID string) (*Loan, error) {
	id, err := uuid.Parse(loanID)
	if err != nil {
		return nil, ErrLoanNotFound
	}
	row, err := s.q.GetLoanByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return nil, ErrLoanNotFound
	}
	return toDomain(row), nil
}

// Installments returns the full amortization schedule as stored, each installment's paid/due status included.
func (s *Service) Installments(ctx context.Context, loanID string) ([]Installment, error) {
	id, err := uuid.Parse(loanID)
	if err != nil {
		return nil, ErrLoanNotFound
	}
	loanRow, err := s.q.GetLoanByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return nil, ErrLoanNotFound
	}
	cur, err := money.ForCode(loanRow.Currency)
	if err != nil {
		return nil, err
	}
	rows, err := s.q.ListLoanInstallments(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return nil, err
	}
	out := make([]Installment, len(rows))
	for i, r := range rows {
		out[i] = Installment{
			Number:    int(r.InstallmentNo),
			DueDate:   r.DueDate.Time,
			Principal: money.FromMinor(r.PrincipalMinor, cur),
			Interest:  money.FromMinor(r.InterestMinor, cur),
			Balance:   money.FromMinor(r.BalanceMinor, cur),
			Status:    r.Status,
		}
	}
	return out, nil
}

// Repay pays the next due installment for a loan out of its deposit account: Dr the deposit for
// principal+interest / Cr Loans Receivable (principal) / Cr Interest Income (interest).
func (s *Service) Repay(ctx context.Context, actorCustomerID, loanID string) (string, error) {
	lid, err := uuid.Parse(loanID)
	if err != nil {
		return "", ErrLoanNotFound
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful commit
	qtx := s.q.WithTx(tx)

	loanRow, err := qtx.GetLoanByID(ctx, pgtype.UUID{Bytes: lid, Valid: true})
	if err != nil {
		return "", ErrLoanNotFound
	}
	if uuid.UUID(loanRow.CustomerID.Bytes).String() != actorCustomerID {
		return "", ErrLoanNotFound
	}
	if loanRow.Status != string(StatusActive) {
		return "", ErrNoInstallmentDue
	}

	inst, err := qtx.NextDueInstallment(ctx, pgtype.UUID{Bytes: lid, Valid: true})
	if err != nil {
		return "", ErrNoInstallmentDue
	}

	deposit, err := qtx.GetAccountForUpdate(ctx, loanRow.DepositAccountID)
	if err != nil {
		return "", err
	}
	if deposit.Status != "active" {
		return "", ErrDepositNotActive
	}

	cur, err := money.ForCode(loanRow.Currency)
	if err != nil {
		return "", err
	}
	principal := money.FromMinor(inst.PrincipalMinor, cur)
	interest := money.FromMinor(inst.InterestMinor, cur)
	total, err := principal.Add(interest)
	if err != nil {
		return "", err
	}

	balance, err := qtx.LedgerBalance(ctx, deposit.LedgerAccountID)
	if err != nil {
		return "", err
	}
	if balance < total.Amount() {
		return "", ErrInsufficientFunds
	}

	receivable, err := qtx.GetAccountByCode(ctx, "1100") // Loans Receivable
	if err != nil {
		return "", err
	}
	income, err := qtx.GetAccountByCode(ctx, "4000") // Interest Income
	if err != nil {
		return "", err
	}

	lines, err := ledger.LinesFor(ledger.Repayment, ledger.PostingParams{
		Amount:          principal,
		FromAccountID:   uuid.UUID(deposit.LedgerAccountID.Bytes).String(),
		ToAccountID:     uuid.UUID(receivable.ID.Bytes).String(),
		SecondAmount:    interest,
		SecondAccountID: uuid.UUID(income.ID.Bytes).String(),
	})
	if err != nil {
		return "", err
	}

	entryID, err := s.ledger.Post(ctx, tx, ledger.PostingRequest{
		Type: ledger.Repayment, Description: "loan repayment", BookingDate: time.Now(), Lines: lines,
	})
	if err != nil {
		return "", err
	}

	if err := qtx.MarkInstallmentPaid(ctx, inst.ID); err != nil {
		return "", err
	}

	remaining, err := qtx.CountDueInstallments(ctx, pgtype.UUID{Bytes: lid, Valid: true})
	if err != nil {
		return "", err
	}
	if remaining == 0 {
		if err := qtx.UpdateLoanStatus(ctx, postgres.UpdateLoanStatusParams{
			ID: pgtype.UUID{Bytes: lid, Valid: true}, Status: string(StatusClosed),
		}); err != nil {
			return "", err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return entryID, nil
}

func toDomain(row postgres.Loan) *Loan {
	return &Loan{
		ID:               uuid.UUID(row.ID.Bytes).String(),
		CustomerID:       uuid.UUID(row.CustomerID.Bytes).String(),
		DepositAccountID: uuid.UUID(row.DepositAccountID.Bytes).String(),
		PrincipalMinor:   row.PrincipalMinor,
		TermMonths:       int(row.TermMonths),
		InterestRateBps:  row.InterestRateBps,
		Currency:         row.Currency,
		Status:           Status(row.Status),
		DisbursedAt:      row.DisbursedAt.Time,
	}
}
