package interview

import (
	"context"
	"log"
	"time"

	"github.com/MonuChaudhary14/Archon/pkg/resilience"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MessagePublisher interface {
	PublishEvent(ctx context.Context, topic string, key []byte, payload []byte) error
}

type OutboxWorker struct {
	db        *pgxpool.Pool
	publisher MessagePublisher
	interval  time.Duration
	breaker   *resilience.CircuitBreaker[struct{}]
}

func NewOutboxWorker(db *pgxpool.Pool, publisher MessagePublisher, interval time.Duration) *OutboxWorker {
	breaker := resilience.NewCircuitBreaker[struct{}](resilience.Config{
		Name:         "kafka-outbox-publisher",
		Threshold:    5,
		FailureRatio: 0.5,
		Timeout:      10 * time.Second,
	})

	return &OutboxWorker{
		db:        db,
		publisher: publisher,
		interval:  interval,
		breaker:   breaker,
	}
}

func (w *OutboxWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping outbox worker...")
			return
		case <-ticker.C:
			w.processPendingEvents(ctx)
		}
	}
}

func getTopicForEvent(eventType string) string {
	switch eventType {
	case "INTERVIEW_STARTED", "INTERVIEW_SUBMITTED":
		return "ai.requests"
	default:
		return "ai.requests"
	}
}

func (w *OutboxWorker) processPendingEvents(ctx context.Context) {
	tx, err := w.db.Begin(ctx)
	if err != nil {
		log.Printf("Outbox worker failed to start transaction: %v", err)
		return
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	query := `
		SELECT id, aggregate_id, type, payload, retries 
		FROM outbox_events 
		WHERE status = 'PENDING' 
		ORDER BY created_at ASC 
		LIMIT 10
		FOR UPDATE SKIP LOCKED;
	`
	rows, err := tx.Query(ctx, query)
	if err != nil {
		log.Printf("Outbox worker failed to query pending events: %v", err)
		return
	}
	defer rows.Close()

	type outboxEvent struct {
		id          string
		aggregateID string
		eventType   string
		payload     []byte
		retries     int
	}

	var events []outboxEvent
	for rows.Next() {
		var ev outboxEvent
		err := rows.Scan(&ev.id, &ev.aggregateID, &ev.eventType, &ev.payload, &ev.retries)
		if err != nil {
			log.Printf("Outbox worker failed to scan event: %v", err)
			continue
		}
		events = append(events, ev)
	}
	rows.Close()

	for _, ev := range events {
		pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		topic := getTopicForEvent(ev.eventType)
		_, err = w.breaker.Execute(func() (struct{}, error) {
			pubErr := w.publisher.PublishEvent(pubCtx, topic, []byte(ev.aggregateID), ev.payload)
			return struct{}{}, pubErr
		})
		cancel()

		if err != nil {
			if resilience.IsCircuitOpenError(err) {
				log.Printf("Outbox worker paused event %s: circuit breaker is open", ev.id)
				break
			}

			newRetries := ev.retries + 1
			var updateQuery string
			var args []interface{}

			log.Printf("Failed to publish outbox event %s (retry %d/5): %v", ev.id, newRetries, err)

			if newRetries >= 5 {
				updateQuery = `
					UPDATE outbox_events 
					SET status = 'FAILED', retries = $1, error_reason = $2 
					WHERE id = $3
				`
				args = []interface{}{newRetries, err.Error(), ev.id}
			} else {
				updateQuery = `
					UPDATE outbox_events 
					SET retries = $1, error_reason = $2 
					WHERE id = $3
				`
				args = []interface{}{newRetries, err.Error(), ev.id}
			}

			_, execErr := tx.Exec(ctx, updateQuery, args...)
			if execErr != nil {
				log.Printf("Failed to update outbox event %s on failure: %v", ev.id, execErr)
			}
		} else {
			updateQuery := `
				UPDATE outbox_events 
				SET status = 'PROCESSED', processed_at = NOW() 
				WHERE id = $1
			`
			_, execErr := tx.Exec(ctx, updateQuery, ev.id)
			if execErr != nil {
				log.Printf("Failed to mark outbox event %s as processed: %v", ev.id, execErr)
			}
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		log.Printf("Outbox worker failed to commit transaction: %v", err)
	}
}
