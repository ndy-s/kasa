package ledger_test

import (
	"context"
	"database/sql"
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/ndy-s/kasa/backend/internal/ledger"
	"github.com/ndy-s/kasa/backend/internal/platform/postgres"
	"github.com/ndy-s/kasa/backend/internal/shared/money"
)

func TestLedgerAlwaysBalances(t *testing.T) {
	ctx := context.Background()
	pool := startPostgres(t, ctx)

	// resolve two GL account IDs to post between
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
	rng := rand.New(rand.NewSource(1))

	for i := 0; i < 200; i++ {
		amount := money.FromMinor(rng.Int63n(100000)+1, money.USD)
		lines, err := ledger.LinesFor(ledger.Deposit, ledger.PostingParams{
			Amount: amount, CashAccountID: cashID, ToAccountID: depID,
		})
		if err != nil {
			t.Fatalf("rule: %v", err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		if _, err := svc.Post(ctx, tx, ledger.PostingRequest{
			Type: ledger.Deposit, BookingDate: time.Now(), Lines: lines,
		}); err != nil {
			t.Fatalf("post: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit: %v", err)
		}
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
		t.Fatalf("ledger does not balance: debits=%d credits=%d", debits, credits)
	}
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

func uuidString(u pgtype.UUID) string { return uuid.UUID(u.Bytes).String() }
