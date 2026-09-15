package quiz

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/MonuChaudhary14/Archon/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

type dbOption struct {
	ID          string `json:"id"`
	Text        string `json:"text"`
	IsCorrect   bool   `json:"is_correct"`
	Explanation string `json:"explanation"`
}

func parseQuestion(id, qText string, scenario sql.NullString, topicTag string, optionsBytes []byte) *models.QuizQuestion {
	var dbOpts []dbOption
	_ = json.Unmarshal(optionsBytes, &dbOpts)

	var pubOpts []models.QuizOption
	for _, o := range dbOpts {
		pubOpts = append(pubOpts, models.QuizOption{
			ID:   o.ID,
			Text: o.Text,
		})
	}

	scenarioStr := ""
	if scenario.Valid {
		scenarioStr = scenario.String
	}

	return &models.QuizQuestion{
		ID:       id,
		Question: qText,
		Scenario: scenarioStr,
		TopicTag: topicTag,
		Options:  pubOpts,
	}
}

func (r *postgresRepository) GetDailyChallenge(ctx context.Context) (*models.QuizQuestion, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM quiz_questions WHERE is_daily = TRUE`).Scan(&count)
	if err != nil || count == 0 {
		queryFallback := `SELECT id, question, scenario, topic_tag, options FROM quiz_questions ORDER BY id ASC LIMIT 1`
		var id, qText, topicTag string
		var scenario sql.NullString
		var optionsBytes []byte
		err = r.db.QueryRow(ctx, queryFallback).Scan(&id, &qText, &scenario, &topicTag, &optionsBytes)
		if err != nil {
			return nil, err
		}
		return parseQuestion(id, qText, scenario, topicTag, optionsBytes), nil
	}

	dayEpoch := int(time.Now().UTC().Unix() / 86400)
	offset := dayEpoch % count

	query := `
		SELECT id, question, scenario, topic_tag, options
		FROM quiz_questions
		WHERE is_daily = TRUE
		ORDER BY id ASC
		LIMIT 1 OFFSET $1
	`
	var id, qText, topicTag string
	var scenario sql.NullString
	var optionsBytes []byte

	err = r.db.QueryRow(ctx, query, offset).Scan(&id, &qText, &scenario, &topicTag, &optionsBytes)
	if err != nil {
		return nil, err
	}

	return parseQuestion(id, qText, scenario, topicTag, optionsBytes), nil
}

func (r *postgresRepository) VerifyDailyChallenge(ctx context.Context, userID int, questionID string, selectedOptionID string) (*models.VerifyDailyChallengeResponse, error) {
	query := `SELECT deck_id, options FROM quiz_questions WHERE id::text = $1`
	var deckID sql.NullString
	var optionsBytes []byte
	err := r.db.QueryRow(ctx, query, questionID).Scan(&deckID, &optionsBytes)
	if err != nil {
		return nil, errors.New("question not found")
	}

	var dbOpts []dbOption
	_ = json.Unmarshal(optionsBytes, &dbOpts)

	var correctOptID string
	var explanation string
	isCorrect := false

	for _, o := range dbOpts {
		if o.IsCorrect {
			correctOptID = o.ID
			explanation = o.Explanation
		}
		if o.ID == selectedOptionID && o.IsCorrect {
			isCorrect = true
		}
	}

	if userID > 0 {
		scorePercent := 0
		correctCount := 0
		if isCorrect {
			scorePercent = 100
			correctCount = 1
		}
		ansMap := map[string]string{questionID: selectedOptionID}
		answersJSON, _ := json.Marshal(ansMap)

		var targetDeckID *string
		if deckID.Valid && deckID.String != "" {
			d := deckID.String
			targetDeckID = &d
		}

		insertAttempt := `
			INSERT INTO quiz_attempts (user_id, deck_id, score_percent, correct_count, total_questions, time_spent_seconds, answers)
			VALUES ($1, $2, $3, $4, 1, 0, $5)
		`
		_, _ = r.db.Exec(ctx, insertAttempt, userID, targetDeckID, scorePercent, correctCount, answersJSON)
	}

	return &models.VerifyDailyChallengeResponse{
		IsCorrect:       isCorrect,
		CorrectOptionID: correctOptID,
		Explanation:     explanation,
	}, nil
}

func (r *postgresRepository) ListDecks(ctx context.Context, userID int) ([]models.QuizDeckItem, error) {
	query := `
		SELECT 
			d.id,
			d.title,
			d.description,
			d.difficulty,
			d.question_count,
			d.est_minutes,
			d.icon_name,
			d.category,
			COALESCE(MAX(a.score_percent), 0) AS completed_percent
		FROM quiz_decks d
		LEFT JOIN quiz_attempts a ON d.id = a.deck_id AND a.user_id = $1
		GROUP BY d.id
		ORDER BY d.created_at ASC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decks []models.QuizDeckItem
	for rows.Next() {
		var item models.QuizDeckItem
		err := rows.Scan(&item.ID, &item.Title, &item.Description, &item.Difficulty, &item.QuestionCount, &item.EstMinutes, &item.IconName, &item.Category, &item.CompletedPercent)
		if err == nil {
			decks = append(decks, item)
		}
	}
	if decks == nil {
		decks = []models.QuizDeckItem{}
	}
	return decks, nil
}

