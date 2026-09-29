-- Players table
CREATE TABLE IF NOT EXISTS players (
    id BIGSERIAL PRIMARY KEY,
    external_id BIGINT NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    club_name VARCHAR(255) NOT NULL,
    league_name VARCHAR(100) NOT NULL,
    league_code VARCHAR(10) NOT NULL,
    date_of_birth VARCHAR(20),
    nationality VARCHAR(100),
    position VARCHAR(50) NOT NULL,
    shirt_number INTEGER,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Search and query optimization indexes (Constitution Principle XI)
CREATE INDEX IF NOT EXISTS idx_players_active ON players (active);
CREATE INDEX IF NOT EXISTS idx_players_league_code ON players (league_code);
CREATE INDEX IF NOT EXISTS idx_players_club_name ON players (club_name);
CREATE INDEX IF NOT EXISTS idx_players_position ON players (position);
CREATE INDEX IF NOT EXISTS idx_players_name ON players (name);

-- Immutable audit log table (Constitution Principle X)
CREATE TABLE IF NOT EXISTS player_audit_logs (
    id BIGSERIAL PRIMARY KEY,
    entity_type VARCHAR(50) NOT NULL,
    entity_id BIGINT NOT NULL,
    action VARCHAR(20) NOT NULL,
    actor VARCHAR(100) NOT NULL,
    diff_old JSONB,
    diff_new JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_entity ON player_audit_logs (entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_created_at ON player_audit_logs (created_at);

CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    email         VARCHAR(254) NOT NULL UNIQUE,
    password_hash VARCHAR(60)  NOT NULL,
    privilege     SMALLINT     NOT NULL,
    active        BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         UUID        PRIMARY KEY,
    family_id  UUID        NOT NULL,
    user_id    BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    issued_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id
    ON refresh_tokens (user_id);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_live
    ON refresh_tokens (user_id)
    WHERE used_at IS NULL AND revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_family
    ON refresh_tokens (family_id)
    WHERE revoked_at IS NULL;
