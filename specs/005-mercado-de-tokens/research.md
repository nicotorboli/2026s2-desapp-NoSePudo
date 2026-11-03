# Research & Design Decisions: Player Token Market (Mercado de tokens)

**Feature**: `005-mercado-de-tokens`

**Status**: Completed

## 1. Context

Section 3.3 of the vision document describes the market in terms of a single operation at a single price: validate availability, use the player's active quote, update the position, record the operation, with the superuser as the initial counterparty. Section 4 names two endpoints for it, `POST /orders/buy` and `POST /orders/sell`.

This feature deliberately implements more than that minimum: a real order book with a price-time priority matching engine, where the superuser is a market maker rather than a hardcoded counterparty. The reasons, and the cost that choice carries, are recorded here so the deviation is justified rather than implicit. Section 7 lists what the extra machinery buys and what it costs.

## 2. Decisions

### Decision 1: Order book with the superuser as market maker

Users submit orders that cross against resting orders of the opposite side. At moment zero the superuser holds the 100 tokens of each player and rests a sell order for them at the active quote, so the book is never empty and the market is tradeable from the first request.

- **Why not a direct purchase against the superuser**: with a book, users can trade with each other without the superuser being involved, and the counterparty stops being a special case in the code. The market maker becomes just another participant whose orders happen to be placed by the system.
- **Why not a pure peer-to-peer book**: with no resting inventory, nothing can execute until two users happen to want opposite sides of the same player, which would make the feature undemonstrable during evaluation.
- **Cost accepted**: resting orders imply reservations, partial executions and cancellation, which is roughly half of the implementation and test surface of this feature. Decision 3 bounds that cost.

### Decision 2: The active quote is the default price, not the only price

`price` is optional in the request. When absent, the order is priced at the player's active quote, which is what section 3.3 requires. When present, the order is a genuine limit order and may rest without crossing.

- **Why**: this keeps the enunciado's reading of the market as the default behavior of the API while leaving the book meaningful. The common path (`{playerId, quantity}`) is exactly the operation the vision document describes.
- **Consequence**: the quote remains the reference price of the system, produced by the valuation strategies of section 3.2. The book determines the execution price only when a user deliberately names one.

### Decision 3: Time in force bounds the lifetime of a remainder

Every order carries a `timeInForce` that decides what happens to the part that could not execute:

| Value | Remainder behavior |
|---|---|
| `GTC` | Rests in the book with its reservation held, until it crosses or is cancelled |
| `IOC` | Cancelled immediately, releasing its reservation |
| `FOK` | The whole order is rejected if it cannot execute in full, leaving no state at all |

Defaults follow the price, which is the user's signal of intent: an order without a price is `FOK` ("I want this quantity now, at today's value, or not at all"), and an order with a price is `GTC` ("naming a price means I am willing to wait").

- **Why**: it answers the starvation worry about a large order waiting forever without inventing a mechanism. Price-time priority already prevents later, smaller orders from jumping ahead of a resting remainder at the same price; time in force is what lets a user opt out of waiting at all.
- **Why it is cheap**: the matching function already computes how much can execute now. Time in force only decides what to do with the rest, so it is one column and one branch, and it collapses the "should orders rest at all" question into a per-order choice instead of an architectural one.
- **Not now**: an expiry timestamp for resting orders. It needs a scheduler, which arrives with section 3.2; sweeping expired orders can then join the same cycle that recalculates quotes.

### Decision 4: Partial executions are required, not optional

If orders rest, they must be allowed to execute partially. Otherwise two orders that agree on price but not on quantity cannot trade, resting orders block each other, and a large order can only ever be filled by a single counterparty of equal or greater size, which is the actual starvation scenario.

### Decision 5: Fixed-point credits, never binary floating point

Credits are carried as a dedicated integer type in units of 1/10000 of a credit, and converted to a decimal representation only at the HTTP edge. Every money column is `BIGINT` in those same scaled units, for the reasons in section 1 of [data-model.md](./data-model.md): `database/sql` maps it to `int64` with no conversion code in any DAO, and a quantity times a price stays exact by construction.

