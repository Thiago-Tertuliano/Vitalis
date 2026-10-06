package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshRepo struct {
	pool *pgxpool.Pool
}

func NewRefreshRepo(pool *pgxpool.Pool) *RefreshRepo {
	return &RefreshRepo{pool: pool}
}

func (r *RefreshRepo) Save(ctx context.Context, jti, userID string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (jti, user_id, expires_at, created_at)
		VALUES ($1,$2,$3,$4)
	`, jti, userID, expiresAt, time.Now().UTC())
	return err
}

func (r *RefreshRepo) FindValid(ctx context.Context, jti string) (string, error) {
	var userID string
	err := r.pool.QueryRow(ctx, `
		SELECT user_id FROM refresh_tokens
		WHERE jti=$1 AND revoked_at IS NULL AND expires_at > NOW()
	`, jti).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	return userID, err
}

func (r *RefreshRepo) Revoke(ctx context.Context, jti string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at=$2 WHERE jti=$1 AND revoked_at IS NULL
	`, jti, time.Now().UTC())
	return err
}