package models

import "time"

type DashboardStats struct {
	ReadinessScore      int     `json:"readiness_score"`
	ReadinessChange     float64 `json:"readiness_change"`
	CompletedInterviews int     `json:"completed_interviews"`
	TotalQuizzesTaken   int     `json:"total_quizzes_taken"`
	QuizAccuracy        int     `json:"quiz_accuracy"`
	StreakDays          int     `json:"streak_days"`
	ScoreHistory        []int   `json:"score_history"`
}

type RecommendedScenario struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Desc       string   `json:"desc"`
	Difficulty string   `json:"difficulty"`
	EstTime    string   `json:"est_time"`
	Topics     []string `json:"topics"`
}

type CompetencyMetric struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}

type RecentInterviewSummary struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"session_id"`
	Title      string    `json:"title"`
	Difficulty string    `json:"difficulty"`
	Score      *int      `json:"score"`
	Status     string    `json:"status"`
	Date       time.Time `json:"date"`
	Duration   string    `json:"duration"`
}

type RoadmapTopic struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Category         string `json:"category"`
	CompletedLessons int    `json:"completed_lessons"`
	TotalLessons     int    `json:"total_lessons"`
	Mastery          int    `json:"mastery"`
	Status           string `json:"status"`
}

type DashboardOverviewResponse struct {
	Stats               DashboardStats           `json:"stats"`
	RecommendedScenario RecommendedScenario      `json:"recommended_scenario"`
	Competencies        []CompetencyMetric       `json:"competencies"`
	RecentInterviews    []RecentInterviewSummary `json:"recent_interviews"`
	Roadmap             []RoadmapTopic           `json:"roadmap"`
}
