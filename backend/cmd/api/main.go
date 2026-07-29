package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"

	"github.com/ndy-s/kasa/backend/internal/platform/config"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, relying on environment variables")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	r := chi.NewRouter()

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		if _, err := w.Write([]byte("ok")); err != nil {
			log.Println("write failed: ", err)
		}
	})

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Println("listening on", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
