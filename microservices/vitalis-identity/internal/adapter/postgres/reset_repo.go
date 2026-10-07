package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ResetRepo struct {
	pool *pgxpool.Pool
}

func NewResetRepo(pool *pgxpool.Pool) *ResetRepo {
	return &ResetRepo{pool: pool}
}

func (r *ResetRepo) Save(ctx context.Context, token, userID string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO password_reset_tokens (token, user_id, expires_at, created_at)
		VALUES ($1,$2,$3,$4)
	`, token, userID, expiresAt, time.Now().UTC())
	return err
}

func (r *ResetRepo) Consume(ctx context.Context, token string) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID string
	err = tx.QueryRow(ctx, `
		SELECT user_id FROM password_reset_tokens
		WHERE token=$1 AND used_at IS NULL AND expires_at > NOW()
	`, token).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE password_reset_tokens SET used_at=$2 WHERE token=$1`, token, time.Now().UTC()); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return userID, nil
}

func (r *ResetRepo) UpdatePassword(ctx context.Context, userID, hash string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE credentials SET password_hash=$2, update_at=$3 WHERE user_id=$1
	`, userID, hash, time.Now().UTC())
	return err
}
