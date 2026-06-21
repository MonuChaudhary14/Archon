package auth

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, handler *Handler, jwtSecret string) {
	
	limiter := NewIPRateLimiter(5, 10)
	rg.Use(RateLimiterMiddleware(limiter))

	rg.POST("/register", handler.Register)
	rg.POST("/verify-email", handler.VerifyEmail)
	rg.POST("/login", handler.Login)
	rg.POST("/refresh", handler.Refresh)
	rg.POST("/forgot-password", handler.ForgotPassword)
	rg.POST("/reset-password", handler.ResetPassword)

	protected := rg.Group("/")
	protected.Use(AuthMiddleware(jwtSecret))
	protected.POST("/logout", handler.Logout)
}
