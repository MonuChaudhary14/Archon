package auth

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

const(
	TypeEmailDelivery = "email:deliver"
)

type EmailQueue struct{
	client *asynq.Client
}

func NewEmailQueue(redisOpt asynq.RedisClientOpt) *EmailQueue{

	return &EmailQueue{
		client: asynq.NewClient(redisOpt),
	}
}

func (q *EmailQueue) Enqueue(ctx context.Context, job EmailJob) error{
	payload, err := json.Marshal(job)

	if err != nil{
		return fmt.Errorf("Failed to marshal email job: %w", err)
	}

	task := asynq.NewTask(TypeEmailDelivery, payload)
	
	info, err := q.client.EnqueueContext(ctx, task, asynq.MaxRetry(3))

	if err != nil{
		return fmt.Errorf("Could not enqueue email task: %w", err)
	}

	fmt.Printf("Enqueued task: id = %s queue = %s\n", info.ID, info.Queue)
	return nil
}
