-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE user_role AS ENUM ('admin', 'ops');
CREATE TYPE tx_jenis AS ENUM ('in', 'out');
CREATE TYPE tx_source AS ENUM ('web', 'wa', 'tele');

CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    initial_balance BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID REFERENCES teams(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role user_role NOT NULL DEFAULT 'ops',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    hari TEXT NOT NULL,
    tanggal DATE NOT NULL,
    jenis tx_jenis NOT NULL,
    deskripsi TEXT NOT NULL,
    total BIGINT NOT NULL CHECK (total > 0),
    nota_key TEXT,
    keterangan TEXT,
    source tx_source NOT NULL DEFAULT 'web',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transactions_team_id ON transactions(team_id);
CREATE INDEX idx_transactions_tanggal ON transactions(tanggal);
CREATE INDEX idx_transactions_team_tanggal ON transactions(team_id, tanggal DESC);

CREATE TABLE integrations (
    team_id UUID PRIMARY KEY REFERENCES teams(id) ON DELETE CASCADE,
    wa_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    wa_session_data TEXT,
    wa_status TEXT NOT NULL DEFAULT 'disconnected',
    tele_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    tele_bot_token TEXT,
    tele_allowed_chat_id BIGINT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE report_tokens (
    team_id UUID PRIMARY KEY REFERENCES teams(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS report_tokens;
DROP TABLE IF EXISTS integrations;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS teams;
DROP TYPE IF EXISTS tx_source;
DROP TYPE IF EXISTS tx_jenis;
DROP TYPE IF EXISTS user_role;
