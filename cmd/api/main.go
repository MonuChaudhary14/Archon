package main

import (
	"context"
	"log"
	"os"

	"github.com/MonuChaudhary14/sys/internal/auth"
	"github.com/MonuChaudhary14/sys/internal/cache"
	"github.com/MonuChaudhary14/sys/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.NewPostgresPool(
		os.Getenv("DATABASE_URL"),
	)
	if err != nil {
		log.Fatal(err)
	}

	redisClient := cache.NewRedisClient()

	if err := cache.Ping(redisClient); err != nil {
		log.Fatal(err)
	}

	log.Println("PostgreSQL Connected")
	log.Println("Redis Connected")

	userRepository := auth.NewPostgresUserRepository(db)

	otpStore := auth.NewOTPStore(
		redisClient,
	)

	mailService := auth.NewMailService(
		os.Getenv("SMTP_HOST"),
		os.Getenv("SMTP_PORT"),
		os.Getenv("SMTP_USERNAME"),
		os.Getenv("SMTP_PASSWORD"),
	)

	emailQueue := auth.NewEmailQueue(
		redisClient,
	)

	worker := auth.NewEmailWorker(
		redisClient,
		mailService,
	)

	go worker.Start(
		context.Background(),
	)

	authService := auth.NewAuthService(
		userRepository,
		otpStore,
		mailService,
		emailQueue,
		os.Getenv("JWT_SECRET"),

	)

	authHandler := auth.NewHandler(
		authService,
	)

	router := gin.Default()

	authGroup := router.Group("/") 
	auth.RegisterRoutes(authGroup, authHandler, os.Getenv("JWT_SECRET"))

	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

