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
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_quotes_price_positive CHECK (price > 0)
);

CREATE INDEX IF NOT EXISTS idx_quotes_player_latest ON player_quotes (player_id, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS uq_quotes_player_initial
    ON player_quotes (player_id)
    WHERE strategy = 'initial';