- **Why**: `float64` cannot represent most decimal credit amounts exactly, and the average purchase price accumulates error across executions. SC-002 asks for exactness to four decimal places, which a binary float cannot guarantee.
- **Why not a decimal library**: the constitution restricts dependencies to the standard library plus the pinned set. An integer type with explicit conversion is enough, and keeps the multiplication of a quantity by a price exact by construction.

### Decision 6: The book is built per transaction, not kept in memory

The matching logic is a pure function over a slice of resting orders. The resting orders are read and locked inside the transaction that is placing the new order, matched in memory for the duration of that transaction, and written back before commit.

- **Why not a long-lived in-memory book**: it would be a second source of truth next to the database, requiring rebuild on boot, invalidation, and a single writer per player, which rules out running more than one backend instance. The benefit it usually buys is latency this system does not need.
- **What is preserved**: the matching function stays pure and unit-testable without a database, which was the reason to isolate it in the first place.

### Decision 7: Serialization by player, not by user pair

Before touching accounts, positions or orders, a transaction takes a row lock on the player being traded. Everything else in the operation happens under that lock.

- **Why**: an execution touches two accounts, two positions, several resting orders and the quote. Ordering locks by user id alone leaves positions and orders unordered, so concurrent executions can still interleave and deadlock. The player is the natural unit of contention: one lock serializes the whole book of that player, which is exactly the invariant that must hold, and orders on different players stay concurrent.

### Decision 8: Both audit mechanisms, with distinct purposes

- The **audit entry** records the actor, the previous state, the resulting state and the timestamp of every change to a balance, position or order. This is the format section 5.2 of the vision document requires, and it reuses the audit entity that already exists in the system rather than inventing a second shape for it.
- The **accounting entries** record every monetary movement as signed debit and credit pairs that sum to zero per settlement. This is what makes "no credits were created or destroyed" a checkable invariant rather than a claim.

The two are not redundant: the audit entry answers "who changed what and from what to what", the accounting entries answer "does the money still add up". The initial balance granted at registration is recorded as an accounting entry with no associated trade, so the ledger reconciles from the first user onward.

### Decision 9: Trading never computes a valuation

The market reads the most recent quote of a player and nothing else. The valuation strategies of section 3.2 only write quotes. The single point of contact in the other direction is the market maker's resting orders, which are re-priced when a new quote appears so that the system never keeps offering tokens at a stale price.

- **Why this matters**: without re-pricing, the first recalculation leaves the market maker offering 100 tokens at the old price, which any user can take at a discount. It is a consequence of combining a book with an externally computed price, so it belongs to this feature and not to 3.2.

### Decision 10: Edge DTOs live in the controller's DTO package, with camelCase fields

The repository has two DTO packages. `internal/dto` holds the shapes that the service layer itself returns, such as the player catalog and synchronization results. `internal/controller/dto` holds the shapes that exist only at the HTTP edge. Market services return domain models, so market DTOs belong in `internal/controller/dto`.

Field names follow the published contract of the domain resources, which is camelCase (`externalId`, `leagueCode`, `totalPages`). The snake_case fields of the session DTO are an OAuth-shaped exception and are not the convention to copy.

## 3. Corrections applied to the previous revision of this design

These were defects in the earlier draft of the plan and data model, kept here so they are not reintroduced.

