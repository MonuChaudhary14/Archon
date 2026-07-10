package interview

import (
	"context"
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
	query := `
		INSERT INTO interviews (user_id, question_id) 
		VALUES ($1, $2) 
		RETURNING id;
	`
	
	var interviewID string
	err := r.db.QueryRow(ctx, query, userID, questionID).Scan(&interviewID)
	if err != nil {
		return "", fmt.Errorf("failed to create interview: %w", err)
	}
	
	return interviewID, nil
}
