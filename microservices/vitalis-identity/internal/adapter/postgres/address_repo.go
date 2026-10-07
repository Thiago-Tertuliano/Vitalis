package postgres

import (
	"context"
	"time"

	"github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AddressRepo struct {
	pool *pgxpool.Pool
}

func NewAddressRepo(pool *pgxpool.Pool) *AddressRepo {
	return &AddressRepo{pool: pool}
}

func (r *AddressRepo) ListByUser(ctx context.Context, userID string) ([]domain.Address, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, label, cep, logradouro, numero, complemento, bairro, cidade, uf, is_default
		FROM addresses WHERE user_id=$1 ORDER BY created_at
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Address
	for rows.Next() {
		var a domain.Address
		if err := rows.Scan(&a.ID, &a.UserID, &a.Label, &a.CEP, &a.Logradouro, &a.Numero,
			&a.Complemento, &a.Bairro, &a.Cidade, &a.UF, &a.IsDefault); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *AddressRepo) Create(ctx context.Context, a domain.Address) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO addresses
		(id, user_id, label, cep, logradouro, numero, complemento, bairro, cidade, uf, is_default, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`, a.ID, a.UserID, a.Label, a.CEP, a.Logradouro, a.Numero, a.Complemento,
		a.Bairro, a.Cidade, a.UF, a.IsDefault, time.Now().UTC())
	return err
}

func (r *AddressRepo) Update(ctx context.Context, a domain.Address) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE addresses SET
			label=$3, cep=$4, logradouro=$5, numero=$6, complemento=$7,
			bairro=$8, cidade=$9, uf=$10, is_default=$11
		WHERE id=$1 AND user_id=$2
	`, a.ID, a.UserID, a.Label, a.CEP, a.Logradouro, a.Numero, a.Complemento,
		a.Bairro, a.Cidade, a.UF, a.IsDefault)
	return err
}

func (r *AddressRepo) Delete(ctx context.Context, userID, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM addresses WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}
