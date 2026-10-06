package outbox

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Event segue o envelope ADR-003.
type Event struct {
	ID            string
	Type          string
	Producer      string
	Version       int
	CorrelationID string
	Data          json.RawMessage
}

type Writer struct {
	pool *pgxpool.Pool
}

func NewWriter(pool *pgxpool.Pool) *Writer {
	return &Writer{pool: pool}
}

// Enqueue grava na outbox_events (mesma TX do domínio quando chamado dentro de uma TX).
// Neste template usa pool simples; nos serviços reais prefira pgx.Tx compartilhada.
func (w *Writer) Enqueue(ctx context.Context, e Event) error {
	_, err := w.pool.Exec(ctx, `
		INSERT INTO outbox_events (id, type, producer, version, correlation_id, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, e.ID, e.Type, e.Producer, e.Version, e.CorrelationID, e.Data, time.Now().UTC())
	return err
}
