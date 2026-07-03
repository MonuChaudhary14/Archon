package auth

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/hibiken/asynq"
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
	asynqRedisOpt := asynq.RedisClientOpt{
		Addr: os.Getenv("REDIS_ADDR"),
	}
	emailQueue := NewEmailQueue(asynqRedisOpt)

	worker := NewEmailWorker(asynqRedisOpt, mailService)
	go worker.Start()

	authService := NewAuthService(
		userRepository,
		otpStore,
		mailService,
		emailQueue,
		os.Getenv("JWT_SECRET"),
	)
	oauthProviders := map[string]OAuthProvider{
		"google": NewGoogleOAuth(
			os.Getenv("GOOGLE_CLIENT_ID"),
			os.Getenv("GOOGLE_CLIENT_SECRET"),
			os.Getenv("GOOGLE_REDIRECT_URL"),
		),
		"github": NewGithubOAuth(
			os.Getenv("GITHUB_CLIENT_ID"),
			os.Getenv("GITHUB_CLIENT_SECRET"),
			os.Getenv("GITHUB_REDIRECT_URL"),
		),
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	authHandler := NewHandler(authService, oauthProviders, frontendURL)

	RegisterRoutes(rg, authHandler, os.Getenv("JWT_SECRET"), userRepository)
}
