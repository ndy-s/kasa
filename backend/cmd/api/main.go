package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/joho/godotenv"

	"github.com/ndy-s/kasa/backend/internal/app"
	"github.com/ndy-s/kasa/backend/internal/platform/auth"
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

	issuer := auth.NewTokenIssuer(cfg.JWTSecret, time.Hour)
	r := app.NewRouter(pool, issuer)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Println("listening on", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
