package outbox

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Publisher lê outbox_events pendentes e publica no Redis Streams.
type Publisher struct {
	pool   *pgxpool.Pool
	rdb    *redis.Client
	stream string
	log    *slog.Logger
}

func NewPublisher(pool *pgxpool.Pool, rdb *redis.Client, stream string, log *slog.Logger) *Publisher {
	return &Publisher{pool: pool, rdb: rdb, stream: stream, log: log}
}

func (p *Publisher) Run(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := p.flush(ctx); err != nil {
				p.log.Error("outbox flush", "err", err)
			}
		}
	}
}

func (p *Publisher) flush(ctx context.Context) error {
	rows, err := p.pool.Query(ctx, `
		SELECT id, type, producer, version, correlation_id, payload
		FROM outbox_events
		WHERE published_at IS NULL
		ORDER BY created_at
		LIMIT 50
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id, typ, producer, correlation string
		var version int
		var payload []byte
		if err := rows.Scan(&id, &typ, &producer, &version, &correlation, &payload); err != nil {
			return err
		}
		envelope, _ := json.Marshal(map[string]any{
			"id":             id,
			"type":           typ,
			"occurred_at":    time.Now().UTC().Format(time.RFC3339),
			"producer":       producer,
			"version":        version,
			"correlation_id": correlation,
			"data":           json.RawMessage(payload),
		})
		if err := p.rdb.XAdd(ctx, &redis.XAddArgs{
			Stream: p.stream,
			Values: map[string]interface{}{"envelope": string(envelope)},
		}).Err(); err != nil {
			return err
		}
		if _, err := p.pool.Exec(ctx, `UPDATE outbox_events SET published_at = $2 WHERE id = $1`, id, time.Now().UTC()); err != nil {
			return err
		}
		p.log.Info("outbox published", "id", id, "type", typ)
	}
	return rows.Err()
}
