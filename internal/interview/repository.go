package interview

import (
	"context"
)

type Repository interface {
	GetRandomUnansweredQuestion(ctx context.Context, userID int, difficulty string) (*Question, error)
	GetQuestions(ctx context.Context) ([]*Question, error)
	GetQuestionByID(ctx context.Context, id string) (*Question, error)
	CreateInterview(ctx context.Context, userID int, questionID string) (string, error)
	GetInterviewByID(ctx context.Context, userID int, interviewID string) (*Interview, error)
	GetInterviewsByUserID(ctx context.Context, userID int) ([]*Interview, error)
	SubmitInterview(ctx context.Context, userID int, interviewID string) error
}
