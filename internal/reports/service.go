package reports

import (
	"context"

	"github.com/MonuChaudhary14/Archon/internal/models"
)

type Service interface {
	ListReports(ctx context.Context, userID int, search string, difficulty string, page int, limit int) (*models.ReportsListResponse, error)
	GetReportDetail(ctx context.Context, userID int, sessionID string) (*models.DetailedReportResponse, bool, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) ListReports(ctx context.Context, userID int, search string, difficulty string, page int, limit int) (*models.ReportsListResponse, error) {
	return s.repo.ListReports(ctx, userID, search, difficulty, page, limit)
}

func (s *service) GetReportDetail(ctx context.Context, userID int, sessionID string) (*models.DetailedReportResponse, bool, error) {
	return s.repo.GetReportDetail(ctx, userID, sessionID)
}
