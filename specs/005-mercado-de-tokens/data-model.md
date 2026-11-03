# Data Model: Player Token Market (Mercado de tokens)

**Feature**: `005-mercado-de-tokens`

## 1. Credit representation

Credits are stored and carried as integers in units of one ten-thousandth of a credit, which is the precision SC-002 requires. One credit is `10000`. The scale lives in exactly two places: the domain type `Credits` and the `BIGINT` columns below.

This is preferred over `NUMERIC(15, 4)` because `database/sql` maps `BIGINT` to `int64` with no conversion code in any of the six DAOs, and because a quantity times a price stays exact by construction. The cost is that raw SQL reads in scaled units; every balance column is named so that this is unmistakable, and the test helpers convert for readability.

Token quantities are whole tokens, `BIGINT` throughout. Tokens are never fractional.

## 2. Relational schema

### 2.1 `user_accounts`

One row per user, created at registration. The available balance is what the user can commit; the reserved balance is what live orders have committed and is not spendable twice.

```sql
CREATE TABLE IF NOT EXISTS user_accounts (
    user_id           BIGINT PRIMARY KEY REFERENCES users (id),
    available_balance BIGINT      NOT NULL DEFAULT 0,
    reserved_balance  BIGINT      NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_accounts_available_non_negative CHECK (available_balance >= 0),
    CONSTRAINT chk_accounts_reserved_non_negative CHECK (reserved_balance >= 0)
);
```

No index on `user_id`: it is the primary key. The initial balance comes from `NSP_INITIAL_USER_BALANCE` through the injected configuration, not from a column default, so there is a single source for that value. The foreign key has no `ON DELETE` clause on purpose: users are deactivated through `users.active`, never hard deleted, and trades must keep referring to existing users.

### 2.2 `user_positions`

One row per user and player. Reserved tokens are those committed by a live sell order. `cost_basis` is the total paid for the tokens currently held, which is what makes the average purchase price derivable without reading the whole trade history.

```sql
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
```

`uq_positions_user_player` is what makes the token issuance of FR-001 idempotent: the bootstrap inserts the superuser position with `ON CONFLICT DO NOTHING`, so a second run cannot issue a second batch of 100 tokens.

The superuser's position starts with a `cost_basis` of zero, because its tokens were issued rather than bought. Its average purchase price is therefore zero and its unrealized result equals its whole valuation. That is the correct reading for issued inventory, not a defect: the market maker did not pay for the tokens it is offering. Its 100 tokens sit in `reserved_quantity`, not `available_quantity`, because they are committed to its resting sell order.

### 2.3 `market_orders`

Every order that executed something or rests. An order rejected in full under `FOK` is never inserted, which is what FR-004 means by leaving no state at all.

```sql
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
```

Three things to note. `idx_orders_book` is partial because the only orders ever matched are the live ones, which keeps the index small as history grows; a single btree serves both sides because Postgres scans it backwards for the descending half. The `market_maker` flag exists so the re-pricing path of FR-011 can find the market maker's orders without knowing the superuser id, and so the uniqueness guard below can be scoped to them. `uq_orders_market_maker_live` is what makes the bootstrap of FR-001 idempotent: at most one live market maker order per player and side, so a restart cannot duplicate the offer. Regular users are unconstrained in how many live orders they hold.

There is no order type column. An order without a price is priced at the active quote, which covers what a market order would have been, and the earlier `MARKET` type is dropped for the reasons in research decision 10.

### 2.4 `market_trades`

One row per execution. Immutable: only inserted, never updated.

```sql
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
```

`chk_trades_total_consistent` makes the stored total derivable rather than trusted, and `chk_trades_distinct_parties` is the self-crossing rule of the edge cases enforced at the last possible layer, so a bug in the matcher fails loudly instead of recording a trade with itself.

### 2.5 `ledger_entries`

Signed accounting entries. Entries of one settlement share a `settlement_group` and must sum to zero. A null `user_id` is the platform's own account, which is what makes the initial balance grant balance: the user is credited and the platform is debited.

```sql
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
```

The zero-sum invariant cannot be expressed as a row constraint, so it is a checkable property instead of a claim:

```sql
SELECT settlement_group
FROM ledger_entries
GROUP BY settlement_group
HAVING SUM(amount) <> 0;
```

That query returning no rows is an assertion in the repository integration tests and in the end-to-end test, run after the whole scenario rather than per operation.

### 2.6 `player_quotes`

The valuation history of a player's token. This feature writes only the initial quote and reads the most recent one; the strategies of section 3.2 write the rest, which is why the strategy and its version are columns from the start.

```sql
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
```

