package auth

import (
	"context"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Setup initializes all repositories, services, workers, and registers routes for the auth module
func Setup(db *pgxpool.Pool, redisClient *redis.Client, rg *gin.RouterGroup) {
	userRepository := NewPostgresUserRepository(db)
	otpStore := NewOTPStore(redisClient)
	mailService := NewMailService(
		os.Getenv("SMTP_HOST"),
		os.Getenv("SMTP_PORT"),
		os.Getenv("SMTP_USERNAME"),
		os.Getenv("SMTP_PASSWORD"),
	)
	emailQueue := NewEmailQueue(redisClient)

	worker := NewEmailWorker(redisClient, mailService)
	go worker.Start(context.Background())

	authService := NewAuthService(
		userRepository,
		otpStore,
		mailService,
		emailQueue,
		os.Getenv("JWT_SECRET"),
	)
	authHandler := NewHandler(authService)

	RegisterRoutes(rg, authHandler, os.Getenv("JWT_SECRET"))
}
