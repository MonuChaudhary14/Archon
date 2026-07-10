package interview
import (
	"context"
)

type Service interface {
	StartInterview(ctx context.Context, userID int, req CreateInterviewRequest) (*Question, string, error)
}
