package quiz

import (
	"context"
	"encoding/json"
	"time"

	"github.com/MonuChaudhary14/Archon/internal/models"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

type cachedRepository struct {
	next  Repository
	redis *redis.Client
	sfg   singleflight.Group
}

func NewCachedRepository(next Repository, redisClient *redis.Client) Repository {
	return &cachedRepository{
		next:  next,
		redis: redisClient,
	}
}

func (r *cachedRepository) GetDailyChallenge(ctx context.Context) (*models.QuizQuestion, error) {
	today := time.Now().UTC().Format("2006-01-02")
	cacheKey := "daily_challenge:" + today

	if r.redis != nil {
		val, err := r.redis.Get(ctx, cacheKey).Result()
		if err == nil && val != "" {
			var q models.QuizQuestion
			if jsonErr := json.Unmarshal([]byte(val), &q); jsonErr == nil {
				return &q, nil
			}
		}
	}

	v, err, _ := r.sfg.Do(cacheKey, func() (interface{}, error) {
		if r.redis != nil {
			val, err := r.redis.Get(ctx, cacheKey).Result()
			if err == nil && val != "" {
				var q models.QuizQuestion
				if jsonErr := json.Unmarshal([]byte(val), &q); jsonErr == nil {
					return &q, nil
				}
			}
		}

		q, dbErr := r.next.GetDailyChallenge(ctx)
		if dbErr != nil {
			return nil, dbErr
		}

		if r.redis != nil && q != nil {
			bytes, mErr := json.Marshal(q)
			if mErr == nil {
				now := time.Now().UTC()
				midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
				ttl := midnight.Sub(now)
				if ttl < 10*time.Second {
					ttl = 10 * time.Second
				}
				_ = r.redis.Set(ctx, cacheKey, bytes, ttl).Err()
			}
		}

		return q, nil
	})

	if err != nil {
		return nil, err
	}

	return v.(*models.QuizQuestion), nil
}

func (r *cachedRepository) VerifyDailyChallenge(ctx context.Context, userID int, questionID string, selectedOptionID string) (*models.VerifyDailyChallengeResponse, error) {
	return r.next.VerifyDailyChallenge(ctx, userID, questionID, selectedOptionID)
}

func (r *cachedRepository) ListDecks(ctx context.Context, userID int) ([]models.QuizDeckItem, error) {
	return r.next.ListDecks(ctx, userID)
}

func (r *cachedRepository) GetDeckQuestions(ctx context.Context, deckID string) ([]models.QuizQuestion, error) {
	return r.next.GetDeckQuestions(ctx, deckID)
}

func (r *cachedRepository) SubmitDeckQuiz(ctx context.Context, userID int, deckID string, req models.SubmitDeckQuizRequest) (*models.SubmitDeckQuizResponse, error) {
	return r.next.SubmitDeckQuiz(ctx, userID, deckID, req)
}