func (r *postgresRepository) GetDeckQuestions(ctx context.Context, deckID string) ([]models.QuizQuestion, error) {
	query := `
		SELECT id, question, scenario, topic_tag, options
		FROM quiz_questions
		WHERE deck_id = $1
	`
	rows, err := r.db.Query(ctx, query, deckID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []models.QuizQuestion
	for rows.Next() {
		var id, qText, topicTag string
		var scenario sql.NullString
		var optionsBytes []byte

		err := rows.Scan(&id, &qText, &scenario, &topicTag, &optionsBytes)
		if err != nil {
			continue
		}

		q := parseQuestion(id, qText, scenario, topicTag, optionsBytes)
		questions = append(questions, *q)
	}
	if questions == nil {
		questions = []models.QuizQuestion{}
	}
	return questions, nil
}

func (r *postgresRepository) SubmitDeckQuiz(ctx context.Context, userID int, deckID string, req models.SubmitDeckQuizRequest) (*models.SubmitDeckQuizResponse, error) {
	query := `SELECT id, options FROM quiz_questions WHERE deck_id = $1`
	rows, err := r.db.Query(ctx, query, deckID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	totalQuestions := 0
	correctCount := 0
	var review []models.QuestionReviewItem

	for rows.Next() {
		var qID string
		var optionsBytes []byte
		err := rows.Scan(&qID, &optionsBytes)
		if err != nil {
			continue
		}
		totalQuestions++

		var dbOpts []dbOption
		_ = json.Unmarshal(optionsBytes, &dbOpts)

		userOptionID := req.Answers[qID]
		var correctOptID string
		var explanation string
		isCorrect := false

		for _, o := range dbOpts {
			if o.IsCorrect {
				correctOptID = o.ID
				explanation = o.Explanation
			}
			if o.ID == userOptionID && o.IsCorrect {
				isCorrect = true
			}
		}

		if isCorrect {
			correctCount++
		}

		review = append(review, models.QuestionReviewItem{
			QuestionID:      qID,
			UserOptionID:    userOptionID,
			CorrectOptionID: correctOptID,
			IsCorrect:       isCorrect,
			Explanation:     explanation,
		})
	}

	scorePercent := 0
	if totalQuestions > 0 {
		scorePercent = (correctCount * 100) / totalQuestions
	}

	answersJSON, _ := json.Marshal(req.Answers)
	insertQuery := `
		INSERT INTO quiz_attempts (user_id, deck_id, score_percent, correct_count, total_questions, time_spent_seconds, answers)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, _ = r.db.Exec(ctx, insertQuery, userID, deckID, scorePercent, correctCount, totalQuestions, req.TimeSpentSeconds, answersJSON)

	return &models.SubmitDeckQuizResponse{
		DeckID:         deckID,
		TotalQuestions: totalQuestions,
		CorrectCount:   correctCount,
		ScorePercent:   scorePercent,
		Review:         review,
	}, nil
}
