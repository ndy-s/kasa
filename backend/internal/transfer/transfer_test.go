package transfer_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ndy-s/kasa/backend/internal/account"
	"github.com/ndy-s/kasa/backend/internal/customer"
	"github.com/ndy-s/kasa/backend/internal/deposit"
	"github.com/ndy-s/kasa/backend/internal/ledger"
	"github.com/ndy-s/kasa/backend/internal/platform/auth"
	"github.com/ndy-s/kasa/backend/internal/shared/money"
	"github.com/ndy-s/kasa/backend/internal/transfer"
)

// A negative transfer amount flips both journal lines: it would debit (not credit) the "to" account
// and credit (not debit) the "from" account, draining the destination with no funds check on it at all.
// ErrInvalidAmount closes that off.
func TestTransferRejectsNegativeAmount(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(t, ctx)

	custSvc := customer.NewService(customer.NewPgRepository(pool), auth.NewTokenIssuer("test-secret", time.Hour))
	attacker, err := custSvc.Register(ctx, "Attacker", "attacker@example.com", "s3cret-password")
	if err != nil {
		t.Fatalf("register attacker: %v", err)
	}
	victim, err := custSvc.Register(ctx, "Victim", "victim@example.com", "s3cret-password")
	if err != nil {
		t.Fatalf("register victim: %v", err)
	}

	accSvc := account.NewService(pool)
	attackerAcc, err := accSvc.OpenAccount(ctx, attacker.ID, "SAV")
	if err != nil {
		t.Fatalf("open attacker account: %v", err)
	}
	victimAcc, err := accSvc.OpenAccount(ctx, victim.ID, "SAV")
	if err != nil {
		t.Fatalf("open victim account: %v", err)
	}

	depSvc := deposit.NewService(pool, ledger.NewService())
	if _, err := depSvc.Deposit(ctx, victim.ID, victimAcc.ID, money.FromMinor(1_000_000, money.IDR)); err != nil {
		t.Fatalf("fund victim: %v", err)
	}
	// the attacker's own account is left empty on purpose

	xferSvc := transfer.NewService(pool, ledger.NewService())
	_, err = xferSvc.Transfer(ctx, attacker.ID, attackerAcc.ID, victimAcc.ID, money.FromMinor(-500_000, money.IDR))
	if !errors.Is(err, transfer.ErrInvalidAmount) {
		t.Fatalf("negative transfer: got %v, want ErrInvalidAmount", err)
	}

	if balance := balanceOf(t, ctx, pool, victimAcc.LedgerAccountID); balance != 1_000_000 {
		t.Fatalf("victim balance = %d, want 1000000 (untouched)", balance)
	}
	if balance := balanceOf(t, ctx, pool, attackerAcc.LedgerAccountID); balance != 0 {
		t.Fatalf("attacker balance = %d, want 0 (untouched)", balance)
	}
}
