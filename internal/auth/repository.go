package auth

import "context"

type UserRepository interface {
	CreateUser(ctx context.Context, user *User) error

	FindByEmail(ctx context.Context, email string) (*User, error)

	FindByID(ctx context.Context, id uint) (*User, error)

	UpdateVerificationStatus(ctx context.Context, email string) error

	SaveRefreshToken(ctx context.Context, token *RefreshToken) error

	FindRefreshTokenByHash(ctx context.Context, hash string) (*RefreshToken, error)

	DeleteRefreshToken(ctx context.Context, id uint) error

	MarkRefreshTokenAsUsed(ctx context.Context, id uint) error

	RevokeAllUserTokens(ctx context.Context, userID uint) error

	UpdatePassword(ctx context.Context, email string, hashedPassword string) error

	IncrementTokenVersion(ctx context.Context, userID uint) error

	RunInTx(ctx context.Context, fn func(txRepo UserRepository) error) error
}
