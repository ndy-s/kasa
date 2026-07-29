package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port		string
	DatabaseURL string
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

	return cfg, nil
}
