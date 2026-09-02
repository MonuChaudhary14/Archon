package analytics

import (
	"context"

	"github.com/MonuChaudhary14/Archon/internal/models"
)

type Service interface {
	GetAnalytics(ctx context.Context, userID int, timeRange string) (*models.AnalyticsResponse, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetAnalytics(ctx context.Context, userID int, timeRange string) (*models.AnalyticsResponse, error) {
	if timeRange == "" {
		timeRange = "30d"
	}
	return s.repo.GetAnalytics(ctx, userID, timeRange)
}