`uq_quotes_player_initial` is the third idempotency guard of the bootstrap: the moment-zero quote of a player can exist only once. The initial quote is written as `strategy = 'initial'`, `strategy_version = '1'`.

### 2.7 Audit entries: no new table

The audit requirement of FR-012 and of section 5.2 of the vision document is already served by what exists. `model.AuditLog` carries the actor, the previous state, the resulting state and the timestamp, and `dao.AuditDAO.InsertAuditLog` accepts the enclosing transaction and is generic over `entity_type` and `entity_id`. The market calls it with `entity_type` values of `user_account`, `user_position` and `market_order`.

No schema change, no new DAO, no change to feature 002. The only wart is that the table is still named `player_audit_logs` while it is now generic in use. Renaming it would touch the DAO and the tests of a delivered feature, so it is left alone here and noted as an opt-in cleanup of its own.

## 3. Domain model (`backend/internal/model`)

### 3.1 Credits

```go
package model

type Credits int64

const CreditScale = 10000

func CreditsFromUnits(units int64) Credits {
	return Credits(units * CreditScale)
}

func (c Credits) Times(quantity int64) Credits {
	return Credits(int64(c) * quantity)
}

func (c Credits) Plus(other Credits) Credits {
	return c + other
}

func (c Credits) Minus(other Credits) Credits {
	return c - other
}

func (c Credits) DividedBy(quantity int64) Credits {
	if quantity == 0 {
		return 0
	}
	return Credits(int64(c) / quantity)
}
```

The DTO layer is the only place that renders `Credits` as a decimal number, and the only place that parses one.

### 3.2 Order

```go
type OrderSide string

const (
	SideBuy  OrderSide = "BUY"
	SideSell OrderSide = "SELL"
)

type TimeInForce string

const (
	GoodTillCancelled TimeInForce = "GTC"
	ImmediateOrCancel TimeInForce = "IOC"
	FillOrKill        TimeInForce = "FOK"
)

type OrderStatus string

const (
	StatusPending         OrderStatus = "PENDING"
	StatusPartiallyFilled OrderStatus = "PARTIALLY_FILLED"
	StatusFilled          OrderStatus = "FILLED"
	StatusCancelled       OrderStatus = "CANCELLED"
)

type Order struct {
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Side             OrderSide
	TimeInForce      TimeInForce
	Status           OrderStatus
	ID               int64
	UserID           int64
	PlayerID         int64
	Quantity         int64
	ExecutedQuantity int64
	Price            Credits
	MarketMaker      bool
}

func (o *Order) RemainingQuantity() int64 {
	return o.Quantity - o.ExecutedQuantity
}

func (o *Order) IsLive() bool {
	return o.Status == StatusPending || o.Status == StatusPartiallyFilled
}

func (o *Order) Execute(quantity int64, at time.Time) error {
	if !o.IsLive() {
		return ErrOrderNotLive
	}
	if quantity <= 0 || quantity > o.RemainingQuantity() {
		return ErrInvalidQuantity
	}
	o.ExecutedQuantity += quantity
	if o.RemainingQuantity() == 0 {
		o.Status = StatusFilled
	} else {
		o.Status = StatusPartiallyFilled
	}
	o.UpdatedAt = at
	return nil
}

func (o *Order) Cancel(at time.Time) error {
	if !o.IsLive() {
		return ErrOrderNotLive
	}
	o.Status = StatusCancelled
	o.UpdatedAt = at
	return nil
}
```

`Execute` and `Cancel` take the timestamp as a parameter instead of calling `time.Now()`, so the state machine stays a pure function of its inputs and its tests are deterministic without a clock abstraction.

The execution price is not a field of the order's state transition, because the order keeps its own limit price; it is the matcher that reports the price of each execution, and the settlement that uses it. That split is what makes the reservation release of FR-004 expressible at all.

### 3.3 Account

```go
type Account struct {
	UserID           int64
	AvailableBalance Credits
	ReservedBalance  Credits
}

func (a *Account) Reserve(amount Credits) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if a.AvailableBalance < amount {
		return ErrInsufficientFunds
	}
	a.AvailableBalance -= amount
	a.ReservedBalance += amount
	return nil
}

func (a *Account) ReleaseReservation(amount Credits) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if a.ReservedBalance < amount {
		return ErrReservationTooSmall
	}
	a.ReservedBalance -= amount
	a.AvailableBalance += amount
	return nil
}

func (a *Account) SettleReservation(amount Credits) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if a.ReservedBalance < amount {
		return ErrReservationTooSmall
	}
	a.ReservedBalance -= amount
	return nil
}

func (a *Account) Credit(amount Credits) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	a.AvailableBalance += amount
	return nil
}
```

