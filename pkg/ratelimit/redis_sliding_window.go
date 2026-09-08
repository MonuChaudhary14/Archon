package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var slidingWindowScript = redis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local clearBefore = now - window

redis.call('ZREMRANGEBYSCORE', key, 0, clearBefore)
local currentRequests = redis.call('ZCARD', key)

if currentRequests < limit then
    redis.call('ZADD', key, now, now)
    redis.call('PEXPIRE', key, window)
    return {1, limit, limit - currentRequests - 1, 0}
else
    local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
    local resetDuration = 0
    if oldest and #oldest >= 2 then
        local oldestTime = tonumber(oldest[2])
        resetDuration = (oldestTime + window) - now
        if resetDuration < 0 then
            resetDuration = 0
        end
    end
    return {0, limit, 0, resetDuration}
end
`)

type RedisSlidingWindowLimiter struct {
	client   redis.Cmdable
	prefix   string
	failOpen bool
}

type Option func(*RedisSlidingWindowLimiter)

func WithPrefix(prefix string) Option {
	return func(r *RedisSlidingWindowLimiter) {
		r.prefix = prefix
	}
}

func WithFailOpen(failOpen bool) Option {
	return func(r *RedisSlidingWindowLimiter) {
		r.failOpen = failOpen
	}
}

func NewRedisSlidingWindowLimiter(client redis.Cmdable, opts ...Option) *RedisSlidingWindowLimiter {
	limiter := &RedisSlidingWindowLimiter{
		client:   client,
		prefix:   "ratelimit:",
		failOpen: true,
	}

	for _, opt := range opts {
		opt(limiter)
	}

	return limiter
}

func (r *RedisSlidingWindowLimiter) Allow(ctx context.Context, key string, limit int64, window time.Duration) (*Result, error) {
	redisKey := r.prefix + key
	now := time.Now().UnixMilli()
	windowMillis := window.Milliseconds()

	res, err := slidingWindowScript.Run(ctx, r.client, []string{redisKey}, now, windowMillis, limit).Result()
	if err != nil {
		if r.failOpen {
			return &Result{
				Allowed:       true,
				Limit:         limit,
				Remaining:     1,
				ResetDuration: 0,
			}, nil
		}
		return nil, fmt.Errorf("redis rate limiter failed: %w", err)
	}

	values, ok := res.([]interface{})
	if !ok || len(values) < 4 {
		return nil, errors.New("invalid response from rate limiter script")
	}

	allowedInt, err := toInt64(values[0])
	if err != nil {
		return nil, err
	}

	limitInt, err := toInt64(values[1])
	if err != nil {
		return nil, err
	}

	remainingInt, err := toInt64(values[2])
	if err != nil {
		return nil, err
	}

	resetMillis, err := toInt64(values[3])
	if err != nil {
		return nil, err
	}

	return &Result{
		Allowed:       allowedInt == 1,
		Limit:         limitInt,
		Remaining:     remainingInt,
		ResetDuration: time.Duration(resetMillis) * time.Millisecond,
	}, nil
}

func toInt64(val interface{}) (int64, error) {
	switch v := val.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("unsupported type: %T", val)
	}
}
