package ratelimit

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type KeyFunc func(c *gin.Context) string

func IPKeyFunc(c *gin.Context) string {
	return c.ClientIP()
}

func UserOrIPKeyFunc(c *gin.Context) string {
	userID, exists := c.Get("user_id")
	if exists {
		return fmt.Sprintf("user:%v", userID)
	}
	return fmt.Sprintf("ip:%s", c.ClientIP())
}

type MiddlewareConfig struct {
	Limiter    RateLimiter
	Limit      int64
	Window     time.Duration
	KeyFunc    KeyFunc
	OnExceeded gin.HandlerFunc
}

func RateLimiterMiddleware(cfg MiddlewareConfig) gin.HandlerFunc {
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = IPKeyFunc
	}
	if cfg.OnExceeded == nil {
		cfg.OnExceeded = func(c *gin.Context) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests, please slow down",
			})
			c.Abort()
		}
	}

	return func(c *gin.Context) {
		key := cfg.KeyFunc(c)
		result, err := cfg.Limiter.Allow(c.Request.Context(), key, cfg.Limit, cfg.Window)
		if err != nil {
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.FormatInt(result.Limit, 10))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(result.Remaining, 10))

		resetSeconds := int(math.Ceil(result.ResetDuration.Seconds()))
		if resetSeconds < 0 {
			resetSeconds = 0
		}
		c.Header("X-RateLimit-Reset", strconv.Itoa(resetSeconds))

		if !result.Allowed {
			if resetSeconds > 0 {
				c.Header("Retry-After", strconv.Itoa(resetSeconds))
			} else {
				c.Header("Retry-After", "1")
			}
			cfg.OnExceeded(c)
			return
		}

		c.Next()
	}
}
