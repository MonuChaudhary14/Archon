package ratelimit

import (
	"context"
	"time"
)

type Result struct {
	Allowed       bool
	Limit         int64
	Remaining     int64
	ResetDuration time.Duration
}

type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int64, window time.Duration) (*Result, error)
}
