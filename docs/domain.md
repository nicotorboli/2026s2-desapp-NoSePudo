# Domain model

The rules the backend must preserve, independent of transport. Owning package and enforcement
point are named for each rule; line numbers are from the current tree.

Inventory (26 non-test files, all in `internal/model`):

```bash
find backend/internal/model -maxdepth 1 -name '*.go' -not -name '*_test.go' | sort
```

`internal/model` imports the standard library only and carries no JSON tags
(`grep -rn 'json:"' backend/internal/model/` is empty). Persistence constraints live in
`backend/db/init.sql`; the model repeats the invariants where a caller could violate them
without touching the database.

## Credits

Owning package: `model`.

- `Credits` is an `int64` in scaled units, not a decimal type (`model/credits.go:3`).
  `CreditScale = 10000`, so one credit is 10,000 units (`model/credits.go:5`).
- `CreditsFromUnits(units)` multiplies by the scale (`model/credits.go:7-9`).
- `Times(quantity)` multiplies by an integer token quantity (`model/credits.go:11-13`);
  `Plus` and `Minus` are additive (`model/credits.go:15-21`).
- `DividedBy(quantity)` performs integer division: it truncates toward zero and returns `0`
  when the divisor is zero (`model/credits.go:23-28`). Callers must not rely on it rounding.
- The wire format is a decimal string with exactly four fraction digits
  (`dto/credits.go:9-25`). Parsing rejects empty input, more than four decimals, and values that
  would overflow the scaled `int64` (`controller/dto/market.go:42-76`).
- Every monetary database column is `BIGINT`: order price (`db/init.sql:227`), trade price and
  total (`db/init.sql:258-260`), ledger amount (`db/init.sql:280`), quote price
  (`db/init.sql:294`), P2P price (`db/init.sql:315`), account balances
  (`db/init.sql:195-196`), and position cost basis (`db/init.sql:209`).

## Accounts and the double-entry ledger

Owning package: `model` for the arithmetic; `persistence/repository` for the transaction.

- An account has `AvailableBalance` and `ReservedBalance`
  (`model/account.go:13-17`). `Reserve` moves credits from available to reserved and fails on a
  non-positive amount or insufficient available funds (`model/account.go:19-31`).
  `ReleaseReservation` moves them back (`model/account.go:33-45`); `SettleReservation` drops
  reserved credits without returning them (`model/account.go:47-58`); `Credit` adds available
  credits (`model/account.go:60-68`).
- Non-negativity of both balances is enforced twice: in the model methods and by
  `chk_accounts_available_non_negative` / `chk_accounts_reserved_non_negative`
  (`db/init.sql:199-200`).
- A `LedgerEntry` belongs to a `SettlementGroup` and carries an optional trade and user, a
  reason (`TRADE` or `INITIAL_GRANT`), a non-zero signed amount, and a description
  (`model/trade.go:19-35`). The schema enforces the reason set, `amount <> 0`, and that a
  `TRADE` entry always names a trade (`db/init.sql:274-286`).
- Settlement groups come from the `ledger_settlement_seq` sequence, allocated inside the same
  transaction that writes the entries (`persistence/dao/ledger_sql.go:19-31`,
  `db/init.sql:272`).
- The ledger is double-entry: each group must net to zero. `UnbalancedSettlementGroups` is the
  read-side check (`persistence/dao/ledger_sql.go:57-62`). Trade settlements write a negative
  buyer entry and a positive seller entry in one group
  (`persistence/repository/market.go:317-353`, `persistence/repository/p2p.go:282-313`).
- Registration opens the account and writes the initial grant: a positive entry for the user and
  an offsetting negative entry in the same settlement group
  (`persistence/repository/registration.go:63-105`). The amount comes from
  `NSP_INITIAL_USER_BALANCE`, default 1000 credits (`configuration/configuration.go:22`,
  `configuration/configuration.go:134-147`), and is scaled before it reaches the repository
  (`cmd/server/main.go:104-106`).
- The account row is the live balance; the ledger is the append-only audit trail. A change that
  alters a balance without a ledger entry is a domain bug even if the row stays consistent.

Invariants: available and reserved are never negative; each settlement group sums to zero;
initial grants are paired; a trade settlement names the same trade on both entries.

## Positions

Owning package: `model`.

- A position tracks `AvailableQuantity`, `ReservedQuantity`, and `CostBasis`
  (`model/position.go:10-16`). `TotalQuantity` is the sum (`model/position.go:18-20`);
  `AveragePrice` is cost basis divided by total, with the truncation from `Credits.DividedBy`
  (`model/position.go:22-24`).
- `ReserveTokens` moves tokens from available to reserved and returns
  `ErrInsufficientTokens` when available is short or `ErrInvalidQuantity` for a non-positive
  quantity (`model/position.go:6`, `model/position.go:26-38`).
