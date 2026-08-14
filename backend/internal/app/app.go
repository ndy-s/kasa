package app

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ndy-s/kasa/backend/internal/account"
	"github.com/ndy-s/kasa/backend/internal/customer"
	"github.com/ndy-s/kasa/backend/internal/deposit"
	"github.com/ndy-s/kasa/backend/internal/education"
	"github.com/ndy-s/kasa/backend/internal/history"
	"github.com/ndy-s/kasa/backend/internal/interest"
	"github.com/ndy-s/kasa/backend/internal/ledger"
	"github.com/ndy-s/kasa/backend/internal/loan"
	"github.com/ndy-s/kasa/backend/internal/platform/auth"
	"github.com/ndy-s/kasa/backend/internal/platform/clock"
	"github.com/ndy-s/kasa/backend/internal/platform/postgres"
	"github.com/ndy-s/kasa/backend/internal/platform/web"
	"github.com/ndy-s/kasa/backend/internal/statement"
	"github.com/ndy-s/kasa/backend/internal/transfer"
)

// Config holds the per-environment settings NewRouter needs beyond the pool and token issuer.
type Config struct {
	AllowedOrigin string // the web app's origin, for CORS
	AdminToken    string // required header value for the learn-mode admin routes
}

func NewRouter(pool *pgxpool.Pool, issuer *auth.TokenIssuer, cfg Config) http.Handler {
	ledgerSvc := ledger.NewService()
	q := postgres.New(pool)
	custHandler := customer.NewHandler(customer.NewService(customer.NewPgRepository(pool), issuer))
	accSvc := account.NewService(pool)
	depSvc := deposit.NewService(pool, ledgerSvc)
	accHandler := account.NewHandler(accSvc)
	depHandler := deposit.NewHandler(depSvc)
	xferHandler := transfer.NewHandler(transfer.NewService(pool, ledgerSvc))
	histHandler := history.NewHandler(pool)
	stmtHandler := statement.NewHandler(statement.NewService(q), q)
	eduHandler := education.NewHandler(pool, accSvc, depSvc, ledgerSvc)
	loanHandler := loan.NewHandler(loan.NewService(pool, ledgerSvc))

	fakeClock := clock.NewFake(time.Now())
	intHandler := interest.NewAdminHandler(interest.NewService(pool, ledgerSvc), fakeClock, cfg.AdminToken)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	// deliberately no RealIP: chi's middleware.RealIP trusts client-supplied X-Forwarded-For/X-Real-IP
	// unconditionally, which lets a client spoof its rate-limit identity (GHSA-3fxj-6jh8-hvhx); the rate
	// limiter below keys on the raw TCP peer address instead.
	r.Use(web.Logger)
	r.Use(middleware.Recoverer)
	r.Use(web.SecurityHeaders)
	r.Use(web.Metrics)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.AllowedOrigin},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "Idempotency-Key", "X-Admin-Token"},
		AllowCredentials: false,
	}))

	authLimit := web.RateLimit(1, 5)   // register/login: tight, brute-force resistant
	moneyLimit := web.RateLimit(5, 10) // deposit/withdraw/transfer/loans

	r.Get("/healthz", healthz(pool))
	r.Handle("/metrics", web.MetricsHandler())
	web.MountDocs(r)
	guard := web.AuthGuard(issuer)
	custHandler.Mount(r, guard, authLimit)
	accHandler.Mount(r, guard)
	depHandler.Mount(r, guard, moneyLimit)
	xferHandler.Mount(r, guard, web.Idempotency(pool), moneyLimit)
	histHandler.Mount(r, guard)
	stmtHandler.Mount(r, guard)
	intHandler.Mount(r, guard)
	eduHandler.Mount(r, guard)
	loanHandler.Mount(r, guard, moneyLimit)
	return r
}

func healthz(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("db unreachable"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}
