package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"

	"github.com/ndy-s/kasa/backend/internal/platform/config"
	"github.com/ndy-s/kasa/backend/internal/platform/postgres"
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

	queries := postgres.New(pool)

	r := chi.NewRouter()

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			if _, err := w.Write([]byte("db unreachable")); err != nil {
				log.Println("write failed: ", err)
			}
			return
		}

		count, err := queries.CountPings(r.Context())
		if err != nil {
			log.Println("count pings failed: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.Println("ping count:", count)

		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ok")); err != nil {
			log.Println("write failed: ", err)
		}
	})

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Println("listening on", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
