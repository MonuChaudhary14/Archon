package quiz

import (
	"context"
	"testing"

	"github.com/MonuChaudhary14/Archon/internal/models"
)

type mockQuizRepo struct {
	dailyQuestion *models.QuizQuestion
	verifyResp    *models.VerifyDailyChallengeResponse
	decks         []models.QuizDeckItem
}

func (m *mockQuizRepo) GetDailyChallenge(ctx context.Context) (*models.QuizQuestion, error) {
	return m.dailyQuestion, nil
}

func (m *mockQuizRepo) VerifyDailyChallenge(ctx context.Context, userID int, questionID string, selectedOptionID string) (*models.VerifyDailyChallengeResponse, error) {
	return m.verifyResp, nil
}

func (m *mockQuizRepo) ListDecks(ctx context.Context, userID int) ([]models.QuizDeckItem, error) {
	return m.decks, nil
}

func (m *mockQuizRepo) GetDeckQuestions(ctx context.Context, deckID string) ([]models.QuizQuestion, error) {
	return []models.QuizQuestion{*m.dailyQuestion}, nil
}

func (m *mockQuizRepo) SubmitDeckQuiz(ctx context.Context, userID int, deckID string, req models.SubmitDeckQuizRequest) (*models.SubmitDeckQuizResponse, error) {
	return &models.SubmitDeckQuizResponse{
		DeckID:         deckID,
		TotalQuestions: 1,
		CorrectCount:   1,
		ScorePercent:   100,
	}, nil
}

func TestQuizService_DailyChallenge(t *testing.T) {
	mockQuestion := &models.QuizQuestion{
		ID:       "q-daily-1",
		Question: "How does Raft handle network partitions?",
		TopicTag: "Distributed Consensus",
		Options: []models.QuizOption{
			{ID: "opt-1", Text: "Strict majority quorum"},
			{ID: "opt-2", Text: "Central coordinator"},
		},
	}
	mockVerify := &models.VerifyDailyChallengeResponse{
		IsCorrect:       true,
		CorrectOptionID: "opt-1",
		Explanation:     "Raft requires (N/2 + 1) majority votes to elect a leader.",
	}

	repo := &mockQuizRepo{
		dailyQuestion: mockQuestion,
		verifyResp:    mockVerify,
	}

	svc := NewService(repo)

	q, err := svc.GetDailyChallenge(context.Background())
	if err != nil {
		t.Fatalf("unexpected error fetching daily challenge: %v", err)
	}
	if q.ID != "q-daily-1" {
		t.Errorf("expected ID 'q-daily-1', got %s", q.ID)
	}
	if len(q.Options) != 2 {
		t.Errorf("expected 2 options, got %d", len(q.Options))
	}

	verifyReq := models.VerifyDailyChallengeRequest{
		QuestionID:       "q-daily-1",
		SelectedOptionID: "opt-1",
	}
	resp, err := svc.VerifyDailyChallenge(context.Background(), 10, verifyReq)
	if err != nil {
		t.Fatalf("unexpected error verifying daily challenge: %v", err)
	}
	if !resp.IsCorrect {
		t.Errorf("expected is_correct to be true")
	}
	if resp.CorrectOptionID != "opt-1" {
		t.Errorf("expected correct_option_id 'opt-1', got %s", resp.CorrectOptionID)
	}
}
