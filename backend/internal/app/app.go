package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ndy-s/kasa/backend/internal/account"
	"github.com/ndy-s/kasa/backend/internal/customer"
	"github.com/ndy-s/kasa/backend/internal/deposit"
	"github.com/ndy-s/kasa/backend/internal/ledger"
	"github.com/ndy-s/kasa/backend/internal/platform/auth"
	"github.com/ndy-s/kasa/backend/internal/platform/web"
)

func NewRouter(pool *pgxpool.Pool, issuer *auth.TokenIssuer) http.Handler {
	ledgerSvc := ledger.NewService()
	custHandler := customer.NewHandler(customer.NewService(customer.NewPgRepository(pool), issuer))
	accHandler := account.NewHandler(account.NewService(pool))
	depHandler := deposit.NewHandler(deposit.NewService(pool, ledgerSvc))

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(web.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", healthz(pool))
	web.MountDocs(r)
	guard := web.AuthGuard(issuer)
	custHandler.Mount(r, guard)
	accHandler.Mount(r, guard)
	depHandler.Mount(r, guard)
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
