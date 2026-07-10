package interview

import (
	"context"
)

type Repository interface{
	GetRandomUnansweredQuestion(ctx context.Context, userID int, difficulty string) (*Question, error)
	CreateInterview(ctx context.Context, userID int, questionID string)(string, error)
}
