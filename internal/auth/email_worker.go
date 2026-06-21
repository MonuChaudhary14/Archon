package auth

import (
	"context"
	"encoding/json"
	"log"

	"github.com/redis/go-redis/v9"
)

type EmailWorker struct {
	client      *redis.Client
	mailService *MailService
}

func NewEmailWorker(
	client *redis.Client,
	mailService *MailService,
) *EmailWorker {

	return &EmailWorker{
		client:      client,
		mailService: mailService,
	}
}

func (w *EmailWorker) Start(ctx context.Context) {

	for {

		result, err := w.client.BLPop(
			ctx,
			0,
			"email_queue",
		).Result()

		if err != nil {
			log.Println(err)
			continue
		}

		var job EmailJob

		err = json.Unmarshal(
			[]byte(result[1]),
			&job,
		)

		if err != nil {
			log.Println(err)
			continue
		}

		err = w.mailService.SendVerificationEmail(
			job.Email,
			job.OTP,
		)

		if err != nil {
			log.Println(err)
			continue
		}

		log.Printf(
			"Verification email sent to %s",
			job.Email,
		)
	}
}
