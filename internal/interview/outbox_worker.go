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
	query := `
		SELECT id, aggregate_id, payload 
		FROM outbox_events 
		WHERE status = 'PENDING' 
		ORDER BY created_at ASC 
		LIMIT 10;
	`
	rows, err := w.db.Query(ctx, query)
	if err != nil {
		log.Printf("Outbox worker failed to fetch events: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, aggregateID string
		var payload []byte

		if err := rows.Scan(&id, &aggregateID, &payload); err != nil {
			log.Printf("Outbox worker failed to scan event: %v", err)
			continue
		}

		err = w.publisher.PublishEvent(ctx, []byte(aggregateID), payload)
		if err != nil {
			log.Printf("Failed to publish event %s: %v", id, err)
			continue
		}

		updateQuery := `UPDATE outbox_events SET status = 'PROCESSED', processed_at = NOW() WHERE id = $1`
		if _, err := w.db.Exec(ctx, updateQuery, id); err != nil {
			log.Printf("Failed to mark event %s as processed: %v", id, err)
		}
	}
}
