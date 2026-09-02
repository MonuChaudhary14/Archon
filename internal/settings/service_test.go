package settings

import (
	"context"
	"testing"

	"github.com/MonuChaudhary14/Archon/internal/models"
)

type mockRepo struct {
	settings models.UserSettings
}

func (m *mockRepo) GetSettings(ctx context.Context, userID int) (*models.UserSettings, error) {
	return &m.settings, nil
}

func (m *mockRepo) UpdateSettings(ctx context.Context, userID int, s models.UserSettings) error {
	m.settings = s
	return nil
}

func TestSettingsService(t *testing.T) {
	repo := &mockRepo{
		settings: models.UserSettings{TargetLevel: "senior"},
	}
	svc := NewService(repo)

	s, err := svc.GetSettings(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.TargetLevel != "senior" {
		t.Errorf("expected level senior, got %s", s.TargetLevel)
	}

	s.TargetLevel = "staff"
	err = svc.UpdateSettings(context.Background(), 1, *s)
	if err != nil {
		t.Fatalf("unexpected error updating settings: %v", err)
	}

	updated, _ := svc.GetSettings(context.Background(), 1)
	if updated.TargetLevel != "staff" {
		t.Errorf("expected level staff, got %s", updated.TargetLevel)
	}
}
