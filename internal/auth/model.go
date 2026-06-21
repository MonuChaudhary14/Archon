package auth

import "time"

type User struct {
	ID           uint
	Name         string
	Email        string
	PasswordHash string
	IsVerified   bool
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
