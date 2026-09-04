package dashboard

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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

func (r *postgresRepository) GetDashboardData(ctx context.Context, userID int) (*models.DashboardOverviewResponse, error) {
	var avgScore sql.NullFloat64
	var completedInterviews int
	queryStats := `
		SELECT 
			COALESCE(AVG(score), 0),
			COUNT(id)
		FROM interviews
		WHERE user_id = $1 AND score IS NOT NULL AND score >= 0
	`
	err := r.db.QueryRow(ctx, queryStats, userID).Scan(&avgScore, &completedInterviews)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	var totalQuizzes int
	var avgQuizScore sql.NullFloat64
	queryQuizStats := `
		SELECT 
			COUNT(id),
			COALESCE(AVG(score_percent), 0)
		FROM quiz_attempts
		WHERE user_id = $1
	`
	_ = r.db.QueryRow(ctx, queryQuizStats, userID).Scan(&totalQuizzes, &avgQuizScore)

	interviewScoreVal := 0.0
	if avgScore.Valid && completedInterviews > 0 {
		interviewScoreVal = avgScore.Float64
	}

	quizAccVal := 0.0
	if avgQuizScore.Valid && totalQuizzes > 0 {
		quizAccVal = avgQuizScore.Float64
	}

	readinessScore := 0
	if completedInterviews > 0 && totalQuizzes > 0 {
		readinessScore = int((interviewScoreVal * 0.60) + (quizAccVal * 0.40))
	} else if completedInterviews > 0 {
		readinessScore = int(interviewScoreVal)
	} else if totalQuizzes > 0 {
		readinessScore = int(quizAccVal)
	}

	scoreHistory := r.getScoreHistory(ctx, userID)

	streakDays := r.calculateStreak(ctx, userID)

	stats := models.DashboardStats{
		ReadinessScore:      readinessScore,
		ReadinessChange:     0.0,
		CompletedInterviews: completedInterviews,
		TotalQuizzesTaken:   totalQuizzes,
		QuizAccuracy:        int(quizAccVal),
		StreakDays:          streakDays,
		ScoreHistory:        scoreHistory,
	}

	recScenario := r.getRecommendedScenario(ctx, userID)
	competencies := r.getCompetencies(ctx, userID)

	recentInterviews, err := r.getRecentInterviews(ctx, userID)
	if err != nil {
		recentInterviews = []models.RecentInterviewSummary{}
	}

	roadmap := r.getRoadmap(ctx, userID)

	return &models.DashboardOverviewResponse{
		Stats:               stats,
		RecommendedScenario: recScenario,
		Competencies:        competencies,
		RecentInterviews:    recentInterviews,
		Roadmap:             roadmap,
	}, nil
}

