package dashboard

import (
	"context"
	"testing"

	"github.com/MonuChaudhary14/Archon/internal/models"
)

type mockRepo struct{}

func (m *mockRepo) GetDashboardData(ctx context.Context, userID int) (*models.DashboardOverviewResponse, error) {
	return &models.DashboardOverviewResponse{
		Stats: models.DashboardStats{
			ReadinessScore: 78,
		},
	}, nil
}

func TestGetOverview(t *testing.T) {
	svc := NewService(&mockRepo{})
	res, err := svc.GetOverview(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Stats.ReadinessScore != 78 {
		t.Errorf("expected readiness score 78, got %d", res.Stats.ReadinessScore)
	}
}
