package settings

import (
	"context"

	"github.com/MonuChaudhary14/Archon/internal/models"
)

type Service interface {
	GetSettings(ctx context.Context, userID int) (*models.UserSettings, error)
	UpdateSettings(ctx context.Context, userID int, settings models.UserSettings) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetSettings(ctx context.Context, userID int) (*models.UserSettings, error) {
	return s.repo.GetSettings(ctx, userID)
}

func (s *service) UpdateSettings(ctx context.Context, userID int, settings models.UserSettings) error {
	return s.repo.UpdateSettings(ctx, userID, settings)
}
