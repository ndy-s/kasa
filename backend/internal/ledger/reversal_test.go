package ledger_test

import (
	"context"
	"testing"
	"time"

	"github.com/ndy-s/kasa/backend/internal/ledger"
	"github.com/ndy-s/kasa/backend/internal/platform/postgres"
	"github.com/ndy-s/kasa/backend/internal/shared/money"
)

func TestReversalNetsToZero(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(t, ctx)

	q := postgres.New(pool)
	cash, err := q.GetAccountByCode(ctx, "1000")
	if err != nil {
		t.Fatalf("cash account: %v", err)
	}
	deposits, err := q.GetAccountByCode(ctx, "2000")
	if err != nil {
		t.Fatalf("deposits account: %v", err)
	}
	cashID := uuidString(cash.ID)
	depID := uuidString(deposits.ID)

	svc := ledger.NewService()

	lines, err := ledger.LinesFor(ledger.Deposit, ledger.PostingParams{
		Amount: money.FromMinor(10000, money.USD), CashAccountID: cashID, ToAccountID: depID,
	})
	if err != nil {
		t.Fatalf("rule: %v", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	entryID, err := svc.Post(ctx, tx, ledger.PostingRequest{
		Type: ledger.Deposit, BookingDate: time.Now(), Lines: lines,
	})
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	tx, err = pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin reversal: %v", err)
	}
	if _, err := svc.Reverse(ctx, tx, entryID); err != nil {
		t.Fatalf("reverse: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit reversal: %v", err)
	}

	for _, accID := range []string{cashID, depID} {
		var net int64
		row := pool.QueryRow(ctx, `
			SELECT COALESCE(SUM(CASE WHEN direction = 'debit' THEN amount_minor ELSE -amount_minor END), 0)
			FROM journal_line WHERE ledger_account_id = $1`, accID)
		if err := row.Scan(&net); err != nil {
			t.Fatalf("net query: %v", err)
		}
		if net != 0 {
			t.Fatalf("account %s does not net to zero after reversal: %d", accID, net)
		}
	}
}
