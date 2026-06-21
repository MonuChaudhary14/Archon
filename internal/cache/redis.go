package cache

import (
	"context"
	"github.com/redis/go-redis/v9"
)

func NewRedisClient() *redis.Client {
	return redis.NewClient(
		&redis.Options{
			Addr: "localhost:6379",
			DB:   0,
		},
	)
}

func Ping(client *redis.Client) error {
	return client.Ping(
		context.Background(),
	).Err()
}
