package reports

import (
	"context"

	"github.com/MonuChaudhary14/Archon/internal/models"
)

type Repository interface {
	ListReports(ctx context.Context, userID int, search string, difficulty string, page int, limit int) (*models.ReportsListResponse, error)
	GetReportDetail(ctx context.Context, userID int, sessionID string) (*models.DetailedReportResponse, bool, error)
}
