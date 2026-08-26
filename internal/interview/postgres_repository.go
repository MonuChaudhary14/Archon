package interview

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) GetRandomUnansweredQuestion(ctx context.Context, userID int, difficulty string) (*Question, error) {
	query := `
		SELECT id, title, difficulty, expected_topics, time_limit_minutes, created_at 
		FROM questions 
		WHERE difficulty = $1 
		AND id NOT IN (
			SELECT question_id FROM interviews WHERE user_id = $2
		)
		ORDER BY RANDOM() LIMIT 1;
	`

	var q Question
	err := r.db.QueryRow(ctx, query, difficulty, userID).Scan(
		&q.ID, &q.Title, &q.Difficulty, &q.ExpectedTopics, &q.TimeLimitMinutes, &q.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch question: %w", err)
	}
	return &q, nil
}

func (r *postgresRepository) CreateInterview(ctx context.Context, userID int, questionID string) (string, error) {

	tx, err := r.db.Begin(ctx)

	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	insertInterviewQuery := `
		INSERT INTO interviews (user_id, question_id) 
		VALUES ($1, $2) 
		RETURNING id;
	`

	var interviewID string

	err = tx.QueryRow(ctx, insertInterviewQuery, userID, questionID).Scan(&interviewID)

	if err != nil {
		return "", fmt.Errorf("failed to create interview: %w", err)
	}

	payloadMap := map[string]interface{}{
		"interview_id": interviewID,
		"user_id":      userID,
		"question_id":  questionID,
	}

	payloadBytes, err := json.Marshal(payloadMap)

	if err != nil {
		return "", fmt.Errorf("failed to marshal outbox payload: %w", err)
	}

	insertOutboxQuery := `
		INSERT INTO outbox_events (aggregate_type, aggregate_id, type, payload)
		VALUES ('Interview', $1, 'INTERVIEW_STARTED', $2)
	`

	_, err = tx.Exec(ctx, insertOutboxQuery, interviewID, payloadBytes)

	if err != nil {
		return "", fmt.Errorf("failed to insert outbox event: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	return interviewID, nil
}

func (r *postgresRepository) GetInterviewByID(ctx context.Context, userID int, interviewID string) (*Interview, error) {
	query := `
		SELECT id, user_id, question_id, score, feedback, started_at, ended_at 
		FROM interviews 
		WHERE id = $1 AND user_id = $2;
	`
	var i Interview
	err := r.db.QueryRow(ctx, query, interviewID, userID).Scan(
		&i.ID, &i.UserID, &i.QuestionID, &i.Score, &i.Feedback, &i.StartedAt, &i.EndedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch interview: %w", err)
	}
	return &i, nil
}
