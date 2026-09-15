package quiz

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MonuChaudhary14/Archon/internal/models"
)

type countingMockRepo struct {
	fetchCount int64
	question   *models.QuizQuestion
}

func (m *countingMockRepo) GetDailyChallenge(ctx context.Context) (*models.QuizQuestion, error) {
	atomic.AddInt64(&m.fetchCount, 1)
	time.Sleep(50 * time.Millisecond)
	return m.question, nil
}

func (m *countingMockRepo) VerifyDailyChallenge(ctx context.Context, userID int, questionID string, selectedOptionID string) (*models.VerifyDailyChallengeResponse, error) {
	return nil, nil
}

func (m *countingMockRepo) ListDecks(ctx context.Context, userID int) ([]models.QuizDeckItem, error) {
	return nil, nil
}

func (m *countingMockRepo) GetDeckQuestions(ctx context.Context, deckID string) ([]models.QuizQuestion, error) {
	return nil, nil
}

func (m *countingMockRepo) SubmitDeckQuiz(ctx context.Context, userID int, deckID string, req models.SubmitDeckQuizRequest) (*models.SubmitDeckQuizResponse, error) {
	return nil, nil
}

func TestCachedRepository_SingleflightDeduplication(t *testing.T) {
	mockRepo := &countingMockRepo{
		question: &models.QuizQuestion{
			ID:       "q-sf-1",
			Question: "Test singleflight daily challenge",
			TopicTag: "Concurrency",
			Options:  []models.QuizOption{{ID: "o1", Text: "Option 1"}},
		},
	}

	cachedRepo := NewCachedRepository(mockRepo, nil)

	concurrentRequests := 30
	var wg sync.WaitGroup
	wg.Add(concurrentRequests)

	for i := 0; i < concurrentRequests; i++ {
		go func() {
			defer wg.Done()
			q, err := cachedRepo.GetDailyChallenge(context.Background())
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if q.ID != "q-sf-1" {
				t.Errorf("expected ID 'q-sf-1', got %s", q.ID)
			}
		}()
	}

	wg.Wait()

	calls := atomic.LoadInt64(&mockRepo.fetchCount)
	if calls != 1 {
		t.Errorf("expected exactly 1 DB fetch across %d concurrent requests, got %d", concurrentRequests, calls)
	}
}