- `ReleaseReservedTokens` moves them back (`model/position.go:40-52`).
  `DeliverReservedTokens` removes reserved tokens and releases a proportional share of the cost
  basis; when the whole position is delivered it releases the full basis
  (`model/position.go:54-71`). `Receive` adds tokens and cost basis
  (`model/position.go:73-82`).
- The schema has one row per `(user_id, player_id)`, and checks that available, reserved, and
  cost basis are non-negative (`db/init.sql:203-216`).

Invariants: available plus reserved equals the total holding; reserved never exceeds the total;
cost basis never goes negative; a user has at most one position per player.

## Orders and trades

Owning package: `model` for matching and order state; `internal/service` for orchestration.

- Sides are `BUY` and `SELL` (`model/order.go:14-19`); time in force is `GTC`, `IOC`, or `FOK`
  (`model/order.go:21-27`); status is `PENDING`, `PARTIALLY_FILLED`, `FILLED`, or `CANCELLED`
  (`model/order.go:29-36`). The schema repeats all three sets as check constraints
  (`db/init.sql:233-235`).
- `RemainingQuantity` is quantity minus executed (`model/order.go:53-55`); `IsLive` is pending or
  partially filled (`model/order.go:57-59`).
- `Execute` requires a live order and `0 < quantity <= remaining`, then moves to `FILLED` or
  `PARTIALLY_FILLED` (`model/order.go:61-78`). `Cancel` requires a live order
  (`model/order.go:80-89`). The DB also enforces `executed_quantity <= quantity`
  (`db/init.sql:238`).
- Matching considers only resting orders for the same player, on the opposite side, owned by a
  different user, still live with remaining quantity, and at a crossing price
  (`model/matching.go:51-73`). Price-time priority is: best price first (lowest for a buy,
  highest for a sell), then `CreatedAt`, then `ID` (`model/matching.go:75-87`).
- A trade records buyer, seller, player, quantity, price, and total; `TotalAmount` must equal
  `price * quantity` in the same unit, buyer and seller must differ, and the schema enforces both
  (`model/trade.go:5-17`, `db/init.sql:264-265`).
- Market trades set `BuyOrderID` and `SellOrderID` and leave `P2PSessionID` nil
  (`persistence/repository/market.go:355-366`). P2P trades set `P2PSessionID` and leave the order
  ids zero (`persistence/repository/p2p.go:264-275`). The schema makes both order columns nullable
  and adds the P2P FK (`db/init.sql:330-337`).
- Cancelling an order releases the remaining reservation: credits for a buy, tokens for a sell
  (`service/market.go:416-451`, `service/market.go:492-533`).
- `FOK` with a remainder fails as `ErrNoLiquidity`; `IOC` cancels the remainder and releases it;
  `GTC` leaves the remainder resting (`service/market.go:365-387`).

Invariants: executed never exceeds quantity; only live orders match; an order never matches its
own owner; total amount is price times quantity; buyer and seller are distinct.

## Quotes and scores

Owning package: `model` for scoring math; `internal/service` for activation and persistence.

- A quote records player, price, strategy name, strategy version, and an optional score id
  (`model/quote.go:18-26`). "Current quote" means the latest row by `created_at`, then `id`; there
  is no expiry column (`persistence/dao/quote_sql.go:67-103`).
- Exactly one `initial` quote may exist per player, enforced by a partial unique index
  (`db/init.sql:304-306`). A duplicate insert is translated to `ErrQuoteAlreadyExists`
  (`persistence/dao/quote_sql.go:58-60`); a missing latest quote is `ErrQuoteNotFound`
  (`persistence/dao/quote_sql.go:93-98`).
- A histogram has 1,000 buckets (`model/histogram.go:3`). `BuildHistogram` counts scores and
  clamps out-of-range values into the first or last bucket (`model/histogram.go:5-11`,
  `model/histogram.go:35-44`). `Percentile` is the share of the population below the bucket plus
  half the bucket itself (`model/histogram.go:13-33`).
- A strategy scores a player as `alpha * rating/10 + (1-alpha) * weighted stats`
  (`model/scoring_strategy.go:45-55`); unknown positions fall back to the unweighted average of
  all statistics (`model/scoring_strategy.go:57-64`). Each statistic is normalized to `[0,1]`,
  and fouls and `was_dribbled` are inverted (`model/scoring_strategy.go:66-106`). Positions
  outside the four known ones get a 0.5 discount (`model/scoring_strategy.go:17-31`).
- Scores are append-only and idempotent per `(player_id, strategy_id, statistics_id)`
  (`db/init.sql:126-137`). Recalculation skips a score that already exists
  (`service/score.go:329-335`).
- Activation appends a row and immediately recalculates every player with the new strategy
  (`service/score.go:139-154`). The default strategy is 1 and is activated only when no activation
  exists (`service/score.go:22-24`, `service/score.go:159-169`).
