package e2e

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
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

	"github.com/ndy-s/kasa/backend/internal/app"
	"github.com/ndy-s/kasa/backend/internal/ledger"
	"github.com/ndy-s/kasa/backend/internal/platform/auth"
	"github.com/ndy-s/kasa/backend/internal/platform/postgres"
	"github.com/ndy-s/kasa/backend/internal/shared/money"
)

func TestAccountBalanceIsDerivedFromTheLedger(t *testing.T) {
	ctx := context.Background()

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
		t.Fatalf("connection string: %v", err)
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

	srv := httptest.NewServer(app.NewRouter(pool, auth.NewTokenIssuer("test-secret", time.Hour)))
	t.Cleanup(srv.Close)

	// register and log in
	do(t, http.MethodPost, srv.URL+"/register", "",
		`{"name":"Demo","email":"demo@example.com","password":"s3cret-password"}`)

	resp := do(t, http.MethodPost, srv.URL+"/login", "",
		`{"email":"demo@example.com","password":"s3cret-password"}`)
	var login struct {
		Token string `json:"token"`
	}
	decode(t, resp, &login)

	// open an account
	resp = do(t, http.MethodPost, srv.URL+"/accounts", login.Token, `{"product_code":"SAV"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("open account status = %d", resp.StatusCode)
	}
	var acc struct {
		ID string `json:"id"`
	}
	decode(t, resp, &acc)

	// resolve the account's GL sub-account and the cash account, then post a deposit directly
	q := postgres.New(pool)
	accID, err := uuid.Parse(acc.ID)
	if err != nil {
		t.Fatalf("parse account id: %v", err)
	}
	row, err := q.GetAccountByID(ctx, pgtype.UUID{Bytes: accID, Valid: true})
	if err != nil {
		t.Fatalf("get account: %v", err)
	}
	cash, err := q.GetAccountByCode(ctx, "1000")
	if err != nil {
		t.Fatalf("cash account: %v", err)
	}

	lines, err := ledger.LinesFor(ledger.Deposit, ledger.PostingParams{
		Amount:        money.FromMinor(50000, money.IDR),
		CashAccountID: uuid.UUID(cash.ID.Bytes).String(),
		ToAccountID:   uuid.UUID(row.LedgerAccountID.Bytes).String(),
	})
	if err != nil {
		t.Fatalf("rule: %v", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if _, err := ledger.NewService().Post(ctx, tx, ledger.PostingRequest{
		Type: ledger.Deposit, Description: "opening deposit", BookingDate: time.Now(), Lines: lines,
	}); err != nil {
		t.Fatalf("post: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// the balance endpoint must reflect the posting
	resp = do(t, http.MethodGet, srv.URL+"/accounts/"+acc.ID+"/balances", login.Token, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("balances status = %d", resp.StatusCode)
	}
	var balances struct {
		Ledger string `json:"ledger"`
	}
	decode(t, resp, &balances)
	if balances.Ledger != "500.00 IDR" {
		t.Fatalf("ledger balance = %q, want %q", balances.Ledger, "500.00 IDR")
	}
}