1. **Money as `float64`** in the domain structs while the schema used `NUMERIC(15, 4)`, which breaks SC-002. Replaced by the fixed-point credit type of Decision 5.
2. **Cost basis was never reduced on a sale**: the sell path only decremented the token count, so the average purchase price of a position rose after every partial sale and the profit or loss was wrong. The position now releases cost basis proportionally, which is FR-010.
3. **The fill operation did not receive the execution price**, so an order crossing a cheaper resting order had no way to release the difference between the reserved and executed amounts, leaving credits reserved forever. The execution price is now an explicit input of the fill.
4. **The reservation mutators did not validate and could not fail**, so an inconsistent release would silently corrupt a balance and only surface as a database constraint violation three layers away. They now validate and return domain errors, like their locking counterparts.
5. **The bootstrap was not idempotent**: nothing prevented a second run from issuing another 100 tokens, recording another initial quote or resting a duplicate market maker order. Enforced by partial unique indexes rather than by application convention.
6. **Lock ordering by user id was insufficient** for what an execution actually touches. Replaced by Decision 7.
7. **The accounting entries could not satisfy the invariant they claimed**: unsigned amounts with a type column cannot be constrained to balance, and a mandatory trade reference made the initial balance grant unrecordable. Amounts are now signed and the trade reference is optional.
8. **The quote had no strategy version**, which section 3.2 explicitly requires to be recorded per quote. Added now to avoid a later migration.
9. **The in-memory book and the pessimistic locking were specified together**, which are two sources of truth. Resolved by Decision 6.
10. **`MARKET` orders were specified with a mandatory positive price** and no rule for how much to reserve on a buy whose execution price is unknown. The order type is dropped: an order without a price priced at the active quote covers the same use case, and time in force covers the rest.

## 4. Testing approach

Per constitution principle XIII, with equivalence classes and boundary values:

- **Pure unit tests, no database**: order state transitions including every invalid transition; the matching function over handcrafted books (price priority, time priority at equal price, self-crossing skipped, multiple counterparties, insufficient liquidity); reservation arithmetic; average purchase price and profit or loss, including the boundary of a position reduced to zero.
- **Repository integration tests with testcontainers**: every query, the partial unique indexes that enforce idempotency, and the database check constraints that forbid negative balances and positions.
- **Service integration tests with testcontainers**: one test per time in force against the same book state, the release of a reservation on every non-executing path, cancellation of a partially executed order, a remainder that crosses later, cost basis after mixed purchases and sales, and market maker re-pricing after a new quote.
- **Concurrency tests**: concurrent goroutines competing for the last tokens of one player, and concurrent orders from one user competing for the same credits.
- **Controller integration tests**: status codes and payloads of every endpoint, authorization of another user's portfolio, history and orders, and the published contract.
- **End-to-end**: the evaluation scenario of section 8.3, four users buying tokens of five players and reporting their valuation after the quotes move.

## 5. Scope boundaries

Out of this feature, and explicitly not stubbed here: the valuation strategies and quote recalculation, the ranking, the quote history endpoints, the job scheduler and the cache layer. This feature defines where quotes live and reads the most recent one.

## 6. Features this one touches

The market is additive in its data, but not isolated in its behavior. Three delivered features change on their write path, and their existing tests must keep passing untouched:

- **Registration (003)**: creating a user now also creates an account with the configured initial balance, and the superuser bootstrap now also needs an account, a position per player and its resting orders.
- **Player synchronization (002)**: synchronizing the catalog now also triggers the market bootstrap for new players. The synchronization runs against an external API and can run many times, which is why the bootstrap must be idempotent at the schema level.
- **Published contract (004)**: every new route must be registered, documented with its access level and consistent with the contract test. A route that declares no access level fails at startup by design.

## 7. Complexity justification

The constitution requires that complexity not foreseen by the enunciado be justified formally. What the order book adds beyond section 3.3, and why it is accepted:

| Added | Why it is worth it | What it costs |
|---|---|---|
| Resting orders and the matching engine | Users can trade with each other; the counterparty is no longer a special case in the code | Reservations, partial executions, cancellation: about half of the implementation |
| `GET /orders` and `DELETE /orders/{id}` beyond the endpoints of section 4 | Without them, the reservation of an order that never crosses is retained indefinitely | Two endpoints and their contract, plus an order management screen in the frontend of section 9 |
| Time in force | Lets a user choose not to wait, instead of a hardcoded answer to that question | One column and one branch in the matching path |
| Market maker re-pricing | Without it, the first quote recalculation leaves inventory offered at a stale price | A re-pricing path to run whenever a quote is recorded |
| Double-entry accounting on top of the audit entries | Makes conservation of credits a checkable invariant | A second recording mechanism and its tests |

If this cost ever has to be cut, the collapse is mechanical rather than a redesign: fixing every order to `FOK` removes resting orders, and with them reservations, cancellation, partial executions and the two extra endpoints, leaving the matching function and the domain model intact.
