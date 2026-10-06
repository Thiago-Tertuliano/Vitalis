-- +goose Up
CREATE TABLE IF NOT EXISTS examples (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS outbox_events (
    id              TEXT PRIMARY KEY,
    type            TEXT NOT NULL,
    producer        TEXT NOT NULL,
    version         INT  NOT NULL DEFAULT 1,
    correlation_id  TEXT NOT NULL DEFAULT '',
    payload         JSONB NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at    TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_outbox_unpublished
    ON outbox_events (created_at)
    WHERE published_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS examples;
