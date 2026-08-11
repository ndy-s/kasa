package e2e

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/ndy-s/kasa/backend/internal/app"
	"github.com/ndy-s/kasa/backend/internal/platform/auth"
)

func TestRegisterLoginMe(t *testing.T) {
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
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("dialect: %v", err)
	}
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

	resp := do(t, http.MethodPost, srv.URL+"/register", "",
		`{"name":"Alice","email":"alice@example.com","password":"s3cret-password"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d", resp.StatusCode)
	}

	resp = do(t, http.MethodPost, srv.URL+"/login", "",
		`{"email":"alice@example.com","password":"s3cret-password"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d", resp.StatusCode)
	}
	var login struct {
		Token string `json:"token"`
	}
	decode(t, resp, &login)
	if login.Token == "" {
		t.Fatal("login returned empty token")
	}

	resp = do(t, http.MethodGet, srv.URL+"/me", login.Token, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("me status = %d", resp.StatusCode)
	}
	var me struct {
		Email string `json:"email"`
	}
	decode(t, resp, &me)
	if me.Email != "alice@example.com" {
		t.Fatalf("me email = %q", me.Email)
	}
}

func do(t *testing.T, method, url, token, body string) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = bytes.NewBufferString(body)
	}
	req, _ := http.NewRequest(method, url, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

func decode(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	data, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("decode: %v (body: %s)", err, data)
	}
}
