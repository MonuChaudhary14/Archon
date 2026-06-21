package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresPool(databaseURL string) (*pgxpool.Pool, error) {

	return pgxpool.New(
		context.Background(),
		databaseURL,
	)
}
