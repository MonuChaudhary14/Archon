package interview

import (
	"context"
	"fmt"
)

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) StartInterview(ctx context.Context, userID int, req CreateInterviewRequest) (*Question, string, error) {

	question, err := s.repo.GetRandomUnansweredQuestion(ctx, userID, req.Difficulty)
	if err != nil {
		return nil, "", fmt.Errorf("could not find a suitable question: %w", err)
	}

	interviewID, err := s.repo.CreateInterview(ctx, userID, question.ID)
	if err != nil {
		return nil, "", fmt.Errorf("could not start interview session: %w", err)
	}

	return question, interviewID, nil
}
