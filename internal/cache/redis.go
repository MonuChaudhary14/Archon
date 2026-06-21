package cache

import (
	"context"
	"os"
	"github.com/redis/go-redis/v9"
)

func NewRedisClient() *redis.Client {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	return redis.NewClient(
		&redis.Options{
			Addr: addr,
			DB:   0,
		},
	)
}

func Ping(client *redis.Client) error {
	return client.Ping(
		context.Background(),
	).Err()
}
