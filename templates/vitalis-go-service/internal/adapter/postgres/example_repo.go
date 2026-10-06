package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/Rede-Medica-D-Excelencia-Vitalis/vitalis-service-template/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ExampleRepo struct {
	pool *pgxpool.Pool
}

func NewExampleRepo(pool *pgxpool.Pool) *ExampleRepo {
	return &ExampleRepo{pool: pool}
}

func (r *ExampleRepo) Save(ctx context.Context, e domain.Example) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO examples (id, name, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name
	`, e.ID, e.Name, e.CreatedAt)
	return err
}

func (r *ExampleRepo) FindByID(ctx context.Context, id string) (domain.Example, error) {
	var e domain.Example
	var created time.Time
	err := r.pool.QueryRow(ctx, `SELECT id, name, created_at FROM examples WHERE id = $1`, id).
		Scan(&e.ID, &e.Name, &created)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Example{}, err
	}
	e.CreatedAt = created
	return e, err
}
