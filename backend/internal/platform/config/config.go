package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port          string
	DatabaseURL   string
	JWTSecret     string
	AllowedOrigin string
	AdminToken    string
}

func Load() (Config, error) {
	cfg := Config{}

	port, exists := os.LookupEnv("PORT")
	if !exists || port == "" {
		port = "8080"
	}
	cfg.Port = port

	dbURL, exists := os.LookupEnv("DATABASE_URL")
	if !exists || dbURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	cfg.DatabaseURL = dbURL

	secret, exists := os.LookupEnv("JWT_SECRET")
	if !exists || secret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}
	cfg.JWTSecret = secret

	origin, exists := os.LookupEnv("ALLOWED_ORIGIN")
	if !exists || origin == "" {
		origin = "http://localhost:5173" // the web app's Vite dev server
	}
	cfg.AllowedOrigin = origin

	adminToken, exists := os.LookupEnv("ADMIN_TOKEN")
	if !exists || adminToken == "" {
		return Config{}, fmt.Errorf("ADMIN_TOKEN is required")
	}
	cfg.AdminToken = adminToken

	return cfg, nil
}
