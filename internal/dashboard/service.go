package dashboard

import (
	"context"

	"github.com/MonuChaudhary14/Archon/internal/models"
)

type Service interface {
	GetOverview(ctx context.Context, userID int) (*models.DashboardOverviewResponse, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetOverview(ctx context.Context, userID int) (*models.DashboardOverviewResponse, error) {
	return s.repo.GetDashboardData(ctx, userID)
}
