package deposit

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ndy-s/kasa/backend/internal/ledger"
	"github.com/ndy-s/kasa/backend/internal/platform/postgres"
	"github.com/ndy-s/kasa/backend/internal/platform/txn"
	"github.com/ndy-s/kasa/backend/internal/shared/money"
)

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrAccountNotActive  = errors.New("account is not active")
	ErrInvalidAmount     = errors.New("amount must be positive")
)

type Service struct {
	pool   *pgxpool.Pool
	ledger *ledger.Service
}

func NewService(pool *pgxpool.Pool, ledgerSvc *ledger.Service) *Service {
	return &Service{pool: pool, ledger: ledgerSvc}
}

func (s *Service) Deposit(ctx context.Context, actor, accountID string, amount money.Money) (string, error) {
	return s.move(ctx, actor, accountID, amount, ledger.Deposit, false)
}

func (s *Service) Withdraw(ctx context.Context, actor, accountID string, amount money.Money) (string, error) {
	return s.move(ctx, actor, accountID, amount, ledger.Withdrawal, true)
}

func (s *Service) move(
	ctx context.Context, actor, accountID string, amount money.Money,
	txType ledger.TransactionType, checkFunds bool,
) (string, error) {
	if !amount.IsPositive() {
		return "", ErrInvalidAmount
	}
	aid, err := uuid.Parse(accountID)
	if err != nil {
		return "", err
	}

	pgTx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = pgTx.Rollback(ctx) }() // no-op after a successful commit

	tc := txn.Context{BusinessDate: time.Now(), Actor: actor, Tx: pgTx}
	q := postgres.New(tc.Tx)

	acc, err := q.GetAccountForUpdate(ctx, pgtype.UUID{Bytes: aid, Valid: true})
	if err != nil {
		return "", err
	}
	if acc.Status != "active" {
		return "", ErrAccountNotActive
	}

	if checkFunds {
		balance, err := q.LedgerBalance(ctx, acc.LedgerAccountID)
		if err != nil {
			return "", err
		}
		if balance < amount.Amount() {
			return "", ErrInsufficientFunds
		}
	}

	cash, err := q.GetAccountByCode(ctx, "1000")
	if err != nil {
		return "", err
	}

	params := ledger.PostingParams{
		Amount:        amount,
		CashAccountID: uuid.UUID(cash.ID.Bytes).String(),
	}
	accountGL := uuid.UUID(acc.LedgerAccountID.Bytes).String()
	if txType == ledger.Deposit {
		params.ToAccountID = accountGL
	} else {
		params.FromAccountID = accountGL
	}

	lines, err := ledger.LinesFor(txType, params)
	if err != nil {
		return "", err
	}

	entryID, err := s.ledger.Post(ctx, tc.Tx, ledger.PostingRequest{
		Type:        txType,
		Description: string(txType),
		BookingDate: tc.BusinessDate,
		Lines:       lines,
	})
	if err != nil {
		return "", err
	}

	if err := tc.Tx.Commit(ctx); err != nil {
		return "", err
	}
	return entryID, nil
}
