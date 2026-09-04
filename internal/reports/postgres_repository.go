package reports

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

type feedbackJSON struct {
	InterviewerSummary string                  `json:"interviewer_summary"`
	Rubrics            []models.RubricCategory `json:"rubrics"`
	Strengths          []string                `json:"strengths"`
	Weaknesses         []string                `json:"weaknesses"`
	Recommendations    []string                `json:"recommendations"`
	DiagramComponents  []string                `json:"diagram_components"`
}

func (r *postgresRepository) ListReports(ctx context.Context, userID int, search string, difficulty string, page int, limit int) (*models.ReportsListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	var totalReports int
	var avgScore sql.NullFloat64
	var maxScore sql.NullInt32

	summaryQuery := `
		SELECT COUNT(id), COALESCE(AVG(score), 0), COALESCE(MAX(score), 0)
		FROM interviews
		WHERE user_id = $1 AND score IS NOT NULL AND score >= 0
	`
	_ = r.db.QueryRow(ctx, summaryQuery, userID).Scan(&totalReports, &avgScore, &maxScore)

	whereClause := "WHERE i.user_id = $1 AND i.score IS NOT NULL AND i.score >= 0"
	args := []interface{}{userID}
	argIdx := 2

	if strings.TrimSpace(search) != "" {
		whereClause += fmt.Sprintf(" AND (q.title ILIKE $%d OR i.feedback::text ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+strings.TrimSpace(search)+"%")
		argIdx++
	}

	if strings.TrimSpace(difficulty) != "" {
		whereClause += fmt.Sprintf(" AND LOWER(q.difficulty) = $%d", argIdx)
		args = append(args, strings.ToLower(strings.TrimSpace(difficulty)))
		argIdx++
	}

	query := fmt.Sprintf(`
		SELECT 
			i.id,
			q.title,
			q.difficulty,
			i.score,
			i.started_at,
			i.ended_at,
			i.feedback
		FROM interviews i
		JOIN questions q ON i.question_id = q.id
		%s
		ORDER BY i.started_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []models.ReportListItem
	for rows.Next() {
		var id, title, diff string
		var score int
		var startedAt time.Time
		var endedAt sql.NullTime
		var feedbackBytes []byte

		if err := rows.Scan(&id, &title, &diff, &score, &startedAt, &endedAt, &feedbackBytes); err != nil {
			continue
		}

		durationStr := "-"
		if endedAt.Valid {
			d := endedAt.Time.Sub(startedAt)
			durationStr = fmt.Sprintf("%d mins", int(d.Minutes()))
		}

		summaryText := ""
		if len(feedbackBytes) > 0 {
			var fb feedbackJSON
			if err := json.Unmarshal(feedbackBytes, &fb); err == nil && fb.InterviewerSummary != "" {
				summaryText = fb.InterviewerSummary
			}
		}

		percentile := 0
		if score > 85 {
			percentile = 94
		} else if score > 75 {
			percentile = 88
		} else if score > 60 {
			percentile = 70
		} else if score > 40 {
			percentile = 50
		} else if score > 0 {
			percentile = 25
		}

		reports = append(reports, models.ReportListItem{
			ID:                 id,
			SessionID:          id,
			Title:              title,
			Difficulty:         strings.ToLower(diff),
			OverallScore:       score,
			Percentile:         percentile,
			Duration:           durationStr,
			Date:               startedAt,
			InterviewerSummary: summaryText,
		})
	}

	if reports == nil {
		reports = []models.ReportListItem{}
	}

	highestVal := 0
	if maxScore.Valid {
		highestVal = int(maxScore.Int32)
	}
	avgVal := 0
	if avgScore.Valid {
		avgVal = int(avgScore.Float64)
	}

	return &models.ReportsListResponse{
		Summary: models.ReportsSummary{
			TotalReports: totalReports,
			AverageScore: avgVal,
			HighestScore: highestVal,
		},
		Reports: reports,
	}, nil
}

func (r *postgresRepository) GetReportDetail(ctx context.Context, userID int, sessionID string) (*models.DetailedReportResponse, bool, error) {
	query := `
		SELECT 
			i.id,
			q.title,
			q.difficulty,
			i.score,
			i.started_at,
			i.ended_at,
			i.feedback
		FROM interviews i
		JOIN questions q ON i.question_id = q.id
		WHERE i.id = $1 AND i.user_id = $2
	`

	var id, title, diff string
	var score sql.NullInt32
	var startedAt time.Time
	var endedAt sql.NullTime
	var feedbackBytes []byte

	err := r.db.QueryRow(ctx, query, sessionID, userID).Scan(&id, &title, &diff, &score, &startedAt, &endedAt, &feedbackBytes)
	if err != nil {
		return nil, false, err
	}

	if !score.Valid || len(feedbackBytes) == 0 {
		return nil, true, nil
	}

	if score.Int32 == -1 {
		var errPayload struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(feedbackBytes, &errPayload)
		errMsg := "evaluation failed"
		if errPayload.Error != "" {
			errMsg = errPayload.Error
		}
		return nil, false, fmt.Errorf("%s", errMsg)
	}

	overallScore := int(score.Int32)
	durationStr := "-"
	if endedAt.Valid {
		d := endedAt.Time.Sub(startedAt)
		durationStr = fmt.Sprintf("%d mins", int(d.Minutes()))
	}

	var fb feedbackJSON
	_ = json.Unmarshal(feedbackBytes, &fb)

	if fb.Rubrics == nil {
		fb.Rubrics = []models.RubricCategory{}
	}
	if fb.Strengths == nil {
		fb.Strengths = []string{}
	}
	if fb.Weaknesses == nil {
		fb.Weaknesses = []string{}
	}
	if fb.Recommendations == nil {
		fb.Recommendations = []string{}
	}
	if fb.DiagramComponents == nil {
		fb.DiagramComponents = []string{}
	}
	if fb.InterviewerSummary == "" {
		if overallScore == 0 {
			fb.InterviewerSummary = "Interview submitted without candidate participation or whiteboard architecture diagrams."
		} else {
			fb.InterviewerSummary = "Evaluation completed."
		}
	}

	percentile := 0
	if overallScore > 85 {
		percentile = 94
	} else if overallScore > 75 {
		percentile = 88
	} else if overallScore > 60 {
		percentile = 70
	} else if overallScore > 40 {
		percentile = 50
	} else if overallScore > 0 {
		percentile = 25
	}

	var endedAtPtr *time.Time
	if endedAt.Valid {
		endedAtPtr = &endedAt.Time
	}

	return &models.DetailedReportResponse{
		SessionID:          id,
		Title:              title,
		Difficulty:         strings.ToLower(diff),
		OverallScore:       overallScore,
		Percentile:         percentile,
		Duration:           durationStr,
		StartedAt:          startedAt,
		EndedAt:            endedAtPtr,
		InterviewerSummary: fb.InterviewerSummary,
		Rubrics:            fb.Rubrics,
		Strengths:          fb.Strengths,
		Weaknesses:         fb.Weaknesses,
		Recommendations:    fb.Recommendations,
		DiagramComponents:  fb.DiagramComponents,
	}, false, nil
}
