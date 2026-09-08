package auth

import (
	"time"

	"github.com/MonuChaudhary14/Archon/pkg/ratelimit"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func RegisterRoutes(rg *gin.RouterGroup, handler *Handler, jwtSecret string, repo UserRepository, redisClient redis.Cmdable) {
	limiter := ratelimit.NewRedisSlidingWindowLimiter(redisClient)
	rg.Use(ratelimit.RateLimiterMiddleware(ratelimit.MiddlewareConfig{
		Limiter: limiter,
		Limit:   10,
		Window:  time.Minute,
	}))

	rg.POST("/register", handler.Register)
	rg.POST("/verify-email", handler.VerifyEmail)
	rg.POST("/login", handler.Login)
	rg.POST("/refresh", handler.Refresh)
	rg.POST("/forgot-password", handler.ForgotPassword)
	rg.POST("/verify-reset-otp", handler.VerifyResetOTP)
	rg.POST("/reset-password", handler.ResetPassword)
	rg.POST("/resend-otp", handler.ResendOTP)

	rg.GET("/login/:provider", handler.OAuthLoginInitiate)
	rg.GET("/callback/:provider", handler.OAuthCallback)

	protected := rg.Group("/")
	protected.Use(AuthMiddleware(jwtSecret, repo))
	protected.POST("/logout", handler.Logout)
}
