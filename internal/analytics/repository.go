package analytics

import (
	"context"

	"github.com/MonuChaudhary14/Archon/internal/models"
)

type Repository interface {
	GetAnalytics(ctx context.Context, userID int, timeRange string) (*models.AnalyticsResponse, error)
}
