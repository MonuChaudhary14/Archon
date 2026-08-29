package interview

import (
	"encoding/json"
	"time"
)

type Question struct {
	ID               string     `json:"id"`
	Title            string     `json:"title"`
	Difficulty       string     `json:"difficulty"`
	ExpectedTopics   []string   `json:"expected_topics"`
	TimeLimitMinutes int        `json:"time_limit_minutes"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty"`
}
type Interview struct {
	ID         string           `json:"id"`
	UserID     int              `json:"user_id"`
	QuestionID string           `json:"question_id"`
	Score      *int             `json:"score,omitempty"`
	Feedback   *json.RawMessage `json:"feedback,omitempty"`
	StartedAt  time.Time        `json:"started_at"`
	EndedAt    *time.Time       `json:"ended_at,omitempty"`
	DeletedAt  *time.Time       `json:"deleted_at,omitempty"`
}
type CreateInterviewRequest struct {
	Difficulty string  `json:"difficulty" binding:"required" enums:"Beginner,Intermediate,Senior,Staff" example:"Senior"`
	QuestionID *string `json:"question_id,omitempty" example:"8d2f6c90-9c12-4266-a456-426614174000"`
}

type StartInterviewResponse struct {
	SessionID string    `json:"session_id"`
	Question  *Question `json:"question"`
}
