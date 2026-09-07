package main

import (
	"log"

	_ "github.com/MonuChaudhary14/Archon/docs"
	"github.com/MonuChaudhary14/Archon/internal/database"
	"github.com/MonuChaudhary14/Archon/internal/server"
	"github.com/MonuChaudhary14/Archon/migrations"
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
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := config.Load()

	err = database.RunMigrations(cfg.DatabaseURL, migrations.FS)
	if err != nil {
		log.Fatalf("Database migration failed: %v", err)
	}

	srv, err := server.NewServer(cfg)
	if err != nil {
		log.Fatal(err)
	}

	err = srv.Run(cfg.Port)
	if err != nil {
		log.Fatal(err)
	}
}
