package interview

import (
	"context"
)

type Service interface {
	StartInterview(ctx context.Context, userID int, req CreateInterviewRequest) (*Question, string, error)
	GetQuestions(ctx context.Context) ([]*Question, error)
	GetInterviewByID(ctx context.Context, userID int, interviewID string) (*Interview, error)
	GetInterviewsByUserID(ctx context.Context, userID int) ([]*Interview, error)
	SubmitInterview(ctx context.Context, userID int, interviewID string) error
}
