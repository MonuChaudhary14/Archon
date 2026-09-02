package settings

import (
	"context"

	"github.com/MonuChaudhary14/Archon/internal/models"
)

type Repository interface {
	GetSettings(ctx context.Context, userID int) (*models.UserSettings, error)
	UpdateSettings(ctx context.Context, userID int, settings models.UserSettings) error
}
