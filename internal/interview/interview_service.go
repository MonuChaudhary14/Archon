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
	var question *Question
	var err error

	if req.QuestionID != nil && *req.QuestionID != "" {
		question, err = s.repo.GetQuestionByID(ctx, *req.QuestionID)
		if err != nil {
			return nil, "", fmt.Errorf("could not find specific question: %w", err)
		}
		if question.Difficulty != req.Difficulty {
			return nil, "", fmt.Errorf("question difficulty mismatch: requested %s, but question is %s", req.Difficulty, question.Difficulty)
		}
	} else {
		question, err = s.repo.GetRandomUnansweredQuestion(ctx, userID, req.Difficulty)
		if err != nil {
			return nil, "", fmt.Errorf("could not find a suitable question: %w", err)
		}
	}

	interviewID, err := s.repo.CreateInterview(ctx, userID, question.ID)
	if err != nil {
		return nil, "", fmt.Errorf("could not start interview session: %w", err)
	}

	return question, interviewID, nil
}

func (s *service) GetQuestions(ctx context.Context) ([]*Question, error) {
	return s.repo.GetQuestions(ctx)
}

func (s *service) GetInterviewByID(ctx context.Context, userID int, interviewID string) (*Interview, error) {
	return s.repo.GetInterviewByID(ctx, userID, interviewID)
}

func (s *service) SubmitInterview(ctx context.Context, userID int, interviewID string) error {
	return s.repo.SubmitInterview(ctx, userID, interviewID)
}

func (s *service) GetInterviewsByUserID(ctx context.Context, userID int) ([]*Interview, error) {
	return s.repo.GetInterviewsByUserID(ctx, userID)
}

