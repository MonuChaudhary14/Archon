package ai

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type redisHistoryRepository struct {
	redisClient *redis.Client
}

func NewRedisHistoryRepository(redisClient *redis.Client) ChatHistoryRepository {
	return &redisHistoryRepository{
		redisClient: redisClient,
	}
}

func (r *redisHistoryRepository) GetChatHistory(ctx context.Context, sessionID string) ([]string, error) {
	return r.redisClient.LRange(ctx, "message_store:"+sessionID, 0, -1).Result()
}
