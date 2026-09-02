package models

type UserSettings struct {
	FullName                string   `json:"full_name"`
	Email                   string   `json:"email"`
	TargetLevel             string   `json:"target_level"`
	YearsOfExperience       int      `json:"years_of_experience"`
	PrimaryStack            string   `json:"primary_stack"`
	TargetCompanies         []string `json:"target_companies"`
	InterviewerStrictness   string   `json:"interviewer_strictness"`
	FeedbackStyle           string   `json:"feedback_style"`
	EnableProactiveHints    bool     `json:"enable_proactive_hints"`
	EnableVoiceInterview    bool     `json:"enable_voice_interview"`
	CanvasGridType          string   `json:"canvas_grid_type"`
	SnapToGrid              bool     `json:"snap_to_grid"`
	AutoSaveIntervalSeconds int      `json:"auto_save_interval_seconds"`
	ExportFormat            string   `json:"export_format"`
	WeeklyInterviewTarget   int      `json:"weekly_interview_target"`
	EmailNotifications      bool     `json:"email_notifications"`
	WeeklyReportDigest      bool     `json:"weekly_report_digest"`
}
