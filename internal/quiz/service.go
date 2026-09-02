package quiz

import (
	"context"

	"github.com/MonuChaudhary14/Archon/internal/models"
)

type Service interface {
	GetDailyChallenge(ctx context.Context) (*models.QuizQuestion, error)
	VerifyDailyChallenge(ctx context.Context, req models.VerifyDailyChallengeRequest) (*models.VerifyDailyChallengeResponse, error)
	ListDecks(ctx context.Context, userID int) ([]models.QuizDeckItem, error)
	GetDeckQuestions(ctx context.Context, deckID string) ([]models.QuizQuestion, error)
	SubmitDeckQuiz(ctx context.Context, userID int, deckID string, req models.SubmitDeckQuizRequest) (*models.SubmitDeckQuizResponse, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetDailyChallenge(ctx context.Context) (*models.QuizQuestion, error) {
	return s.repo.GetDailyChallenge(ctx)
}

func (s *service) VerifyDailyChallenge(ctx context.Context, req models.VerifyDailyChallengeRequest) (*models.VerifyDailyChallengeResponse, error) {
	return s.repo.VerifyDailyChallenge(ctx, req.QuestionID, req.SelectedOptionID)
}

func (s *service) ListDecks(ctx context.Context, userID int) ([]models.QuizDeckItem, error) {
	return s.repo.ListDecks(ctx, userID)
}

func (s *service) GetDeckQuestions(ctx context.Context, deckID string) ([]models.QuizQuestion, error) {
	return s.repo.GetDeckQuestions(ctx, deckID)
}

func (s *service) SubmitDeckQuiz(ctx context.Context, userID int, deckID string, req models.SubmitDeckQuizRequest) (*models.SubmitDeckQuizResponse, error) {
	return s.repo.SubmitDeckQuiz(ctx, userID, deckID, req)
}