- League coefficients are append-only rows with an effective timestamp
  (`db/init.sql:143-152`). They are applied on read, with `1.0` when no row exists
  (`service/score.go:265-275`). The exposed score is `percentile * coefficient`, then multiplied
  by the position discount (`dto/score.go:20-30`, `service/score.go:233-236`).
- Quote prices use a base of 1 credit plus `round(score * 100)` credits, floored at the base
  (`model/player_price.go:5-29`). Recalculation prices from
  `percentile * coefficient * positionDiscount` (`service/score.go:398`).

Invariants: at most one initial quote per player; a score is unique per statistics row and
strategy; percentile stays in `[0,1]`; a missing coefficient means `1.0`, never an error.

## P2P sessions

Owning package: `model` for transitions; `internal/service` for ownership; the repository for
locks and token movement.

- Statuses are `PENDING`, `COMPLETED`, `CANCELLED`, and `EXPIRED`
  (`model/p2p_session.go:18-23`), enforced by `chk_p2p_status` (`db/init.sql:320`).
- `IsLive` means `PENDING` only (`model/p2p_session.go:41-43`). `IsExpired` means `PENDING` and
  `now >= ExpiresAt` (`model/p2p_session.go:45-47`). `TotalAmount` is `Price * Quantity`
  (`model/p2p_session.go:49-51`).
- `Accept` checks, in order: expired (terminal row or pending row past expiry) returns
  `ErrP2PSessionExpired`; `COMPLETED` returns `ErrP2PSessionAlreadyTaken`; any other non-live
  status returns `ErrP2PSessionNotLive`; seller equals buyer returns `ErrP2PSessionSelfTrade`.
  Only then does it set `COMPLETED`, the buyer, and `CompletedAt`
  (`model/p2p_session.go:53-72`).
- `Cancel` requires `PENDING` and sets `CANCELLED` (`model/p2p_session.go:74-83`); the service
  checks the seller identity before calling the repository
  (`service/p2p.go:114-125`).
- `Expire` requires `PENDING` and sets `EXPIRED` (`model/p2p_session.go:85-94`).
- The service sets a five-minute window at creation (`service/p2p.go:73-82`). Acceptance passes
  the current time into `session.Accept` inside the transaction
  (`persistence/repository/p2p.go:138-145`).
- Creation reserves seller tokens before inserting the row
  (`persistence/repository/p2p.go:76-89`). Cancellation releases the reservation
  (`persistence/repository/p2p.go:342-352`); the expiry path attempts the same release on a
  best-effort basis (`persistence/repository/p2p.go:410-414`).
- Acceptance moves reserved tokens to the buyer, settles credits at `TotalAmount`, and records a
  trade with `P2PSessionID` (`persistence/repository/p2p.go:147-161`,
  `persistence/repository/p2p.go:264-275`).

Invariants: terminal statuses never transition again; only `PENDING` can become `COMPLETED`,
`CANCELLED`, or `EXPIRED`; a seller cannot accept their own offer; cancellation releases the
reserved quantity once, and expiry attempts it best-effort.

## Invariant summary

| Invariant | Owning package | Enforcement point |
| --- | --- | --- |
| Money is integer scaled by 10,000, never float | `model` | `model/credits.go:3-5` |
| Division truncates and guards zero | `model` | `model/credits.go:23-28` |
| Credits cross the wire as decimal strings | `dto`, `controller/dto` | `dto/credits.go:9-25`, `controller/dto/market.go:42-76` |
| Account balances are non-negative | `model`, `db` | `model/account.go:19-68`, `db/init.sql:199-200` |
| Each settlement group nets to zero | `persistence/dao` (check), repository writers | `persistence/dao/ledger_sql.go:57-62` |
| Initial grant is a paired entry | `persistence/repository` | `persistence/repository/registration.go:76-105` |
| Position quantities and cost basis are non-negative | `model`, `db` | `model/position.go:26-82`, `db/init.sql:213-215` |
| One position per user and player | `db` | `db/init.sql:212` |
| Execution never exceeds order quantity | `model`, `db` | `model/order.go:61-67`, `db/init.sql:238` |
| Only live orders match; no self-match | `model` | `model/matching.go:51-66` |
| Trade total equals price times quantity | `db` | `db/init.sql:264` |
| Trade parties are distinct | `db` | `db/init.sql:265` |
| One initial quote per player | `db` | `db/init.sql:304-306` |
| Score is unique per statistics row and strategy | `db` | `db/init.sql:136` |
| P2P transitions start from `PENDING` | `model`, `db` | `model/p2p_session.go:53-94`, `db/init.sql:320` |
| P2P seller cannot accept their own offer | `model` | `model/p2p_session.go:63-65` |
| P2P cancellation releases the reservation; expiry attempts it best-effort | `persistence/repository` | `persistence/repository/p2p.go:342-352`, `persistence/repository/p2p.go:410-414` |
| Overdue PENDING sessions are expired by a background job | `persistence/repository`, `service` | `persistence/repository/p2p.go:378-435`, `service/p2p_expiration.go:32-33` |
