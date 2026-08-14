package main

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"

	"github.com/ndy-s/kasa/backend/internal/account"
	"github.com/ndy-s/kasa/backend/internal/customer"
	"github.com/ndy-s/kasa/backend/internal/ledger"
	"github.com/ndy-s/kasa/backend/internal/platform/auth"
	"github.com/ndy-s/kasa/backend/internal/platform/config"
	"github.com/ndy-s/kasa/backend/internal/platform/postgres"
	"github.com/ndy-s/kasa/backend/internal/shared/money"
)

func main() {
	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// demo customer
	issuer := auth.NewTokenIssuer(cfg.JWTSecret, time.Hour)
	custSvc := customer.NewService(customer.NewPgRepository(pool), issuer)
	demo, err := custSvc.Register(ctx, "Demo User", "demo@example.com", "demo-password")
	if err != nil {
		log.Fatal("register demo: ", err)
	}

	// open an account
	accSvc := account.NewService(pool)
	acc, err := accSvc.OpenAccount(ctx, demo.ID, "SAV")
	if err != nil {
		log.Fatal("open account: ", err)
	}

	// opening deposit of 500.00 posted directly via the ledger
	q := postgres.New(pool)
	cash, err := q.GetAccountByCode(ctx, "1000")
	if err != nil {
		log.Fatal("cash account: ", err)
	}
	lines, err := ledger.LinesFor(ledger.Deposit, ledger.PostingParams{
		Amount:        money.FromMinor(50000, money.IDR),
		CashAccountID: uuid.UUID(cash.ID.Bytes).String(),
		ToAccountID:   acc.LedgerAccountID,
	})
	if err != nil {
		log.Fatal("rule: ", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful commit
	if _, err := ledger.NewService().Post(ctx, tx, ledger.PostingRequest{
		Type: ledger.Deposit, Description: "opening deposit", BookingDate: time.Now(), Lines: lines,
	}); err != nil {
		log.Fatal("post: ", err)
	}
	if err := tx.Commit(ctx); err != nil {
		log.Fatal(err)
	}

	log.Printf("seeded demo customer %s with account %s", demo.ID, acc.ID)
}
