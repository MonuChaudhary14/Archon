package config

import (
	"os"
	"strings"
)

type Config struct {
	DatabaseURL  string
	FrontendURLs []string
	Port         string
}

func Load() *Config {
	frontendURLStr := os.Getenv("FRONTEND_URL")
	if frontendURLStr == "" {
		frontendURLStr = "http://localhost:3000"
	}

	dbURL := os.Getenv("DATABASE_URL")
	port := os.Getenv("PORT")
	if port == "" {
		port = ":8080"
	}

	return &Config{
		DatabaseURL:  dbURL,
		FrontendURLs: strings.Split(frontendURLStr, ","),
		Port:         port,
	}
}
