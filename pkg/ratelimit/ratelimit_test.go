package ratelimit_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MonuChaudhary14/Archon/pkg/ratelimit"
	"github.com/gin-gonic/gin"
)

type mockLimiter struct {
	allowed       bool
	limit         int64
	remaining     int64
	resetDuration time.Duration
	err           error
}

func (m *mockLimiter) Allow(ctx context.Context, key string, limit int64, window time.Duration) (*ratelimit.Result, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &ratelimit.Result{
		Allowed:       m.allowed,
		Limit:         m.limit,
		Remaining:     m.remaining,
		ResetDuration: m.resetDuration,
	}, nil
}

func TestRateLimiterMiddleware_Allowed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mock := &mockLimiter{
		allowed:       true,
		limit:         10,
		remaining:     9,
		resetDuration: 30 * time.Second,
	}

	router := gin.New()
	router.Use(ratelimit.RateLimiterMiddleware(ratelimit.MiddlewareConfig{
		Limiter: mock,
		Limit:   10,
		Window:  time.Minute,
	}))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if w.Header().Get("X-RateLimit-Limit") != "10" {
		t.Errorf("expected X-RateLimit-Limit '10', got '%s'", w.Header().Get("X-RateLimit-Limit"))
	}
	if w.Header().Get("X-RateLimit-Remaining") != "9" {
		t.Errorf("expected X-RateLimit-Remaining '9', got '%s'", w.Header().Get("X-RateLimit-Remaining"))
	}
	if w.Header().Get("X-RateLimit-Reset") != "30" {
		t.Errorf("expected X-RateLimit-Reset '30', got '%s'", w.Header().Get("X-RateLimit-Reset"))
	}
}

func TestRateLimiterMiddleware_Exceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mock := &mockLimiter{
		allowed:       false,
		limit:         5,
		remaining:     0,
		resetDuration: 15 * time.Second,
	}

	router := gin.New()
	router.Use(ratelimit.RateLimiterMiddleware(ratelimit.MiddlewareConfig{
		Limiter: mock,
		Limit:   5,
		Window:  time.Minute,
	}))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "success")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", w.Code)
	}

	if w.Header().Get("Retry-After") != "15" {
		t.Errorf("expected Retry-After '15', got '%s'", w.Header().Get("Retry-After"))
	}
	if w.Header().Get("X-RateLimit-Remaining") != "0" {
		t.Errorf("expected X-RateLimit-Remaining '0', got '%s'", w.Header().Get("X-RateLimit-Remaining"))
	}
}

func TestRateLimiterMiddleware_FailOpenOnError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mock := &mockLimiter{
		err: errors.New("redis connection refused"),
	}

	router := gin.New()
	router.Use(ratelimit.RateLimiterMiddleware(ratelimit.MiddlewareConfig{
		Limiter: mock,
		Limit:   5,
		Window:  time.Minute,
	}))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "fallback allowed")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 on error fail-open, got %d", w.Code)
	}
}
