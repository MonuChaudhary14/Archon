package auth

import "time"

type User struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
	IsVerified   bool   `json:"is_verified"`
	TokenVersion int    `json:"token_version"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type RefreshToken struct {
	ID        uint
	UserID    uint
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
	IsUsed    bool
}

type UnverifiedUser struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash"`
}

type OAuthConnection struct {
	ID             uint
	UserID         uint
	Provider       string
	ProviderUserID string
	CreatedAt      time.Time
}
