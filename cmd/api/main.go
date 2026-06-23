package main

import (
	"log"

	_ "github.com/MonuChaudhary14/sys/docs"
	"github.com/MonuChaudhary14/sys/internal/server"
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

	srv, err := server.NewServer()
	if err != nil {
		log.Fatal(err)
	}

	if err := srv.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
