package dashboard

import (
	"context"

	"github.com/MonuChaudhary14/Archon/internal/models"
)

type Repository interface {
	GetDashboardData(ctx context.Context, userID int) (*models.DashboardOverviewResponse, error)
}
