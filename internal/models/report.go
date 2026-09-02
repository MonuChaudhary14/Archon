package models

import "time"

type RubricCategory struct {
	Name           string   `json:"name"`
	Score          int      `json:"score"`
	Weight         int      `json:"weight"`
	Summary        string   `json:"summary"`
	FeedbackPoints []string `json:"feedback_points"`
}

type DetailedReportResponse struct {
	SessionID          string           `json:"session_id"`
	Title              string           `json:"title"`
	Difficulty         string           `json:"difficulty"`
	OverallScore       int              `json:"overall_score"`
	Percentile         int              `json:"percentile"`
	Duration           string           `json:"duration"`
	StartedAt          time.Time        `json:"started_at"`
	EndedAt            *time.Time       `json:"ended_at"`
	InterviewerSummary string           `json:"interviewer_summary"`
	Rubrics            []RubricCategory `json:"rubrics"`
	Strengths          []string         `json:"strengths"`
	Weaknesses         []string         `json:"weaknesses"`
	Recommendations    []string         `json:"recommendations"`
	DiagramComponents  []string         `json:"diagram_components"`
}

type ReportListItem struct {
	ID                 string    `json:"id"`
	SessionID          string    `json:"session_id"`
	Title              string    `json:"title"`
	Difficulty         string    `json:"difficulty"`
	OverallScore       int       `json:"overall_score"`
	Percentile         int       `json:"percentile"`
	Duration           string    `json:"duration"`
	Date               time.Time `json:"date"`
	InterviewerSummary string    `json:"interviewer_summary"`
}

type ReportsSummary struct {
	TotalReports int `json:"total_reports"`
	AverageScore int `json:"average_score"`
	HighestScore int `json:"highest_score"`
}

type ReportsListResponse struct {
	Summary ReportsSummary   `json:"summary"`
	Reports []ReportListItem `json:"reports"`
}
