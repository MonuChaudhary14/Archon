package quiz

import (
	"context"

	"github.com/MonuChaudhary14/Archon/internal/models"
)

type Repository interface {
	GetDailyChallenge(ctx context.Context) (*models.QuizQuestion, error)
	VerifyDailyChallenge(ctx context.Context, questionID string, selectedOptionID string) (*models.VerifyDailyChallengeResponse, error)
	ListDecks(ctx context.Context, userID int) ([]models.QuizDeckItem, error)
	GetDeckQuestions(ctx context.Context, deckID string) ([]models.QuizQuestion, error)
	SubmitDeckQuiz(ctx context.Context, userID int, deckID string, req models.SubmitDeckQuizRequest) (*models.SubmitDeckQuizResponse, error)
}
