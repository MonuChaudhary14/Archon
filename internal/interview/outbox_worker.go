package interview

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MessagePublisher interface {
	PublishEvent(ctx context.Context, key []byte, payload []byte) error
}

type OutboxWorker struct {
	db        *pgxpool.Pool
	publisher MessagePublisher
	interval  time.Duration
}

func NewOutboxWorker(db *pgxpool.Pool, publisher MessagePublisher, interval time.Duration) *OutboxWorker {
	return &OutboxWorker{
		db:        db,
		publisher: publisher,
		interval:  interval,
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

func (w *OutboxWorker) processPendingEvents(ctx context.Context) {
	tx, err := w.db.Begin(ctx)
	if err != nil {
		log.Printf("Outbox worker failed to start transaction: %v", err)
		return
	}
	defer tx.Rollback(ctx)

	query := `
		SELECT id, aggregate_id, payload, retries 
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
		payload     []byte
		retries     int
	}

	var events []outboxEvent
	for rows.Next() {
		var ev outboxEvent
		if err := rows.Scan(&ev.id, &ev.aggregateID, &ev.payload, &ev.retries); err != nil {
			log.Printf("Outbox worker failed to scan event: %v", err)
			continue
		}
		events = append(events, ev)
	}
	rows.Close()

	for _, ev := range events {
		pubCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		err = w.publisher.PublishEvent(pubCtx, []byte(ev.aggregateID), ev.payload)
		cancel()

		if err != nil {
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

			if _, execErr := tx.Exec(ctx, updateQuery, args...); execErr != nil {
				log.Printf("Failed to update outbox event %s on failure: %v", ev.id, execErr)
			}
		} else {
			updateQuery := `
				UPDATE outbox_events 
				SET status = 'PROCESSED', processed_at = NOW() 
				WHERE id = $1
			`
			if _, execErr := tx.Exec(ctx, updateQuery, ev.id); execErr != nil {
				log.Printf("Failed to mark outbox event %s as processed: %v", ev.id, execErr)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("Outbox worker failed to commit transaction: %v", err)
	}
}
