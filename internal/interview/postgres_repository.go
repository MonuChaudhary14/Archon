package interview

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	getRandomUnansweredQuestionQuery = `
		SELECT id, title, difficulty, expected_topics, time_limit_minutes, created_at 
		FROM questions 
		WHERE difficulty = $1 
		AND id NOT IN (
			SELECT question_id FROM interviews WHERE user_id = $2
		)
		ORDER BY RANDOM() LIMIT 1;
	`
	getQuestionsQuery = `
		SELECT id, title, difficulty, expected_topics, time_limit_minutes, created_at 
		FROM questions 
		WHERE deleted_at IS NULL
		ORDER BY title ASC;
	`
	getQuestionByIDQuery = `
		SELECT id, title, difficulty, expected_topics, time_limit_minutes, created_at 
		FROM questions 
		WHERE id = $1 AND deleted_at IS NULL;
	`
	insertInterviewQuery = `
		INSERT INTO interviews (user_id, question_id) 
		VALUES ($1, $2) 
		RETURNING id;
	`
	insertOutboxStartedQuery = `
		INSERT INTO outbox_events (aggregate_type, aggregate_id, type, payload)
		VALUES ('Interview', $1, 'INTERVIEW_STARTED', $2);
	`
	insertOutboxSubmittedQuery = `
		INSERT INTO outbox_events (aggregate_type, aggregate_id, type, payload)
		VALUES ('Interview', $1, 'INTERVIEW_SUBMITTED', $2);
	`
	getInterviewByIDQuery = `
		SELECT id, user_id, question_id, score, feedback, started_at, ended_at 
		FROM interviews 
		WHERE id = $1 AND user_id = $2;
	`
	submitInterviewQuery = `
		UPDATE interviews 
		SET ended_at = NOW() 
		WHERE id = $1 AND user_id = $2 AND ended_at IS NULL;
	`
	getInterviewsByUserIDQuery = `
		SELECT id, user_id, question_id, score, feedback, started_at, ended_at 
		FROM interviews 
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY started_at DESC;
	`
)

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) GetRandomUnansweredQuestion(ctx context.Context, userID int, difficulty string) (*Question, error) {
	var q Question
	err := r.db.QueryRow(ctx, getRandomUnansweredQuestionQuery, difficulty, userID).Scan(
		&q.ID, &q.Title, &q.Difficulty, &q.ExpectedTopics, &q.TimeLimitMinutes, &q.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch question: %w", err)
	}
	return &q, nil
}

func (r *postgresRepository) GetQuestions(ctx context.Context) ([]*Question, error) {
	rows, err := r.db.Query(ctx, getQuestionsQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch questions: %w", err)
	}
	defer rows.Close()

	var questions []*Question
	for rows.Next() {
		var q Question
		err := rows.Scan(
			&q.ID, &q.Title, &q.Difficulty, &q.ExpectedTopics, &q.TimeLimitMinutes, &q.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan question: %w", err)
		}
		questions = append(questions, &q)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading question rows: %w", err)
	}

	return questions, nil
}

func (r *postgresRepository) GetQuestionByID(ctx context.Context, id string) (*Question, error) {
	var q Question
	err := r.db.QueryRow(ctx, getQuestionByIDQuery, id).Scan(
		&q.ID, &q.Title, &q.Difficulty, &q.ExpectedTopics, &q.TimeLimitMinutes, &q.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch question by id: %w", err)
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

	_, err = tx.Exec(ctx, insertOutboxStartedQuery, interviewID, payloadBytes)

	if err != nil {
		return "", fmt.Errorf("failed to insert outbox event: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	return interviewID, nil
}

func (r *postgresRepository) GetInterviewByID(ctx context.Context, userID int, interviewID string) (*Interview, error) {
	var i Interview
	err := r.db.QueryRow(ctx, getInterviewByIDQuery, interviewID, userID).Scan(
		&i.ID, &i.UserID, &i.QuestionID, &i.Score, &i.Feedback, &i.StartedAt, &i.EndedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch interview: %w", err)
	}
	return &i, nil
}

func (r *postgresRepository) SubmitInterview(ctx context.Context, userID int, interviewID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	res, err := tx.Exec(ctx, submitInterviewQuery, interviewID, userID)
	if err != nil {
		return fmt.Errorf("failed to update interview ended_at: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("interview already completed or not found")
	}

	payloadMap := map[string]interface{}{
		"interview_id": interviewID,
		"user_id":      userID,
		"status":       "SUBMITTED",
	}
	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		return fmt.Errorf("failed to marshal outbox payload: %w", err)
	}

	_, err = tx.Exec(ctx, insertOutboxSubmittedQuery, interviewID, payloadBytes)
	if err != nil {
		return fmt.Errorf("failed to insert outbox event: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *postgresRepository) GetInterviewsByUserID(ctx context.Context, userID int) ([]*Interview, error) {
	rows, err := r.db.Query(ctx, getInterviewsByUserIDQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch interviews: %w", err)
	}
	defer rows.Close()

	var interviews []*Interview
	for rows.Next() {
		var i Interview
		err := rows.Scan(
			&i.ID, &i.UserID, &i.QuestionID, &i.Score, &i.Feedback, &i.StartedAt, &i.EndedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan interview: %w", err)
		}
		interviews = append(interviews, &i)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading interview rows: %w", err)
	}

	return interviews, nil
}

