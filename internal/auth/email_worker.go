package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hibiken/asynq"
)

type EmailWorker struct {
	server      *asynq.Server
	mailService *MailService
}

func NewEmailWorker(redisOpt asynq.RedisClientOpt, mailService *MailService) *EmailWorker {
	server := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: 10,
		},
	)

	return &EmailWorker{
		server:      server,
		mailService: mailService,
	}
}

func (w *EmailWorker) Start() {
	mux := asynq.NewServeMux()

	mux.HandleFunc(TypeEmailDelivery, w.HandleEmailDeliveryTask)

	if err := w.server.Start(mux); err != nil {
		log.Fatalf("could not start asynq worker server: %v", err)
	}
}

func (w *EmailWorker) HandleEmailDeliveryTask(ctx context.Context, t *asynq.Task) error {
	var job EmailJob
	if err := json.Unmarshal(t.Payload(), &job); err != nil {
		return fmt.Errorf("json unmarshal failed: %w", err)
	}

	err := w.mailService.SendVerificationEmail(job.Email, job.OTP)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Verification email sent successfully to %s", job.Email)
	return nil
}
