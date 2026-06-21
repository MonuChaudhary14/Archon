package auth

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

type EmailQueue struct {
	client *redis.Client
}

func NewEmailQueue(
	client *redis.Client,
) *EmailQueue {

	return &EmailQueue{
		client: client,
	}
}

func (q *EmailQueue) Enqueue(
	ctx context.Context,
	job EmailJob,
) error {

	payload, err := json.Marshal(job)

	if err != nil {
		return err
	}

	return q.client.RPush(
		ctx,
		"email_queue",
		payload,
	).Err()
}