Every mutator validates and returns an error. The release and settle paths are where a wrong amount would silently corrupt a balance, so they fail at the cause rather than at a database constraint three layers away.

### 3.4 Position

```go
type Position struct {
	UserID            int64
	PlayerID          int64
	AvailableQuantity int64
	ReservedQuantity  int64
	CostBasis         Credits
}

func (p *Position) TotalQuantity() int64 {
	return p.AvailableQuantity + p.ReservedQuantity
}

func (p *Position) AveragePrice() Credits {
	return p.CostBasis.DividedBy(p.TotalQuantity())
}

func (p *Position) ReserveTokens(quantity int64) error {
	if quantity <= 0 {
		return ErrInvalidQuantity
	}
	if p.AvailableQuantity < quantity {
		return ErrInsufficientTokens
	}
	p.AvailableQuantity -= quantity
	p.ReservedQuantity += quantity
	return nil
}

func (p *Position) ReleaseReservedTokens(quantity int64) error {
	if quantity <= 0 {
		return ErrInvalidQuantity
	}
	if p.ReservedQuantity < quantity {
		return ErrReservationTooSmall
	}
	p.ReservedQuantity -= quantity
	p.AvailableQuantity += quantity
	return nil
}

func (p *Position) DeliverReservedTokens(quantity int64) (Credits, error) {
	if quantity <= 0 {
		return 0, ErrInvalidQuantity
	}
	if p.ReservedQuantity < quantity {
		return 0, ErrReservationTooSmall
	}
	released := p.CostBasis
	if quantity < p.TotalQuantity() {
		released = p.CostBasis.Times(quantity).DividedBy(p.TotalQuantity())
	}
	p.ReservedQuantity -= quantity
	p.CostBasis -= released
	return released, nil
}

func (p *Position) Receive(quantity int64, cost Credits) error {
	if quantity <= 0 || cost <= 0 {
		return ErrInvalidQuantity
	}
	p.AvailableQuantity += quantity
	p.CostBasis += cost
	return nil
}
```

`DeliverReservedTokens` is the fix for the defect recorded as correction 2 in the research: a sale releases cost basis in proportion to the quantity delivered, so the average purchase price of what remains is unchanged by the sale, which is FR-010. The full-position branch is not an optimization: it assigns the entire remaining cost basis so that integer truncation cannot leave a residue behind on a position that reached zero tokens.

The credits released are returned rather than discarded, because the realized result of the sale is the difference between what was received and that number.

### 3.5 Matching

The matcher is a pure function. It never reads a database, never mutates its inputs and never looks at a clock.

```go
type Execution struct {
	RestingOrderID int64
	Quantity       int64
	Price          Credits
}

func MatchOrder(incoming Order, book []Order) []Execution
```

Its rules, each one a test case:

| Rule | Behavior |
|---|---|
| Eligible side | A buy order matches only sell orders of the same player, and the reverse |
| Crossable price | A buy matches resting prices at or below its own; a sell matches resting prices at or above its own |
| Price priority | A buy consumes the cheapest resting order first; a sell consumes the most expensive first |
| Time priority | At equal price, the older resting order is consumed first |
| Self-crossing | Resting orders of the submitter are skipped, never matched |
| Execution price | Each execution is priced at the resting order's price, not the incoming order's |
| Exhaustion | Each resting order contributes at most its remaining quantity; the walk stops when the incoming remainder reaches zero or no eligible order is left |

The caller decides what to do with the remainder, from the order's time in force. That is the only place the three values differ:

| Time in force | Remainder handling |
|---|---|
| `GTC` | The order is persisted live with its reservation held |
| `IOC` | The order is persisted `CANCELLED` and the remainder's reservation is released |
| `FOK` | Nothing is persisted and the whole operation is rolled back |

An order that executed nothing and does not rest is never persisted, which keeps the order table a record of what happened rather than of what was attempted.

### 3.6 Quote, trade and ledger entry

```go
type PlayerQuote struct {
	CreatedAt       time.Time
	Strategy        string
	StrategyVersion string
	ID              int64
	PlayerID        int64
	Price           Credits
}

type Trade struct {
	ExecutedAt  time.Time
	ID          int64
	BuyOrderID  int64
	SellOrderID int64
	BuyerID     int64
	SellerID    int64
	PlayerID    int64
	Quantity    int64
	Price       Credits
	TotalAmount Credits
}

type LedgerReason string

const (
	ReasonTrade        LedgerReason = "TRADE"
	ReasonInitialGrant LedgerReason = "INITIAL_GRANT"
)

type LedgerEntry struct {
	CreatedAt       time.Time
	Reason          LedgerReason
	Description     string
	ID              int64
	SettlementGroup int64
	TradeID         *int64
	UserID          *int64
	Amount          Credits
}
```

