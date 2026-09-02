package settings

import (
	"context"
	"database/sql"

	"github.com/MonuChaudhary14/Archon/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
)

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) GetSettings(ctx context.Context, userID int) (*models.UserSettings, error) {
	var name, email string
	_ = r.db.QueryRow(ctx, "SELECT name, email FROM users WHERE id = $1", userID).Scan(&name, &email)

	query := `
		SELECT 
			target_level,
			years_of_experience,
			primary_stack,
			target_companies,
			interviewer_strictness,
			feedback_style,
			enable_proactive_hints,
			enable_voice_interview,
			canvas_grid_type,
			snap_to_grid,
			auto_save_interval_seconds,
			export_format,
			weekly_interview_target,
			email_notifications,
			weekly_report_digest
		FROM user_settings
		WHERE user_id = $1
	`

	var s models.UserSettings
	var targetCompanies []string

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&s.TargetLevel,
		&s.YearsOfExperience,
		&s.PrimaryStack,
		pq.Array(&targetCompanies),
		&s.InterviewerStrictness,
		&s.FeedbackStyle,
		&s.EnableProactiveHints,
		&s.EnableVoiceInterview,
		&s.CanvasGridType,
		&s.SnapToGrid,
		&s.AutoSaveIntervalSeconds,
		&s.ExportFormat,
		&s.WeeklyInterviewTarget,
		&s.EmailNotifications,
		&s.WeeklyReportDigest,
	)

	if err == sql.ErrNoRows || err != nil {
		return &models.UserSettings{
			FullName:                name,
			Email:                   email,
			TargetLevel:             "senior",
			YearsOfExperience:       5,
			PrimaryStack:            "Go",
			TargetCompanies:         []string{"Google", "Meta", "Stripe", "Uber"},
			InterviewerStrictness:   "standard",
			FeedbackStyle:           "socratic",
			EnableProactiveHints:    true,
			EnableVoiceInterview:    false,
			CanvasGridType:          "dots",
			SnapToGrid:              true,
			AutoSaveIntervalSeconds: 15,
			ExportFormat:            "svg",
			WeeklyInterviewTarget:   3,
			EmailNotifications:      true,
			WeeklyReportDigest:      true,
		}, nil
	}

	s.FullName = name
	s.Email = email
	s.TargetCompanies = targetCompanies

	return &s, nil
}

func (r *postgresRepository) UpdateSettings(ctx context.Context, userID int, s models.UserSettings) error {
	query := `
		INSERT INTO user_settings (
			user_id,
			target_level,
			years_of_experience,
			primary_stack,
			target_companies,
			interviewer_strictness,
			feedback_style,
			enable_proactive_hints,
			enable_voice_interview,
			canvas_grid_type,
			snap_to_grid,
			auto_save_interval_seconds,
			export_format,
			weekly_interview_target,
			email_notifications,
			weekly_report_digest,
			updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, NOW()
		)
		ON CONFLICT (user_id) DO UPDATE SET
			target_level = EXCLUDED.target_level,
			years_of_experience = EXCLUDED.years_of_experience,
			primary_stack = EXCLUDED.primary_stack,
			target_companies = EXCLUDED.target_companies,
			interviewer_strictness = EXCLUDED.interviewer_strictness,
			feedback_style = EXCLUDED.feedback_style,
			enable_proactive_hints = EXCLUDED.enable_proactive_hints,
			enable_voice_interview = EXCLUDED.enable_voice_interview,
			canvas_grid_type = EXCLUDED.canvas_grid_type,
			snap_to_grid = EXCLUDED.snap_to_grid,
			auto_save_interval_seconds = EXCLUDED.auto_save_interval_seconds,
			export_format = EXCLUDED.export_format,
			weekly_interview_target = EXCLUDED.weekly_interview_target,
			email_notifications = EXCLUDED.email_notifications,
			weekly_report_digest = EXCLUDED.weekly_report_digest,
			updated_at = NOW()
	`

	_, err := r.db.Exec(ctx, query,
		userID,
		s.TargetLevel,
		s.YearsOfExperience,
		s.PrimaryStack,
		pq.Array(s.TargetCompanies),
		s.InterviewerStrictness,
		s.FeedbackStyle,
		s.EnableProactiveHints,
		s.EnableVoiceInterview,
		s.CanvasGridType,
		s.SnapToGrid,
		s.AutoSaveIntervalSeconds,
		s.ExportFormat,
		s.WeeklyInterviewTarget,
		s.EmailNotifications,
		s.WeeklyReportDigest,
	)
	return err
}
