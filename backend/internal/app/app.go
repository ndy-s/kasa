package app

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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

func NewRouter(pool *pgxpool.Pool, issuer *auth.TokenIssuer) http.Handler {
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
	intHandler := interest.NewAdminHandler(interest.NewService(pool, ledgerSvc), fakeClock)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(web.Logger)
	r.Use(middleware.Recoverer)
	r.Use(web.CORS)

	r.Get("/healthz", healthz(pool))
	web.MountDocs(r)
	guard := web.AuthGuard(issuer)
	custHandler.Mount(r, guard)
	accHandler.Mount(r, guard)
	depHandler.Mount(r, guard)
	xferHandler.Mount(r, guard, web.Idempotency(pool))
	histHandler.Mount(r, guard)
	stmtHandler.Mount(r, guard)
	intHandler.Mount(r, guard)
	eduHandler.Mount(r, guard)
	loanHandler.Mount(r, guard)
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
