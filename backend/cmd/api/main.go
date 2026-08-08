package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/ndy-s/kasa/backend/internal/customer"
	"github.com/ndy-s/kasa/backend/internal/platform/auth"
	"github.com/ndy-s/kasa/backend/internal/platform/config"
	"github.com/ndy-s/kasa/backend/internal/platform/postgres"
	"github.com/ndy-s/kasa/backend/internal/platform/web"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, relying on environment variables")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	pool, err := postgres.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	issuer := auth.NewTokenIssuer(cfg.JWTSecret, time.Hour)
	repo := customer.NewPgRepository(pool)
	svc := customer.NewService(repo, issuer)
	handler := customer.NewHandler(svc)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(web.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", healthz(pool))
	handler.Mount(r, web.AuthGuard(issuer))

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Println("listening on", addr)
	log.Fatal(http.ListenAndServe(addr, r))
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
