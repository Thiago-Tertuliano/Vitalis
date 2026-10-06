-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id               TEXT PRIMARY KEY,
    email            TEXT NOT NULL UNIQUE,
    nome             TEXT NOT NULL,
    telefone         TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL DEFAULT 'active',
    cpf              TEXT,
    data_nascimento  DATE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS credentials (
    user_id           TEXT PRIMARY KEY REFERENCES users(id),
    password_hash     TEXT NOT NULL,
    algo              TEXT NOT NULL DEFAULT 'argon2id',
    update_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id           TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role              TEXT NOT NULL,
    PRIMARY KEY (user_id, role),
    CONSTRAINT chk_role CHECK (role IN ('paciente', 'medico', 'farmacia', 'motoboy', 'admin'))
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    jti               TEXT PRIMARY KEY,
    user_id           TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at        TIMESTAMPTZ NOT  NULL,
    revoked_at        TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS addresses (
    id                TEXT PRIMARY KEY,
    user_id           TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label             TEXT NOT NULL DEFAULT 'casa',
    cep               TEXT NOT NULL,
    logradouro        TEXT NOT NULL,
    numero            TEXT NOT NULL,
    complemento       TEXT NOT NULL DEFAULT '',
    bairro            TEXT NOT NULL,
    cidade            TEXT NOT NULL,
    uf                CHAR(2) NOT NULL,
    is_default        BOOLEAN NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    token             TEXT PRIMARY KEY,
    user_id           TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at        TIMESTAMPTZ NOT NULL,
    used_at           TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS outbox_events (
    id                TEXT PRIMARY KEY,
    type              TEXT NOT NULL,
    producer          TEXT NOT NULL,
    version           INT NOT NULL DEFAULT 1,
    correlation_id    TEXT NOT NULL DEFAULT '',
    payload           JSONB NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at      TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_outbox_unpublished  
    ON outbox_events (created_at) WHERE published_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS addresses;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS credentials;
DROP TABLE IF EXISTS users;
