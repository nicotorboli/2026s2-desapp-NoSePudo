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

CREATE TABLE IF NOT EXISTS player_statistics (
    id BIGSERIAL PRIMARY KEY,
    player_id BIGINT NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    goals INTEGER NOT NULL DEFAULT 0,
    assists INTEGER NOT NULL DEFAULT 0,
    shots NUMERIC(6,2) NOT NULL DEFAULT 0,
    key_passes NUMERIC(6,2) NOT NULL DEFAULT 0,
    dribbles NUMERIC(6,2) NOT NULL DEFAULT 0,
    tackles NUMERIC(6,2) NOT NULL DEFAULT 0,
    interceptions NUMERIC(6,2) NOT NULL DEFAULT 0,
    fouls NUMERIC(6,2) NOT NULL DEFAULT 0,
    clears NUMERIC(6,2) NOT NULL DEFAULT 0,
    blocks NUMERIC(6,2) NOT NULL DEFAULT 0,
    was_dribbled NUMERIC(6,2) NOT NULL DEFAULT 0,
    rating NUMERIC(4,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_player_statistics_trace
    ON player_statistics (player_id, created_at DESC);

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

-- Scoring: definición de estrategias (fuente de verdad de pesos)
CREATE TABLE IF NOT EXISTS scoring_strategies (
    id         INTEGER PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    alpha      NUMERIC(4,3) NOT NULL,
    weights    JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO scoring_strategies (id, name, alpha, weights) VALUES
(1, 'rateada', 0.5, '{"Goalkeeper":{"clears":0.45,"blocks":0.25,"was_dribbled":0.25,"fouls":0.05},"Defender":{"tackles":0.20,"interceptions":0.20,"clears":0.20,"blocks":0.10,"was_dribbled":0.10,"key_passes":0.10,"dribbles":0.05,"fouls":0.05},"Midfielder":{"key_passes":0.25,"dribbles":0.15,"tackles":0.15,"interceptions":0.10,"assists":0.15,"goals":0.10,"shots":0.05,"fouls":0.05},"Attacker":{"goals":0.35,"assists":0.25,"shots":0.15,"dribbles":0.10,"key_passes":0.10,"fouls":0.05}}'::jsonb),
(2, 'estadistica', 0.2, '{"Goalkeeper":{"clears":0.45,"blocks":0.25,"was_dribbled":0.25,"fouls":0.05},"Defender":{"tackles":0.30,"interceptions":0.20,"clears":0.25,"blocks":0.10,"was_dribbled":0.10,"fouls":0.05},"Midfielder":{"key_passes":0.30,"dribbles":0.20,"tackles":0.15,"interceptions":0.10,"assists":0.10,"goals":0.05,"shots":0.05,"fouls":0.05},"Attacker":{"goals":0.45,"assists":0.30,"shots":0.10,"dribbles":0.05,"key_passes":0.05,"fouls":0.05}}'::jsonb)
ON CONFLICT (id) DO NOTHING;

-- Scoring: timeline de estrategia activa (append-only)
CREATE TABLE IF NOT EXISTS scoring_strategy_activations (
    id            BIGSERIAL PRIMARY KEY,
    strategy_id   INTEGER NOT NULL,
    activated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    activated_by  BIGINT NOT NULL REFERENCES users(id)
);

-- Scoring: histograma compacto de la población por (posición, liga)
CREATE TABLE IF NOT EXISTS scoring_histograms (
    id          BIGSERIAL PRIMARY KEY,
    position    VARCHAR(50) NOT NULL,
    league_code VARCHAR(10) NOT NULL,
    buckets     BIGINT[] NOT NULL,
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_histograms_scope
    ON scoring_histograms (position, league_code, computed_at DESC);

-- Scoring: score materializado por jugador (append-only, idempotente)
CREATE TABLE IF NOT EXISTS player_scores (
    id            BIGSERIAL PRIMARY KEY,
    player_id     BIGINT NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    statistics_id BIGINT NOT NULL REFERENCES player_statistics(id) ON DELETE CASCADE,
    strategy_id   INTEGER NOT NULL,
    weights       JSONB NOT NULL,
    raw_score     NUMERIC(10,4) NOT NULL,
    percentile    NUMERIC(8,6) NOT NULL,
    histogram_id  BIGINT NOT NULL REFERENCES scoring_histograms(id),
    computed_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (player_id, strategy_id, statistics_id)
);

CREATE INDEX IF NOT EXISTS idx_player_scores_player
    ON player_scores (player_id, computed_at DESC);

-- Scoring: coeficiente por liga (append-only, aplicado on-read)
CREATE TABLE IF NOT EXISTS league_coefficients (
    id             BIGSERIAL PRIMARY KEY,
    league_code    VARCHAR(10) NOT NULL,
    coefficient    NUMERIC(6,4) NOT NULL,
    effective_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    set_by         BIGINT NOT NULL REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_league_coefficients_code
    ON league_coefficients (league_code, effective_from DESC);

-- Sincronizaciones programadas: configuración de frecuencia por tipo.
CREATE TABLE IF NOT EXISTS sync_schedules (
    type             VARCHAR(20) PRIMARY KEY,
    enabled          BOOLEAN NOT NULL DEFAULT FALSE,
    interval_minutes INTEGER NOT NULL CHECK (interval_minutes >= 15),
    next_run_at      TIMESTAMPTZ NOT NULL,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by       BIGINT REFERENCES users(id)
);

-- Sincronizaciones programadas: ejecuciones (append-only). Manual y cron caen
-- acá por igual; el progreso intermedio no se persiste, vive solo en memoria.
CREATE TABLE IF NOT EXISTS sync_runs (
    id                BIGSERIAL PRIMARY KEY,
    type              VARCHAR(20) NOT NULL,
    trigger           VARCHAR(10) NOT NULL CHECK (trigger IN ('cron', 'manual')),
    status            VARCHAR(12) NOT NULL
                      CHECK (status IN ('running', 'succeeded', 'partial', 'failed', 'skipped')),
    started_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at       TIMESTAMPTZ,
    total_processed   INTEGER NOT NULL DEFAULT 0,
    total_updated     INTEGER NOT NULL DEFAULT 0,
    total_deactivated INTEGER NOT NULL DEFAULT 0,
    total_matched     INTEGER NOT NULL DEFAULT 0,
    total_unmatched   INTEGER NOT NULL DEFAULT 0,
    total_inserted    INTEGER NOT NULL DEFAULT 0,
    message           TEXT NOT NULL DEFAULT '',
    error             TEXT NOT NULL DEFAULT '',
    triggered_by      BIGINT REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_sync_runs_type_started
    ON sync_runs (type, started_at DESC);

-- Un solo run en curso por tipo. Es la barrera de la DB contra disparos
-- concurrentes; el service además devuelve skipped cuando corresponde.
CREATE UNIQUE INDEX IF NOT EXISTS idx_sync_runs_one_running
    ON sync_runs (type) WHERE status = 'running';

CREATE TABLE IF NOT EXISTS user_accounts (
    user_id           BIGINT PRIMARY KEY REFERENCES users (id),
    available_balance BIGINT      NOT NULL DEFAULT 0,
    reserved_balance  BIGINT      NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_accounts_available_non_negative CHECK (available_balance >= 0),
    CONSTRAINT chk_accounts_reserved_non_negative CHECK (reserved_balance >= 0)
);

CREATE TABLE IF NOT EXISTS user_positions (
    id                 BIGSERIAL PRIMARY KEY,
    user_id            BIGINT      NOT NULL REFERENCES users (id),
    player_id          BIGINT      NOT NULL REFERENCES players (id),
    available_quantity BIGINT      NOT NULL DEFAULT 0,
    reserved_quantity  BIGINT      NOT NULL DEFAULT 0,
    cost_basis         BIGINT      NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_positions_user_player UNIQUE (user_id, player_id),
    CONSTRAINT chk_positions_available_non_negative CHECK (available_quantity >= 0),
    CONSTRAINT chk_positions_reserved_non_negative CHECK (reserved_quantity >= 0),
    CONSTRAINT chk_positions_cost_basis_non_negative CHECK (cost_basis >= 0)
);

CREATE INDEX IF NOT EXISTS idx_positions_user ON user_positions (user_id);

CREATE TABLE IF NOT EXISTS market_orders (
    id                BIGSERIAL PRIMARY KEY,
    user_id           BIGINT       NOT NULL REFERENCES users (id),
    player_id         BIGINT       NOT NULL REFERENCES players (id),
    side              VARCHAR(4)   NOT NULL,
    time_in_force     VARCHAR(3)   NOT NULL,
    status            VARCHAR(20)  NOT NULL,
    price             BIGINT       NOT NULL,
    quantity          BIGINT       NOT NULL,
    executed_quantity BIGINT       NOT NULL DEFAULT 0,
    market_maker      BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_orders_side CHECK (side IN ('BUY', 'SELL')),
    CONSTRAINT chk_orders_time_in_force CHECK (time_in_force IN ('GTC', 'IOC', 'FOK')),
    CONSTRAINT chk_orders_status CHECK (status IN ('PENDING', 'PARTIALLY_FILLED', 'FILLED', 'CANCELLED')),
    CONSTRAINT chk_orders_price_positive CHECK (price > 0),
    CONSTRAINT chk_orders_quantity_positive CHECK (quantity > 0),
    CONSTRAINT chk_orders_executed_within_quantity CHECK (executed_quantity >= 0 AND executed_quantity <= quantity)
);

CREATE INDEX IF NOT EXISTS idx_orders_book
    ON market_orders (player_id, side, price, created_at)
    WHERE status IN ('PENDING', 'PARTIALLY_FILLED');

CREATE INDEX IF NOT EXISTS idx_orders_user ON market_orders (user_id, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS uq_orders_market_maker_live
    ON market_orders (player_id, side)
    WHERE market_maker AND status IN ('PENDING', 'PARTIALLY_FILLED');

CREATE TABLE IF NOT EXISTS market_trades (
    id            BIGSERIAL PRIMARY KEY,
    buy_order_id  BIGINT      NOT NULL REFERENCES market_orders (id),
    sell_order_id BIGINT      NOT NULL REFERENCES market_orders (id),
    buyer_id      BIGINT      NOT NULL REFERENCES users (id),
    seller_id     BIGINT      NOT NULL REFERENCES users (id),
    player_id     BIGINT      NOT NULL REFERENCES players (id),
    price         BIGINT      NOT NULL,
    quantity      BIGINT      NOT NULL,
    total_amount  BIGINT      NOT NULL,
    executed_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_trades_price_positive CHECK (price > 0),
    CONSTRAINT chk_trades_quantity_positive CHECK (quantity > 0),
    CONSTRAINT chk_trades_total_consistent CHECK (total_amount = price * quantity),
    CONSTRAINT chk_trades_distinct_parties CHECK (buyer_id <> seller_id)
);

CREATE INDEX IF NOT EXISTS idx_trades_buyer ON market_trades (buyer_id, executed_at DESC);
CREATE INDEX IF NOT EXISTS idx_trades_seller ON market_trades (seller_id, executed_at DESC);
CREATE INDEX IF NOT EXISTS idx_trades_player ON market_trades (player_id, executed_at DESC);

CREATE SEQUENCE IF NOT EXISTS ledger_settlement_seq;

CREATE TABLE IF NOT EXISTS ledger_entries (
    id               BIGSERIAL PRIMARY KEY,
    settlement_group BIGINT       NOT NULL,
    trade_id         BIGINT       NULL REFERENCES market_trades (id),
    user_id          BIGINT       NULL REFERENCES users (id),
    reason           VARCHAR(20)  NOT NULL,
    amount           BIGINT       NOT NULL,
    description      VARCHAR(255) NOT NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_ledger_reason CHECK (reason IN ('TRADE', 'INITIAL_GRANT')),
    CONSTRAINT chk_ledger_amount_non_zero CHECK (amount <> 0),
    CONSTRAINT chk_ledger_trade_required_for_trade CHECK ((reason = 'TRADE') = (trade_id IS NOT NULL))
);

CREATE INDEX IF NOT EXISTS idx_ledger_user ON ledger_entries (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ledger_settlement ON ledger_entries (settlement_group);

CREATE TABLE IF NOT EXISTS player_quotes (
    id               BIGSERIAL PRIMARY KEY,
    player_id        BIGINT      NOT NULL REFERENCES players (id) ON DELETE CASCADE,
    price            BIGINT      NOT NULL,
    strategy         VARCHAR(50) NOT NULL,
    strategy_version VARCHAR(20) NOT NULL,
    score_id         BIGINT      NULL REFERENCES player_scores (id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_quotes_price_positive CHECK (price > 0)
);

CREATE INDEX IF NOT EXISTS idx_quotes_player_latest ON player_quotes (player_id, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS uq_quotes_player_initial
    ON player_quotes (player_id)
    WHERE strategy = 'initial';
