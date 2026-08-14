package deposit_test

import (
	"context"
	"database/sql"
	"errors"
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
	"github.com/ndy-s/kasa/backend/internal/ledger"
	"github.com/ndy-s/kasa/backend/internal/platform/auth"
	"github.com/ndy-s/kasa/backend/internal/shared/money"
)

func TestWithdrawRejectsOverdraft(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(t, ctx)

	custSvc := customer.NewService(customer.NewPgRepository(pool), auth.NewTokenIssuer("test-secret", time.Hour))
	cust, err := custSvc.Register(ctx, "Overdraft Test", "overdraft@example.com", "s3cret-password")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	accSvc := account.NewService(pool)
	acc, err := accSvc.OpenAccount(ctx, cust.ID, "SAV")
	if err != nil {
		t.Fatalf("open account: %v", err)
	}

	depSvc := deposit.NewService(pool, ledger.NewService())
	if _, err := depSvc.Deposit(ctx, cust.ID, acc.ID, money.FromMinor(10000, money.IDR)); err != nil {
		t.Fatalf("deposit: %v", err)
	}

	if _, err := depSvc.Withdraw(ctx, cust.ID, acc.ID, money.FromMinor(15000, money.IDR)); !errors.Is(err, deposit.ErrInsufficientFunds) {
		t.Fatalf("withdraw over balance: got %v, want ErrInsufficientFunds", err)
	}

	balance := balanceOf(t, ctx, pool, acc.LedgerAccountID)
	if balance != 10000 {
		t.Fatalf("balance after rejected withdrawal = %d, want 10000 (nothing posted)", balance)
	}

	if _, err := depSvc.Withdraw(ctx, cust.ID, acc.ID, money.FromMinor(4000, money.IDR)); err != nil {
		t.Fatalf("valid withdraw: %v", err)
	}
	if balance := balanceOf(t, ctx, pool, acc.LedgerAccountID); balance != 6000 {
		t.Fatalf("balance after valid withdrawal = %d, want 6000", balance)
	}
}

// A negative "deposit" would debit the account by the sign-flipped amount, acting as an unchecked
// withdrawal that bypasses the overdraft check ErrInvalidAmount closes off.
func TestDepositRejectsNegativeAmount(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(t, ctx)

	custSvc := customer.NewService(customer.NewPgRepository(pool), auth.NewTokenIssuer("test-secret", time.Hour))
	cust, err := custSvc.Register(ctx, "Negative Amount Test", "negative@example.com", "s3cret-password")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	accSvc := account.NewService(pool)
	acc, err := accSvc.OpenAccount(ctx, cust.ID, "SAV")
	if err != nil {
		t.Fatalf("open account: %v", err)
	}

	depSvc := deposit.NewService(pool, ledger.NewService())
	if _, err := depSvc.Deposit(ctx, cust.ID, acc.ID, money.FromMinor(10000, money.IDR)); err != nil {
		t.Fatalf("deposit: %v", err)
	}

	if _, err := depSvc.Deposit(ctx, cust.ID, acc.ID, money.FromMinor(-5000, money.IDR)); !errors.Is(err, deposit.ErrInvalidAmount) {
		t.Fatalf("negative deposit: got %v, want ErrInvalidAmount", err)
	}
	if _, err := depSvc.Withdraw(ctx, cust.ID, acc.ID, money.FromMinor(-5000, money.IDR)); !errors.Is(err, deposit.ErrInvalidAmount) {
		t.Fatalf("negative withdraw: got %v, want ErrInvalidAmount", err)
	}
	if _, err := depSvc.Deposit(ctx, cust.ID, acc.ID, money.FromMinor(0, money.IDR)); !errors.Is(err, deposit.ErrInvalidAmount) {
		t.Fatalf("zero deposit: got %v, want ErrInvalidAmount", err)
	}

	if balance := balanceOf(t, ctx, pool, acc.LedgerAccountID); balance != 10000 {
		t.Fatalf("balance after rejected non-positive amounts = %d, want 10000 (nothing posted)", balance)
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
