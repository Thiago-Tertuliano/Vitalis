package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

// Create grava user + credential + roles na MESMA transação.
func (r *UserRepo) Create(ctx context.Context, u domain.User, passwordHash string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `
		INSERT INTO users (id, email, nome, telefone, status, cpf, data_nascimento, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, u.ID, u.Email, u.Nome, u.Telefone, u.Status, nullIfEmpty(u.CPF), u.DataNascimento, u.CreatedAt, u.UpdateAt)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO credentials (user_id, password_hash, algo, update_at)
		VALUES ($1,$2,'argon2id',$3)
	`, u.ID, passwordHash, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("insert credentials: %w", err)
	}
	for _, role := range u.Roles {
		if _, err := tx.Exec(ctx, `INSERT INTO user_roles (user_id, role) VALUES ($1,$2)`, u.ID, role); err != nil {
			return fmt.Errorf("insert role: %w", err)
		}
	}
	return tx.Commit(ctx)
}
func (r *UserRepo) FindByEmail(ctx context.Context, email string) (domain.User, string, error) {
	var u domain.User
	var hash string
	var cpf *string
	err := r.pool.QueryRow(ctx, `
		SELECT u.id, u.email, u.nome, u.telefone, u.status, u.cpf, u.data_nascimento,
		       u.created_at, u.updated_at, c.password_hash
		FROM users u
		JOIN credentials c ON c.user_id = u.id
		WHERE u.email = $1
	`, email).Scan(
		&u.ID, &u.Email, &u.Nome, &u.Telefone, &u.Status, &cpf, &u.DataNascimento,
		&u.CreatedAt, &u.UpdateAt, &hash,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, "", err
	}
	if err != nil {
		return domain.User{}, "", err
	}
	if cpf != nil {
		u.CPF = *cpf
	}
	roles, err := r.loadRoles(ctx, u.ID)
	if err != nil {
		return domain.User{}, "", err
	}
	u.Roles = roles
	return u, hash, nil
}
func (r *UserRepo) FindByID(ctx context.Context, id string) (domain.User, error) {
	var u domain.User
	var cpf *string
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, nome, telefone, status, cpf, data_nascimento, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(
		&u.ID, &u.Email, &u.Nome, &u.Telefone, &u.Status, &cpf, &u.DataNascimento,
		&u.CreatedAt, &u.UpdateAt,
	)
	if err != nil {
		return domain.User{}, err
	}
	if cpf != nil {
		u.CPF = *cpf
	}
	roles, err := r.loadRoles(ctx, u.ID)
	if err != nil {
		return domain.User{}, err
	}
	u.Roles = roles
	return u, nil
}
func (r *UserRepo) UpdateProfile(ctx context.Context, u domain.User) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users
		SET nome=$2, telefone=$3, cpf=$4, data_nascimento=$5, updated_at=$6
		WHERE id=$1
	`, u.ID, u.Nome, u.Telefone, nullIfEmpty(u.CPF), u.DataNascimento, time.Now().UTC())
	return err
}
func (r *UserRepo) ReplaceRoles(ctx context.Context, userID string, roles []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM user_roles WHERE user_id=$1`, userID); err != nil {
		return err
	}
	for _, role := range roles {
		if _, err := tx.Exec(ctx, `INSERT INTO user_roles (user_id, role) VALUES ($1,$2)`, userID, role); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
func (r *UserRepo) EmailExists(ctx context.Context, email string) (bool, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(1) FROM users WHERE email=$1`, email).Scan(&n)
	return n > 0, err
}
func (r *UserRepo) loadRoles(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT role FROM user_roles WHERE user_id=$1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
