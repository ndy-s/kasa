package transfer

import (
	"context"
	"errors"
	"sort"
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
	ErrSameAccount       = errors.New("cannot transfer to the same account")
)

type Service struct {
	pool   *pgxpool.Pool
	ledger *ledger.Service
}

func NewService(pool *pgxpool.Pool, ledgerSvc *ledger.Service) *Service {
	return &Service{pool: pool, ledger: ledgerSvc}
}

func (s *Service) Transfer(ctx context.Context, actor, fromID, toID string, amount money.Money) (string, error) {
	if fromID == toID {
		return "", ErrSameAccount
	}
	if _, err := uuid.Parse(fromID); err != nil {
		return "", err
	}
	if _, err := uuid.Parse(toID); err != nil {
		return "", err
	}

	pgTx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = pgTx.Rollback(ctx) }() // no-op after a successful commit

	tc := txn.Context{BusinessDate: time.Now(), Actor: actor, Tx: pgTx}
	q := postgres.New(tc.Tx)

	// lock both rows in a deterministic order to avoid deadlock against a
	// concurrent transfer running in the opposite direction, keeping the row
	// from this single locking pass instead of re-querying each account again
	ordered := []string{fromID, toID}
	sort.Strings(ordered)
	locked := make(map[string]postgres.GetAccountForUpdateRow, 2)
	for _, id := range ordered {
		u, err := uuid.Parse(id)
		if err != nil {
			return "", err
		}
		row, err := q.GetAccountForUpdate(ctx, pgtype.UUID{Bytes: u, Valid: true})
		if err != nil {
			return "", err
		}
		locked[id] = row
	}
	from, to := locked[fromID], locked[toID]

	balance, err := q.LedgerBalance(ctx, from.LedgerAccountID)
	if err != nil {
		return "", err
	}
	if balance < amount.Amount() {
		return "", ErrInsufficientFunds
	}

	lines, err := ledger.LinesFor(ledger.Transfer, ledger.PostingParams{
		Amount:        amount,
		FromAccountID: uuid.UUID(from.LedgerAccountID.Bytes).String(),
		ToAccountID:   uuid.UUID(to.LedgerAccountID.Bytes).String(),
	})
	if err != nil {
		return "", err
	}

	entryID, err := s.ledger.Post(ctx, tc.Tx, ledger.PostingRequest{
		Type: ledger.Transfer, Description: "transfer", BookingDate: tc.BusinessDate, Lines: lines,
	})
	if err != nil {
		return "", err
	}

	if err := tc.Tx.Commit(ctx); err != nil {
		return "", err
	}
	return entryID, nil
}