`TradeID` and `UserID` are pointers because both are genuinely optional: the initial grant has no trade, and the platform's own side of a settlement has no user. Struct fields are ordered by descending size to satisfy `govet fieldalignment`, which the constitution enables.

### 3.7 Portfolio

```go
type PortfolioPosition struct {
	PlayerName       string
	ClubName         string
	PlayerPosition   string
	PlayerID         int64
	Quantity         int64
	AveragePrice     Credits
	CurrentPrice     Credits
	CurrentValuation Credits
	UnrealizedResult Credits
}

type Portfolio struct {
	Positions        []PortfolioPosition
	UserID           int64
	AvailableCash    Credits
	ReservedCash     Credits
	TokensValuation  Credits
	TotalValue       Credits
	UnrealizedResult Credits
}

func BuildPortfolio(userID int64, account Account, positions []Position, quotes map[int64]Credits, players map[int64]Player) Portfolio
```

`Quantity` per position is the total owned, reserved tokens included, because tokens committed to a sell order are still owned. Reserved cash is reported separately from available cash so the user can see why part of the balance is not spendable. `BuildPortfolio` is pure: it receives the quotes and players already resolved and does arithmetic only, which is what makes the valuation and result assertions of SC-002 unit-testable.

## 4. Order lifecycle

```
Submit (side, playerId, quantity, price?, timeInForce?)
  |
  +- price absent        -> price = active quote of the player, timeInForce defaults to FOK
  +- price present       -> timeInForce defaults to GTC
  |
  +- lock the player row                      (serializes this player's whole book)
  +- validate player exists and is active
  +- BUY : account.Reserve(quantity * price)  | SELL: position.ReserveTokens(quantity)
  +- read the live opposite side of the book, locked
  +- MatchOrder(incoming, book)
  |
  +- for each execution:
  |    buyer order.Execute / seller order.Execute
  |    buyer account.SettleReservation(quantity * executionPrice)
  |    buyer account.ReleaseReservation(quantity * (orderPrice - executionPrice))   when the price improved
  |    seller position.DeliverReservedTokens(quantity)
  |    seller account.Credit(quantity * executionPrice)
  |    buyer position.Receive(quantity, quantity * executionPrice)
  |    insert trade, two ledger entries sharing one settlement group, audit entries
  |
  +- remainder, by timeInForce:
       GTC -> persist the order live, reservation stays held
       IOC -> persist CANCELLED, release the remainder's reservation
       FOK -> roll back the transaction, return 409

Cancel (orderId)
  +- lock the player row, load the order
  +- reject when not the owner (403) or not live (409)
  +- order.Cancel
  +- BUY : account.ReleaseReservation(remaining * price) | SELL: position.ReleaseReservedTokens(remaining)
  +- audit entry
```

The reservation release on price improvement is the step whose absence was correction 3 in the research: a buy order that reserved at its own limit price and executed cheaper must get the difference back, or those credits stay reserved by an order that is already final.

## 5. Invariants and where each is enforced

| Invariant | Enforcement |
|---|---|
| No balance and no position goes negative | Database check constraints, plus domain errors at the cause |
| Credits are neither created nor destroyed by trading | Ledger settlement groups sum to zero, asserted as a query in tests |
| Exactly 100 tokens exist per player | Issuance guarded by `uq_positions_user_player` with `ON CONFLICT DO NOTHING`; trades only move tokens between positions; asserted per player as a sum over all positions |
| One initial quote per player | `uq_quotes_player_initial` |
| One live market maker order per player and side | `uq_orders_market_maker_live` |
| No reservation outlives its order | Every non-executing path releases it; asserted as a cross-check that the sum of reserved balances equals the sum of what live orders reserve |
| No order trades with itself | Skipped by the matcher, and refused by `chk_trades_distinct_parties` |
| An execution's total is its price times its quantity | `chk_trades_total_consistent` |
| Concurrent orders over one player serialize | Row lock on the player, taken before anything else in the operation |

## 6. Configuration

| Variable | Default | Used for |
|---|---|---|
| `NSP_INITIAL_USER_BALANCE` | `1000` | Credits granted at registration, in whole credits, converted to scaled units on read |

The existing `NSP_SUPERUSER_EMAIL` and `NSP_SUPERUSER_PASSWORD` already identify the market maker; no new variable is needed for it. The 100 tokens per player and the initial quote of 1 credit are domain constants from the enunciado, not configuration.
