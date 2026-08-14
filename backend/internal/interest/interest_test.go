package interest_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/ndy-s/kasa/backend/internal/account"
	"github.com/ndy-s/kasa/backend/internal/customer"
	"github.com/ndy-s/kasa/backend/internal/deposit"
	"github.com/ndy-s/kasa/backend/internal/interest"
	"github.com/ndy-s/kasa/backend/internal/ledger"
	"github.com/ndy-s/kasa/backend/internal/platform/auth"
	"github.com/ndy-s/kasa/backend/internal/shared/money"
)

func TestAccrueThenCapitalize(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(t, ctx)

	custSvc := customer.NewService(customer.NewPgRepository(pool), auth.NewTokenIssuer("test-secret", time.Hour))
	cust, err := custSvc.Register(ctx, "Interest Test", "interest@example.com", "s3cret-password")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	accSvc := account.NewService(pool)
	acc, err := accSvc.OpenAccount(ctx, cust.ID, "SAV") // 150 bps, seeded on Day 9
	if err != nil {
		t.Fatalf("open account: %v", err)
	}

	ledgerSvc := ledger.NewService()
	depSvc := deposit.NewService(pool, ledgerSvc)
	if _, err := depSvc.Deposit(ctx, cust.ID, acc.ID, money.FromMinor(100000, money.IDR)); err != nil {
		t.Fatalf("deposit: %v", err)
	}

	intSvc := interest.NewService(pool, ledgerSvc)
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	const days = 30
	for i := 0; i < days; i++ {
		if err := intSvc.Accrue(ctx, base.AddDate(0, 0, i)); err != nil {
			t.Fatalf("accrue day %d: %v", i, err)
		}
	}

	if err := intSvc.CapitalizeAll(ctx, base.AddDate(0, 0, days)); err != nil {
		t.Fatalf("capitalize: %v", err)
	}

	// 1000.00 at 1.50% is 4 minor units/day (see the unit test); 30 days accrued before capitalizing.
	const wantBalance = 100000 + 30*4
	balance := balanceOf(t, ctx, pool, acc.LedgerAccountID)
	if balance != wantBalance {
		t.Fatalf("balance after capitalization = %d, want %d", balance, wantBalance)
	}

	var debits, credits int64
	row := pool.QueryRow(ctx, `
		SELECT
		  COALESCE(SUM(amount_minor) FILTER (WHERE direction = 'debit'), 0),
		  COALESCE(SUM(amount_minor) FILTER (WHERE direction = 'credit'), 0)
		FROM journal_line`)
	if err := row.Scan(&debits, &credits); err != nil {
		t.Fatalf("scan totals: %v", err)
	}
	if debits != credits {
		t.Fatalf("ledger does not balance after capitalization: debits=%d credits=%d", debits, credits)
	}
}

func balanceOf(t *testing.T, ctx context.Context, pool *pgxpool.Pool, ledgerAccountID string) int64 {
	t.Helper()
	var balance int64
	row := pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(CASE WHEN direction = 'credit' THEN amount_minor ELSE -amount_minor END), 0)
		FROM journal_line WHERE ledger_account_id = $1`, ledgerAccountID)
	if err := row.Scan(&balance); err != nil {
		t.Fatalf("balance query: %v", err)
	}
	return balance
}

func startPostgres(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()
	container, err := tcpostgres.Run(ctx, "postgres:16",
		tcpostgres.WithDatabase("kasa"),
		tcpostgres.WithUsername("kasa"),
		tcpostgres.WithPassword("kasa"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("conn string: %v", err)
	}

	sqlDB, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("open sql: %v", err)
	}
	_ = goose.SetDialect("postgres")
	if err := goose.Up(sqlDB, "../../db/migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	_ = sqlDB.Close()

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
