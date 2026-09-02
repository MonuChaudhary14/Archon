package dashboard

import (
	"context"
	"database/sql"
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

	mockScoreVal := 75.0
	if avgScore.Valid && completedInterviews > 0 {
		mockScoreVal = avgScore.Float64
	}

	quizAccVal := 80.0
	if avgQuizScore.Valid && totalQuizzes > 0 {
		quizAccVal = avgQuizScore.Float64
	}

	readinessScore := int((mockScoreVal * 0.60) + (quizAccVal * 0.40))

	scoreHistory := []int{62, 65, 70, 68, 74, 76, readinessScore}

	streakDays := r.calculateStreak(ctx, userID)

	stats := models.DashboardStats{
		ReadinessScore:      readinessScore,
		ReadinessChange:     4.5,
		CompletedInterviews: completedInterviews,
		TotalQuizzesTaken:   totalQuizzes,
		QuizAccuracy:        int(quizAccVal),
		StreakDays:          streakDays,
		ScoreHistory:        scoreHistory,
	}

	recScenario := models.RecommendedScenario{
		ID:         "global-cdn",
		Title:      "Design a Global CDN & Edge Video Cache",
		Desc:       "Architect a multi-region edge caching layer supporting 50M concurrent video streams.",
		Difficulty: "hard",
		EstTime:    "45 mins",
		Topics:     []string{"Geo-DNS", "Consistent Hashing", "Origin Shield", "Edge Caching"},
	}

	competencies := []models.CompetencyMetric{
		{Label: "Scalability & Sharding", Value: 84},
		{Label: "Storage & Schemas", Value: 76},
		{Label: "Fault Tolerance", Value: 68},
		{Label: "API & Protocols", Value: 92},
		{Label: "Capacity Estimation", Value: 74},
	}

	recentInterviews, err := r.getRecentInterviews(ctx, userID)
	if err != nil {
		recentInterviews = []models.RecentInterviewSummary{}
	}

	roadmap := []models.RoadmapTopic{
		{
			ID:               "caching",
			Title:            "Distributed Caching & Invalidation",
			Category:         "Performance",
			CompletedLessons: 6,
			TotalLessons:     6,
			Mastery:          95,
			Status:           "completed",
		},
		{
			ID:               "sharding",
			Title:            "Database Sharding & Partitioning",
			Category:         "Storage",
			CompletedLessons: 4,
			TotalLessons:     8,
			Mastery:          50,
			Status:           "in_progress",
		},
	}

	return &models.DashboardOverviewResponse{
		Stats:               stats,
		RecommendedScenario: recScenario,
		Competencies:        competencies,
		RecentInterviews:    recentInterviews,
		Roadmap:             roadmap,
	}, nil
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
		return 1
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
	if streak == 0 {
		return 1
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
			i.started_at
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

		if err := rows.Scan(&id, &title, &diff, &score, &startedAt); err != nil {
			continue
		}

		status := "completed"
		var scorePtr *int
		if score.Valid {
			v := int(score.Int32)
			scorePtr = &v
		} else {
			status = "evaluating"
		}

		summaries = append(summaries, models.RecentInterviewSummary{
			ID:         "int-" + id[:8],
			SessionID:  id,
			Title:      title,
			Difficulty: diff,
			Score:      scorePtr,
			Status:     status,
			Date:       startedAt,
			Duration:   "45 mins",
		})
	}
	return summaries, nil
}
