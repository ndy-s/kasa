package interest

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ndy-s/kasa/backend/internal/ledger"
	"github.com/ndy-s/kasa/backend/internal/platform/postgres"
	"github.com/ndy-s/kasa/backend/internal/shared/money"
)

type Service struct {
	pool   *pgxpool.Pool
	ledger *ledger.Service
	q      *postgres.Queries
}

func NewService(pool *pgxpool.Pool, ledgerSvc *ledger.Service) *Service {
	return &Service{pool: pool, ledger: ledgerSvc, q: postgres.New(pool)}
}

// Accrue computes one day of interest for every active deposit account.
func (s *Service) Accrue(ctx context.Context, businessDate time.Time) error {
	accounts, err := s.q.ListActiveDepositAccounts(ctx)
	if err != nil {
		return err
	}
	for _, a := range accounts {
		raw, err := s.q.LedgerBalance(ctx, a.LedgerAccountID)
		if err != nil {
			return err
		}
		cur, err := money.ForCode(a.Currency)
		if err != nil {
			return err
		}
		daily := money.FromMinor(raw, cur).InterestForDays(int64(a.InterestRateBps), 1)
		if daily.IsZero() {
			continue
		}
		err = s.q.CreateAccrual(ctx, postgres.CreateAccrualParams{
			AccountID:   a.ID,
			AccrualDate: pgtype.Date{Time: businessDate, Valid: true},
			AmountMinor: daily.Amount(),
			Currency:    a.Currency,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// CapitalizeAll runs Capitalize for every active deposit account.
func (s *Service) CapitalizeAll(ctx context.Context, businessDate time.Time) error {
	accounts, err := s.q.ListActiveDepositAccounts(ctx)
	if err != nil {
		return err
	}
	for _, a := range accounts {
		accountID := uuid.UUID(a.ID.Bytes).String()
		ledgerAccountID := uuid.UUID(a.LedgerAccountID.Bytes).String()
		if err := s.Capitalize(ctx, accountID, ledgerAccountID, a.Currency, businessDate); err != nil {
			return err
		}
	}
	return nil
}

// Capitalize posts the accumulated accruals for one account: Dr Interest Expense, Cr the deposit.
func (s *Service) Capitalize(ctx context.Context, accountID, ledgerAccountID, currency string, businessDate time.Time) error {
	aid, err := uuid.Parse(accountID)
	if err != nil {
		return err
	}
	raw, err := s.q.SumUncapitalizedAccruals(ctx, pgtype.UUID{Bytes: aid, Valid: true})
	if err != nil {
		return err
	}
	if raw == 0 {
		return nil
	}
	cur, err := money.ForCode(currency)
	if err != nil {
		return err
	}

	expense, err := s.q.GetAccountByCode(ctx, "5000") // Interest Expense
	if err != nil {
		return err
	}

	lines, err := ledger.LinesFor(ledger.InterestCredit, ledger.PostingParams{
		Amount:        money.FromMinor(raw, cur),
		FromAccountID: uuid.UUID(expense.ID.Bytes).String(),
		ToAccountID:   ledgerAccountID,
	})
	if err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful commit

	if _, err := s.ledger.Post(ctx, tx, ledger.PostingRequest{
		Type: ledger.InterestCredit, Description: "interest capitalization", BookingDate: businessDate, Lines: lines,
	}); err != nil {
		return err
	}
	if err := s.q.WithTx(tx).MarkAccrualsCapitalized(ctx, pgtype.UUID{Bytes: aid, Valid: true}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