func (r *postgresRepository) getScoreHistory(ctx context.Context, userID int) []int {
	query := `
		SELECT score
		FROM interviews
		WHERE user_id = $1 AND score IS NOT NULL AND score >= 0
		ORDER BY started_at ASC
		LIMIT 10
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return []int{}
	}
	defer rows.Close()

	var history []int
	for rows.Next() {
		var s int
		if err := rows.Scan(&s); err == nil {
			history = append(history, s)
		}
	}
	if history == nil {
		history = []int{}
	}
	return history
}

func (r *postgresRepository) getRecommendedScenario(ctx context.Context, userID int) models.RecommendedScenario {
	query := `
		SELECT q.id, q.title, q.difficulty, q.time_limit_minutes, q.expected_topics
		FROM questions q
		WHERE q.deleted_at IS NULL
		AND q.id NOT IN (
			SELECT question_id FROM interviews WHERE user_id = $1 AND score IS NOT NULL AND score >= 0
		)
		ORDER BY RANDOM()
		LIMIT 1
	`
	var qID, title, diff string
	var timeLimit int
	var topics []string

	err := r.db.QueryRow(ctx, query, userID).Scan(&qID, &title, &diff, &timeLimit, &topics)
	if err != nil {
		fallbackQuery := `
			SELECT id, title, difficulty, time_limit_minutes, expected_topics
			FROM questions
			WHERE deleted_at IS NULL
			ORDER BY RANDOM()
			LIMIT 1
		`
		_ = r.db.QueryRow(ctx, fallbackQuery).Scan(&qID, &title, &diff, &timeLimit, &topics)
	}

	if topics == nil {
		topics = []string{}
	}

	return models.RecommendedScenario{
		ID:         qID,
		Title:      title,
		Desc:       fmt.Sprintf("Practice full system architecture design for %s.", title),
		Difficulty: strings.ToLower(diff),
		EstTime:    fmt.Sprintf("%d mins", timeLimit),
		Topics:     topics,
	}
}

func (r *postgresRepository) getCompetencies(ctx context.Context, userID int) []models.CompetencyMetric {
	query := `
		SELECT feedback
		FROM interviews
		WHERE user_id = $1 AND score IS NOT NULL AND score >= 0 AND feedback IS NOT NULL
		ORDER BY started_at DESC
		LIMIT 10
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return []models.CompetencyMetric{}
	}
	defer rows.Close()

	type rubricItem struct {
		Name  string `json:"name"`
		Score int    `json:"score"`
	}
	type fbPayload struct {
		Rubrics []rubricItem `json:"rubrics"`
	}

	scoresMap := make(map[string][]int)
	for rows.Next() {
		var fbBytes []byte
		if err := rows.Scan(&fbBytes); err == nil && len(fbBytes) > 0 {
			var fb fbPayload
			if err := json.Unmarshal(fbBytes, &fb); err == nil {
				for _, rub := range fb.Rubrics {
					scoresMap[rub.Name] = append(scoresMap[rub.Name], rub.Score)
				}
			}
		}
	}

	standardLabels := []string{
		"Requirements & Scope",
		"Capacity Estimation",
		"High-Level Architecture",
		"Scalability & Deep Dive",
	}

	var result []models.CompetencyMetric
	for _, label := range standardLabels {
		vals := scoresMap[label]
		val := 0
		if len(vals) > 0 {
			sum := 0
			for _, v := range vals {
				sum += v
			}
			val = sum / len(vals)
		}
		result = append(result, models.CompetencyMetric{
			Label: label,
			Value: val,
		})
	}
	return result
}

func (r *postgresRepository) getRoadmap(ctx context.Context, userID int) []models.RoadmapTopic {
	return []models.RoadmapTopic{}
}

func (r *postgresRepository) calculateStreak(ctx context.Context, userID int) int {
	query := `
		SELECT date_trunc('day', created_at)::date AS act_date
		FROM (
			SELECT started_at AS created_at FROM interviews WHERE user_id = $1
			UNION ALL
			SELECT created_at FROM quiz_attempts WHERE user_id = $1
		) activity
		GROUP BY act_date
		ORDER BY act_date DESC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return 0
	}
	defer rows.Close()

	streak := 0
	today := time.Now().UTC().Truncate(24 * time.Hour)

	for rows.Next() {
		var actDate time.Time
		if err := rows.Scan(&actDate); err != nil {
			break
		}
		actDate = actDate.UTC().Truncate(24 * time.Hour)
		diffDays := int(today.Sub(actDate).Hours() / 24)
		if diffDays == streak || diffDays == streak+1 {
			streak++
		} else if diffDays > streak+1 {
			break
		}
	}
	return streak
}

func (r *postgresRepository) getRecentInterviews(ctx context.Context, userID int) ([]models.RecentInterviewSummary, error) {
	query := `
		SELECT 
			i.id,
			q.title,
			q.difficulty,
			i.score,
			i.started_at,
			i.ended_at
		FROM interviews i
		JOIN questions q ON i.question_id = q.id
		WHERE i.user_id = $1
		ORDER BY i.started_at DESC
		LIMIT 5
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []models.RecentInterviewSummary
	for rows.Next() {
		var id string
		var title, diff string
		var score sql.NullInt32
		var startedAt time.Time
		var endedAt sql.NullTime

		if err := rows.Scan(&id, &title, &diff, &score, &startedAt, &endedAt); err != nil {
			continue
		}

		status := "completed"
		var scorePtr *int
		if score.Valid {
			if score.Int32 == -1 {
				status = "failed"
			} else {
				v := int(score.Int32)
				scorePtr = &v
			}
		} else {
			status = "evaluating"
		}

		durationStr := "-"
		if endedAt.Valid {
			d := endedAt.Time.Sub(startedAt)
			durationStr = fmt.Sprintf("%d mins", int(d.Minutes()))
		}

		summaries = append(summaries, models.RecentInterviewSummary{
			ID:         id,
			SessionID:  id,
			Title:      title,
			Difficulty: diff,
			Score:      scorePtr,
			Status:     status,
			Date:       startedAt,
			Duration:   durationStr,
		})
	}
	if summaries == nil {
		summaries = []models.RecentInterviewSummary{}
	}
	return summaries, nil
}

