package auth

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DBTX interface {
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
	QueryRow(context.Context, string, ...interface{}) pgx.Row
}

type postgresUserRepository struct {
	pool *pgxpool.Pool
	db   DBTX
}

func NewPostgresUserRepository(pool *pgxpool.Pool) UserRepository {
	return &postgresUserRepository{
		pool: pool,
		db:   pool,
	}
}

func (r *postgresUserRepository) RunInTx(ctx context.Context, fn func(txRepo UserRepository) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txRepo := &postgresUserRepository{
		pool: r.pool,
		db:   tx,
	}

	if err := fn(txRepo); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *postgresUserRepository) CreateUser(ctx context.Context, user *User) error {

	query := `
		INSERT INTO users (
			name,
			email,
			password_hash,
			is_verified,
			token_version
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			1
		)
		RETURNING id, token_version
	`

	return r.db.QueryRow(
		ctx,
		query,
		user.Name,
		user.Email,
		user.PasswordHash,
		user.IsVerified,
	).Scan(
		&user.ID,
		&user.TokenVersion,
	)
}

func (r *postgresUserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {

	query := `
		SELECT
			id,
			name,
			email,
			password_hash,
			is_verified,
			token_version,
			created_at,
			updated_at
		FROM users
		WHERE email = $1
	`

	var user User

	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.IsVerified,
		&user.TokenVersion,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, ErrUserNotFound
	}

	return &user, nil
}

func (r *postgresUserRepository) FindByID(ctx context.Context, id uint) (*User, error) {

	query := `
		SELECT
			id,
			name,
			email,
			password_hash,
			is_verified,
			token_version,
			created_at,
			updated_at
		FROM users
		WHERE id = $1
	`

	var user User

	err := r.db.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.IsVerified,
		&user.TokenVersion,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, ErrUserNotFound
	}

	return &user, nil
}

func (r *postgresUserRepository) UpdateVerificationStatus(ctx context.Context, email string) error {

	query := `

		UPDATE users
		SET is_verified = TRUE
		WHERE email = $1
	`
	_, err := r.db.Exec(
		ctx,
		query,
		email,
	)

	return err
}

func (r *postgresUserRepository) SaveRefreshToken(ctx context.Context, token *RefreshToken) error {

	query := `
		INSERT INTO refresh_tokens (
			user_id,
			token_hash,
			expires_at
		)
		VALUES (
			$1,
			$2,
			$3
		)
		RETURNING id
	`

	return r.db.QueryRow(
		ctx,
		query,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
	).Scan(
		&token.ID,
	)
}

func (r *postgresUserRepository) FindRefreshTokenByHash(ctx context.Context, hash string) (*RefreshToken, error) {
	query := `
		SELECT
			id,
			user_id,
			token_hash,
			expires_at,
			created_at,
			is_used
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	var token RefreshToken
	err := r.db.QueryRow(ctx, query, hash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.IsUsed,
	)

	if err != nil {
		return nil, ErrInvalidToken
	}

	return &token, nil
}

func (r *postgresUserRepository) DeleteRefreshToken(ctx context.Context, id uint) error {
	query := `DELETE FROM refresh_tokens WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *postgresUserRepository) MarkRefreshTokenAsUsed(ctx context.Context, id uint) error {
	query := `UPDATE refresh_tokens SET is_used = TRUE WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *postgresUserRepository) RevokeAllUserTokens(ctx context.Context, userID uint) error {
	query := `DELETE FROM refresh_tokens WHERE user_id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}

func (r *postgresUserRepository) UpdatePassword(ctx context.Context, email string, hashedPassword string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE email = $2`
	_, err := r.db.Exec(ctx, query, hashedPassword, email)
	return err
}

func (r *postgresUserRepository) IncrementTokenVersion(ctx context.Context, userID uint) error {
	query := `UPDATE users SET token_version = token_version + 1, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}

func (r *postgresUserRepository) SaveOAuthConnection(ctx context.Context, connection *OAuthConnection) error {
	query := `
		INSERT INTO oauth_connections (
			user_id,
			provider,
			provider_user_id
		)
		VALUES (
			$1,
			$2,
			$3
		)
		RETURNING id, created_at
	`
	return r.db.QueryRow(
		ctx,
		query,
		connection.UserID,
		connection.Provider,
		connection.ProviderUserID,
	).Scan(
		&connection.ID,
		&connection.CreatedAt,
	)
}

func (r *postgresUserRepository) FindOAuthConnection(ctx context.Context, provider, providerUserID string) (*OAuthConnection, error) {
	query := `
		SELECT
			id,
			user_id,
			provider,
			provider_user_id,
			created_at
		FROM oauth_connections
		WHERE provider = $1 AND provider_user_id = $2
	`
	var conn OAuthConnection
	err := r.db.QueryRow(ctx, query, provider, providerUserID).Scan(
		&conn.ID,
		&conn.UserID,
		&conn.Provider,
		&conn.ProviderUserID,
		&conn.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &conn, nil
}
