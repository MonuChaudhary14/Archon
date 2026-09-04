package analytics

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/MonuChaudhary14/Archon/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) GetAnalytics(ctx context.Context, userID int, timeRange string) (*models.AnalyticsResponse, error) {
	var targetLevel string
	_ = r.db.QueryRow(ctx, "SELECT target_level FROM user_settings WHERE user_id = $1", userID).Scan(&targetLevel)
	if targetLevel == "" {
		targetLevel = "Senior Software Engineer"
	} else {
		targetLevel = targetLevel + " Software Engineer"
	}

	var avgScore sql.NullFloat64
	var countInterviews int
	_ = r.db.QueryRow(ctx, "SELECT COALESCE(AVG(score), 0), COUNT(id) FROM interviews WHERE user_id = $1 AND score >= 0", userID).Scan(&avgScore, &countInterviews)

	var avgQuizScore sql.NullFloat64
	var countQuizzes int
	_ = r.db.QueryRow(ctx, "SELECT COALESCE(AVG(score_percent), 0), COUNT(id) FROM quiz_attempts WHERE user_id = $1", userID).Scan(&avgQuizScore, &countQuizzes)

	interviewScoreVal := 0.0
	if avgScore.Valid && countInterviews > 0 {
		interviewScoreVal = avgScore.Float64
	}
	quizAccVal := 0.0
	if avgQuizScore.Valid && countQuizzes > 0 {
		quizAccVal = avgQuizScore.Float64
	}

	readinessScore := 0
	if countInterviews > 0 && countQuizzes > 0 {
		readinessScore = int((interviewScoreVal * 0.60) + (quizAccVal * 0.40))
	} else if countInterviews > 0 {
		readinessScore = int(interviewScoreVal)
	} else if countQuizzes > 0 {
		readinessScore = int(quizAccVal)
	}

	totalHours := float64(countInterviews)*0.75 + float64(countQuizzes)*0.15

	trend := r.getTrendPoints(ctx, userID)
	domainMastery := r.getDomainMastery(ctx, userID)
	heatmapDays := r.getHeatmapDays(ctx, userID)
	pitfalls := r.getPitfalls(ctx, userID)

	percentile := 0
	if readinessScore > 85 {
		percentile = 94
	} else if readinessScore > 75 {
		percentile = 88
	} else if readinessScore > 60 {
		percentile = 70
	} else if readinessScore > 40 {
		percentile = 50
	} else if readinessScore > 0 {
		percentile = 25
	}

	return &models.AnalyticsResponse{
		ReadinessScore:    readinessScore,
		Percentile:        percentile,
		TotalHoursTrained: totalHours,
		AvgScore:          int(interviewScoreVal),
		TargetLevel:       targetLevel,
		Trend:             trend,
		DomainMastery:     domainMastery,
		HeatmapDays:       heatmapDays,
		Pitfalls:          pitfalls,
	}, nil
}

func (r *postgresRepository) getTrendPoints(ctx context.Context, userID int) []models.TrendPoint {
	query := `
		SELECT TO_CHAR(started_at, 'Mon DD'), score
		FROM interviews
		WHERE user_id = $1 AND score IS NOT NULL AND score >= 0
		ORDER BY started_at ASC
		LIMIT 10
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return []models.TrendPoint{}
	}
	defer rows.Close()

	var points []models.TrendPoint
	for rows.Next() {
		var label string
		var val int
		if err := rows.Scan(&label, &val); err == nil {
			points = append(points, models.TrendPoint{Label: label, Value: val})
		}
	}
	if points == nil {
		points = []models.TrendPoint{}
	}
	return points
}

func (r *postgresRepository) getDomainMastery(ctx context.Context, userID int) []models.DomainMastery {
	domains := []string{
		"Requirements & Scope",
		"Capacity Estimation",
		"High-Level Architecture",
		"Scalability & Deep Dive",
	}

	query := `
		SELECT feedback
		FROM interviews
		WHERE user_id = $1 AND score IS NOT NULL AND score >= 0 AND feedback IS NOT NULL
		ORDER BY started_at DESC
		LIMIT 10
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return []models.DomainMastery{}
	}
	defer rows.Close()

	type rubricItem struct {
		Name  string `json:"name"`
		Score int    `json:"score"`
	}
	type fbPayload struct {
		Rubrics []rubricItem `json:"rubrics"`
	}

	domainScores := make(map[string][]int)
	domainCounts := make(map[string]int)

	for rows.Next() {
		var fbBytes []byte
		if err := rows.Scan(&fbBytes); err == nil && len(fbBytes) > 0 {
			var fb fbPayload
			if err := json.Unmarshal(fbBytes, &fb); err == nil {
				for _, rub := range fb.Rubrics {
					domainScores[rub.Name] = append(domainScores[rub.Name], rub.Score)
					domainCounts[rub.Name]++
				}
			}
		}
	}

	var result []models.DomainMastery
	for _, domain := range domains {
		scores := domainScores[domain]
		avgScore := 0
		if len(scores) > 0 {
			sum := 0
			for _, s := range scores {
				sum += s
			}
			avgScore = sum / len(scores)
		}
		result = append(result, models.DomainMastery{
			Domain:         domain,
			Score:          avgScore,
			Benchmark:      75,
			QuestionsCount: domainCounts[domain],
		})
	}
	return result
}

func (r *postgresRepository) getPitfalls(ctx context.Context, userID int) []models.PitfallInsight {
	query := `
		SELECT feedback
		FROM interviews
		WHERE user_id = $1 AND score IS NOT NULL AND score >= 0 AND feedback IS NOT NULL
		ORDER BY started_at DESC
		LIMIT 10
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return []models.PitfallInsight{}
	}
	defer rows.Close()

	type fbPayload struct {
		Weaknesses []string `json:"weaknesses"`
	}

	var insights []models.PitfallInsight
	idx := 1

	for rows.Next() {
		var fbBytes []byte
		if err := rows.Scan(&fbBytes); err == nil && len(fbBytes) > 0 {
			var fb fbPayload
			if err := json.Unmarshal(fbBytes, &fb); err == nil {
				for _, w := range fb.Weaknesses {
					if len(w) > 0 && idx <= 5 {
						insights = append(insights, models.PitfallInsight{
							ID:          fmt.Sprintf("pit-%d", idx),
							Category:    "warning",
							Title:       w,
							Description: w,
							Impact:      "Medium",
							Frequency:   "1 session",
						})
						idx++
					}
				}
			}
		}
	}
	if insights == nil {
		insights = []models.PitfallInsight{}
	}
	return insights
}

func (r *postgresRepository) getHeatmapDays(ctx context.Context, userID int) []models.HeatmapDay {
	query := `
		SELECT 
			TO_CHAR(act_date, 'YYYY-MM-DD') AS day_str,
			COUNT(*) AS cnt
		FROM (
			SELECT DATE(started_at) AS act_date FROM interviews WHERE user_id = $1
			UNION ALL
			SELECT DATE(created_at) AS act_date FROM quiz_attempts WHERE user_id = $1
		) sub
		GROUP BY act_date
		ORDER BY act_date DESC
		LIMIT 90
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return []models.HeatmapDay{}
	}
	defer rows.Close()

	var days []models.HeatmapDay
	for rows.Next() {
		var dayStr string
		var cnt int
		if err := rows.Scan(&dayStr, &cnt); err == nil {
			days = append(days, models.HeatmapDay{Date: dayStr, Count: cnt})
		}
	}
	if days == nil {
		days = []models.HeatmapDay{}
	}
	return days
}

