package main

import (
	"log"

	_ "github.com/MonuChaudhary14/Archon/docs"
	"github.com/MonuChaudhary14/Archon/internal/server"
	"github.com/MonuChaudhary14/Archon/pkg/config"
	"github.com/joho/godotenv"
)

// @title           Sys API
// @version         1.0
// @description     This is the Sys API server.

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := config.Load()

	srv, err := server.NewServer(cfg)
	if err != nil {
		log.Fatal(err)
	}

	if err := srv.Run(cfg.Port); err != nil {
		log.Fatal(err)
	}
}
