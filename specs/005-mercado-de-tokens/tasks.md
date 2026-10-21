# Tasks: Player Token Market (Mercado de tokens)

**Input**: Design documents from `/specs/005-mercado-de-tokens/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/openapi.yaml](./contracts/openapi.yaml), [quickstart.md](./quickstart.md)

**Tests**: Included and mandatory. Constitution principle XIII requires unit, integration and e2e coverage per layer, and SC-004/SC-005 name specific scenarios. Each task below delivers its implementation **and** its tests in one commit, so the task ends green on its own.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel — different files, and every task it depends on is already done
- **[Story]**: The user story the task serves (US1..US7). Tasks in the two foundational phases carry no label because every story depends on them
- Every description carries the exact file paths it touches

## Organization

Phases are the five phases of [plan.md](./plan.md), not one phase per user story. The plan's phases are layered (domain → persistence → service → bootstrap → edge) because this feature is one bounded context whose stories share a single schema, a single matcher and a single transaction boundary; no story is independently deliverable before Phase 5. Story traceability is preserved by the `[Story]` labels and by the matrix at the end of this document.

## Rules that override the style of the code you will read

- **No comments and no docstrings.** Not inline (`//`), not package-level, not in production code, not in tests. The *what* is the name and the type; the *why* is already in `research.md` and `plan.md`. The only exception is a directive a tool reads (`//go:generate`, `//nolint`, build tags)
- **Principle XIV (preservation)**: no existing comment, string or test is modified. Where a task extends a file that already has tests, it adds a **new** test file rather than editing the existing one. The suites of features 002 and 003 are the regression gate and must stay green untouched
- **Copy the existing patterns.** DAO: struct `XxxSql` with field `Db *sql.DB`, constructor `NewXxxDao(db *sql.DB) *XxxSql`, `const query = ...` parameterized, Postgres errors mapped to `model` errors via `errors.AsType[*pq.Error]` (`23505` for a unique violation). DI through the per-layer containers (`dao/container.go`, `repository/container.go`, `service/container.go`, `controller/container.go`), assembled in `cmd/server/main.go`
- **No new dependency.** Standard library `net/http` and `database/sql` only; `testcontainers-go` for anything touching persistence
- Snake_case file names matching the main type; tests mirror the folder of what they test
- `Credits` is rendered as a decimal **only** in `internal/controller/dto`; every other layer carries scaled `int64`

---

## Phase 1 — Foundations

**Purpose**: The schema and the whole pure domain. Nothing in this phase touches a database, so it ends with the domain green and no container started.

- [X] T001 Add the six tables, the sequence and their indexes and check constraints to `backend/db/init.sql`, literally as written in `data-model.md` section 2: `user_accounts`, `user_positions`, `market_orders`, `market_trades`, `ledger_settlement_seq`, `ledger_entries`, `player_quotes`, including `idx_orders_book`, `idx_orders_user`, `uq_orders_market_maker_live`, `uq_positions_user_player`, `uq_quotes_player_initial`, `idx_quotes_player_latest`, `idx_trades_buyer`, `idx_trades_seller`, `idx_trades_player`, `idx_ledger_user`, `idx_ledger_settlement`, `idx_positions_user`
- [X] T002 Create the fixed-point `Credits` type with `CreditScale`, `CreditsFromUnits`, `Times`, `Plus`, `Minus`, `DividedBy` in `backend/internal/model/credits.go`, with boundary and truncation unit tests in `backend/internal/model/credits_test.go`
- [X] T003 [P] Create `OrderSide`, `TimeInForce`, `OrderStatus`, `Order` and its state machine (`RemainingQuantity`, `IsLive`, `Execute`, `Cancel`) plus the sentinels `ErrOrderNotLive` and `ErrInvalidQuantity` in `backend/internal/model/order.go`, with table-driven tests covering every valid and every invalid transition in `backend/internal/model/order_test.go` (depends on T002)
- [X] T004 [P] Create `Account` with `Reserve`, `ReleaseReservation`, `SettleReservation`, `Credit` plus the sentinels `ErrInvalidAmount`, `ErrInsufficientFunds`, `ErrReservationTooSmall` in `backend/internal/model/account.go`, with equivalence-class and boundary tests in `backend/internal/model/account_test.go` (depends on T002)
- [X] T005 [P] Create `Trade`, `LedgerReason` and `LedgerEntry` with their field ordering by descending size for `govet fieldalignment` in `backend/internal/model/trade.go` (depends on T002)
- [X] T006 [P] Create `PlayerQuote` in `backend/internal/model/quote.go` (depends on T002)
- [X] T007 [P] Create `Position` with `TotalQuantity`, `AveragePrice`, `ReserveTokens`, `ReleaseReservedTokens`, `DeliverReservedTokens`, `Receive` and the sentinel `ErrInsufficientTokens` in `backend/internal/model/position.go`, with tests in `backend/internal/model/position_test.go` covering the proportional cost-basis release of FR-010 and the full-position branch that leaves no truncation residue (depends on T003, T004)
- [X] T008 [P] Create `Execution` and the pure `MatchOrder(incoming Order, book []Order) []Execution` in `backend/internal/model/matching.go`, with one test case per rule of `data-model.md` section 3.5 (eligible side, crossable price, price priority, time priority at equal price, self-crossing skipped, execution at the resting price, exhaustion) in `backend/internal/model/matching_test.go` (depends on T003)
- [X] T009 Create `PortfolioPosition`, `Portfolio` and the pure `BuildPortfolio` in `backend/internal/model/portfolio.go`, with tests in `backend/internal/model/portfolio_test.go` asserting the four-decimal exactness of SC-002 after mixed purchases at different prices, reserved tokens counted in the quantity owned, and the empty-position case (depends on T002, T004, T007)

**Checkpoint**: `go test ./internal/model/...` green with no container started.

---

## Phase 2 — Persistence

**Purpose**: The six DAOs, the player row lock and the two repositories that own the transaction boundary and the settlement. Every query is covered by an integration test with `testcontainers-go`; no mock stands in for the database.

- [X] T010 [P] Create `AccountSql` with `NewAccountDao`, insert, lock-for-update by user and balance update in `backend/internal/persistence/dao/account_sql.go`, with integration tests in `backend/internal/persistence/dao/account_sql_test.go`
- [X] T011 [P] Create `PositionSql` with `NewPositionDao`, upsert with `ON CONFLICT DO NOTHING`, lock-for-update by user and player, quantity and cost-basis update, and list by user in `backend/internal/persistence/dao/position_sql.go`, with integration tests in `backend/internal/persistence/dao/position_sql_test.go`
- [X] T012 [P] Create `OrderSql` with `NewOrderDao`, insert, get by id, update of executed quantity and status, the live opposite-side book read `FOR UPDATE` ordered by price then `created_at`, the caller's live orders with an optional player filter, and the lookup of the live market maker order by player and side in `backend/internal/persistence/dao/order_sql.go`, with integration tests in `backend/internal/persistence/dao/order_sql_test.go`
- [X] T013 [P] Create `TradeSql` with `NewTradeDao`, insert and the paginated read by participant newest first in `backend/internal/persistence/dao/trade_sql.go`, with integration tests in `backend/internal/persistence/dao/trade_sql_test.go`
- [X] T014 [P] Create `LedgerSql` with `NewLedgerDao`, `nextval` on `ledger_settlement_seq` and the signed entry insert in `backend/internal/persistence/dao/ledger_sql.go`, with integration tests in `backend/internal/persistence/dao/ledger_sql_test.go`
- [X] T015 [P] Create `QuoteSql` with `NewQuoteDao`, the initial-quote insert, the generic quote insert and the latest quote by player in `backend/internal/persistence/dao/quote_sql.go`, with integration tests in `backend/internal/persistence/dao/quote_sql_test.go`
- [X] T016 [P] Add the player row lock (`SELECT id FROM players WHERE id = $1 FOR UPDATE`) to `backend/internal/persistence/dao/player_sql.go` and its interface, with its integration test in the **new** file `backend/internal/persistence/dao/player_lock_test.go` so that `player_sql_test.go` is not touched
- [X] T017 [P] Add the schema guard integration tests in `backend/internal/persistence/dao/market_schema_test.go`: the three partial unique indexes (`uq_positions_user_player`, `uq_quotes_player_initial`, `uq_orders_market_maker_live`) and the check constraints forbidding negative balances, negative positions, a non-positive price or quantity, an inconsistent trade total and a trade with itself (depends on T001)
- [X] T018 Register the six new DAOs in `backend/internal/persistence/dao/container.go` (depends on T010–T016)
- [ ] T019 Create `MarketRepository` in `backend/internal/persistence/repository/market.go` with the transaction boundary: begin, lock the player row before anything else, load the caller's account or position under the lock, read the live opposite side of the book under the lock, commit and roll back; with integration tests in `backend/internal/persistence/repository/market_test.go` (depends on T018)
- [ ] T020 Add the settlement of one execution to `backend/internal/persistence/repository/market.go`: both order updates, both accounts, both positions, the trade row, the two ledger entries sharing one settlement group and the audit entries through the existing `dao.AuditDAO` with `entity_type` of `user_account`, `user_position` and `market_order`; extend `backend/internal/persistence/repository/market_test.go` with the zero-sum assertion `SELECT settlement_group FROM ledger_entries GROUP BY settlement_group HAVING SUM(amount) <> 0` returning no rows (depends on T019)
- [ ] T021 [P] Create `PortfolioRepository` in `backend/internal/persistence/repository/portfolio.go` with the read side: a user's account, their positions, the latest quote per player and the paginated trade history; with integration tests in `backend/internal/persistence/repository/portfolio_test.go` (depends on T018)
- [ ] T022 Register `MarketRepository` and `PortfolioRepository` in `backend/internal/persistence/repository/container.go` (depends on T020, T021)

**Checkpoint**: `go test ./internal/persistence/...` green, including the suites of features 002 and 003 untouched.

**Checkpoint reached.** `dao` green in 182s, `repository` green in 58s, no test of features 002 or 003 modified. Two notes for whoever runs this suite next:

- Run the persistence packages serially with `go test -p 1`. Each test starts its own Postgres container, which is the pattern features 002 and 003 already use; running both packages at once saturates the Docker daemon and starves one of them into a timeout that looks like a failure but is not. The whole suite is roughly four minutes serially.
- Two defects were found and fixed here, both in the new test files and neither in the schema or the queries: the bcrypt fixture of the `dao` seed helper was 62 characters against a `VARCHAR(60)` column, now reusing the 60-character value feature 002 already settled on; and the lock serialization test asserted `context.DeadlineExceeded` where `lib/pq` cancels the statement and reports `57014`, so it now asserts that the context deadline expired while waiting, which is the actual claim.

---

## Phase 3 — Placement and cancellation

**Purpose**: `MarketService`. Every path that ends without full execution must leave the reservation released; that is asserted, not assumed.

- [ ] T023 [US1] Create `MarketService` with the buy path of `PlaceOrder` in `backend/internal/service/market.go`: price defaulted to the player's active quote, `timeInForce` defaulted to `FOK` without a price and `GTC` with one, rejection of an unknown or inactive player, of a non-positive quantity and of a non-positive price, credit reservation of `quantity * price`, the call to `model.MatchOrder`, settlement of each execution at the resting price, the release of the difference when the price improved, and the `GTC` remainder left resting; with integration tests in `backend/internal/service/market_test.go` covering acceptance scenarios 1, 2, 6, 7, 8 and 9 of US1 (depends on T022)
- [ ] T024 [US2] Add the sell path of `PlaceOrder` to `backend/internal/service/market.go`: token reservation instead of credits, the available position required to cover the quantity, crossing against resting buy orders most expensive first and oldest first at equal price, cost basis delivered proportionally and credits paid to the seller; with tests in `backend/internal/service/market_test.go` covering the five acceptance scenarios of US2, including the second order over already reserved tokens and the sale that leaves the average purchase price unchanged (depends on T023)
- [ ] T025 [US1] Add the `IOC` and `FOK` remainder rules to `backend/internal/service/market.go` — `IOC` persists `CANCELLED` and releases the remainder's reservation, `FOK` rolls the whole transaction back and returns the liquidity error — with one test per time in force against the same book state in `backend/internal/service/market_test.go`, covering US1 scenarios 3, 4 and 5 and the zero-liquidity edge case (depends on T024)
- [ ] T026 [US3] Add `CancelOrder` to `backend/internal/service/market.go`: the player lock, the ownership check, the liveness check, the state transition and the release of the reservation covering the remaining quantity for both sides; with tests in `backend/internal/service/market_test.go` covering cancellation of a partially executed order, an order already final and a non-owner (depends on T025)
- [ ] T027 [US3] Add `ListOpenOrders` to `backend/internal/service/market.go`, returning only the caller's `PENDING` and `PARTIALLY_FILLED` orders with an optional player filter; with tests in `backend/internal/service/market_test.go` asserting that another user's orders are never returned (depends on T026)
- [ ] T028 [P] [US3] Add the reservation invariant assertions in `backend/internal/service/market_invariant_test.go`: after every non-executing path (rejection, `IOC` remainder, cancellation, `FOK` rollback) the sum of reserved balances equals what live orders reserve, and the sum of tokens per player is exactly 100; plus the remainder that crosses a later order (depends on T027)
- [ ] T029 [P] Add the concurrency tests in `backend/internal/service/market_concurrency_test.go`: concurrent goroutines competing for the last tokens of one player, and concurrent orders of one user competing for the same credits, asserting SC-001 (depends on T027)
- [ ] T030 Register `MarketService` in `backend/internal/service/container.go` (depends on T027)

**Checkpoint**: `go test ./internal/service/...` green; SC-001 and SC-005 covered.

---

## Phase 4 — Bootstrap and registration

**Purpose**: The market exists from the first boot without manual setup, and every user can trade. The suites of features 002 and 003 stay green untouched, which is why every new test here goes in a new file.

- [ ] T031 [P] [US7] Add `InitialUserBalance` read from `NSP_INITIAL_USER_BALANCE` with a default of 1000 whole credits to `backend/internal/configuration/configuration.go`, with its tests in the **new** file `backend/internal/configuration/configuration_balance_test.go` so that `configuration_test.go` is not touched
- [ ] T032 [US6] Create `MarketMakerService` with the idempotent bootstrap in `backend/internal/service/market_maker.go`: per player with no market position, the initial quote of 1 credit as `strategy = 'initial'` and `strategy_version = '1'`, the superuser position of 100 tokens held reserved, and one resting sell order for the unsold tokens at the active quote with `market_maker = TRUE`; with integration tests in `backend/internal/service/market_maker_test.go` covering US6 scenarios 1, 2 and 3, including a second run that duplicates nothing and a player whose tokens were partly sold (depends on T030)
- [ ] T033 [US6] Add the re-pricing entry point to `backend/internal/service/market_maker.go`: when a quote is recorded, the live market maker order of that player is cancelled and reposted at the new price for the quantity still available, and orders of regular users are never touched; with tests in `backend/internal/service/market_maker_test.go` covering US6 scenario 4 and FR-011 (depends on T032)
- [ ] T034 [US7] Extend `Register` and `EnsureSuperuser` in `backend/internal/service/auth.go` so that a new user gets an account with the configured initial balance, zero reserved, and an `INITIAL_GRANT` ledger settlement whose signed entries sum to zero; with tests in the **new** file `backend/internal/service/auth_account_test.go` so that `auth_test.go` and `auth_session_test.go` are not touched (depends on T031, T022). The user, the account and the ledger settlement are persisted in one transaction, reached by adding an **additive** `InsertTx(ctx, tx, user)` to the concrete `*UserSql` struct in `backend/internal/persistence/dao/user_sql.go`: the existing `Insert`, the existing `UserSql` interface in `repository/user.go` and the feature 002 mocks are not touched, and the consumer declares its own narrow interface. See finding 3
- [ ] T035 [US6] Call the market maker bootstrap after a catalog synchronization in `backend/internal/service/player_sync.go`, with tests in the **new** file `backend/internal/service/player_sync_market_test.go` asserting that a repeated synchronization issues no new tokens (depends on T032)
- [ ] T036 [US6] Register `MarketMakerService` in `backend/internal/service/container.go` and run the bootstrap at startup after the superuser exists, in `backend/cmd/server/main.go` (depends on T033, T034, T035)

**Checkpoint**: boot on an empty database, synchronize, restart; the quickstart queries for 100 tokens per player and a zero-sum ledger hold. Features 002 and 003 green with no test modified.

---

## Phase 5 — Edge and contract

**Purpose**: The HTTP surface, the published contract and the evaluation scenario. A route exists in three places or it is broken: `routes.go`, `backend/api/openapi.yaml` with its `x-access-level`, and the route table of the plan.

- [ ] T037 [US4] Create `PortfolioService` in `backend/internal/service/portfolio.go` resolving the account, the positions, the latest quotes and the players and delegating the arithmetic to `model.BuildPortfolio`, with the authorization rule that a regular user reads only their own portfolio and the superuser reads any; with integration tests in `backend/internal/service/portfolio_test.go` covering the five acceptance scenarios of US4 (depends on T022, T009)
- [ ] T038 [US5] Add the paginated transaction history to `backend/internal/service/portfolio.go`, one entry per execution newest first with the side from the requesting user's perspective and the same authorization rule, and register `PortfolioService` in `backend/internal/service/container.go`; with tests in `backend/internal/service/portfolio_test.go` covering the three acceptance scenarios of US5 (depends on T037)
- [ ] T039 [P] [US1] Create the order request and response DTOs with their `DesdeModelo` factories and the decimal rendering and parsing of `Credits` in `backend/internal/controller/dto/market.go`, matching the shapes of `plan.md` and `contracts/openapi.yaml` in camelCase; with tests in `backend/internal/controller/dto/market_test.go` asserting the four-decimal round trip (depends on T009)
- [ ] T040 [US4] Add the portfolio response DTO and the transaction page DTO mirroring the published `items`/`total`/`page`/`limit`/`totalPages` envelope to `backend/internal/controller/dto/market.go`, with tests in `backend/internal/controller/dto/market_test.go` (depends on T039)
- [ ] T041 [US1] Create `MarketController` with the `POST /orders/buy` and `POST /orders/sell` handlers in `backend/internal/controller/market.go`, taking the acting user from the request context and never from the body, and mapping domain errors to 400, 404, 409 and 422; with integration tests in `backend/internal/controller/market_test.go` (depends on T040, T030)
- [ ] T042 [US3] Add the `GET /orders` and `DELETE /orders/{id}` handlers to `backend/internal/controller/market.go`, mapping to 200, 403 and 409; with tests in `backend/internal/controller/market_test.go` (depends on T041)
- [ ] T043 [US4] Add the `GET /users/{id}/portfolio` handler to `backend/internal/controller/market.go`, with authorization tests for another user's portfolio in `backend/internal/controller/market_portfolio_test.go` (depends on T042, T038)
- [ ] T044 [US5] Add the `GET /users/{id}/transactions` handler with its `page` and `limit` parameters to `backend/internal/controller/market.go`, with authorization and pagination tests in `backend/internal/controller/market_portfolio_test.go` (depends on T043)
- [ ] T045 Register `MarketController` in `backend/internal/controller/container.go` (depends on T044)
- [ ] T046 Register the six routes with `AccessAuthenticated` in `backend/internal/server/routes.go`: `POST /orders/buy`, `POST /orders/sell`, `GET /orders`, `DELETE /orders/{id}`, `GET /users/{id}/portfolio`, `GET /users/{id}/transactions` (depends on T045)
- [ ] T047 Publish the six paths with their schemas, status codes and `x-access-level` in `backend/api/openapi.yaml`, taking the definitions from `specs/005-mercado-de-tokens/contracts/openapi.yaml`, until the existing `backend/internal/server/contract_test.go` passes with no route missing from the document, satisfying FR-016 and SC-003 (depends on T046)
- [ ] T048 [US6] Add the bootstrap idempotency end-to-end test in `backend/test/e2e/market_bootstrap_test.go`: boot on an empty database, synchronize, assert one initial quote per player, exactly 100 tokens per player held by the superuser and one resting market maker sell order; then boot again and assert nothing was duplicated (depends on T047)
- [ ] T049 Add the evaluation scenario of section 8.3 as an end-to-end test in `backend/test/e2e/market_test.go`: four users created, each buying tokens of five players, the quotes moved, and the valuation of every position reported, with the zero-sum ledger and the 100-tokens-per-player assertions run after the whole scenario; this is SC-004 (depends on T048)
- [ ] T050 Run the full validation of `specs/005-mercado-de-tokens/quickstart.md` by hand and `cd backend && go test ./... && golangci-lint run` until both are clean, which is SC-006 and SC-007 (depends on T049)

**Checkpoint**: every success criterion of the spec is covered by an automated check.

---

## Dependencies & Execution Order

### Phase dependencies

Strictly sequential. Each phase ends green before the next begins.

```
Phase 1 (T001-T009) -> Phase 2 (T010-T022) -> Phase 3 (T023-T030) -> Phase 4 (T031-T036) -> Phase 5 (T037-T050)
```

The one exception is T031 (`InitialUserBalance` in the configuration), which depends on nothing in Phases 1 to 3 and can be done at any time.

### Critical path

`T001 -> T002 -> T003 -> T007 -> T009` then `T018 -> T019 -> T020 -> T022` then `T023 -> T024 -> T025 -> T026 -> T027 -> T030` then `T032 -> T033 -> T036` then `T037 -> T038 -> ... -> T050`. The single-file chains inside `service/market.go` (T023-T027) and `controller/market.go` (T041-T044) are the longest unavoidable serial stretches.

### Parallel opportunities

| Group | Tasks | Why they do not collide |
|---|---|---|
| Domain types after `Credits` | T003, T004, T005, T006 | Four distinct files in `internal/model`; each defines its own sentinels |
| Domain types after the sentinels | T007, T008 | `position.go` and `matching.go` are distinct files |
| The whole DAO layer | T010, T011, T012, T013, T014, T015, T016, T017 | Eight distinct files in `internal/persistence/dao`, each with its own test file; none of them imports another |
| Repository read side | T021 alongside T019 and T020 | `portfolio.go` is a separate file from `market.go` |
| Service test suites | T028, T029 | Two new test files, both read-only against the service built in T023-T027 |
| Configuration | T031 | Independent of every other task in the feature |
| Order DTOs | T039 alongside T037 | `controller/dto/market.go` is independent of `service/portfolio.go` |

Tasks that touch the same file are never parallel: `init.sql` (T001), `dao/container.go` (T018), `repository/market.go` (T019, T020), `repository/container.go` (T022), `service/market.go` (T023-T027), `service/market_maker.go` (T032, T033), `service/container.go` (T030, T036, T038), `controller/dto/market.go` (T039, T040), `controller/market.go` (T041-T044), `routes.go` (T046), `openapi.yaml` (T047).

### Parallel example: the DAO layer

```bash
# After T001 and the whole of Phase 1, eight independent tasks:
Task: "AccountSql in backend/internal/persistence/dao/account_sql.go"
Task: "PositionSql in backend/internal/persistence/dao/position_sql.go"
Task: "OrderSql in backend/internal/persistence/dao/order_sql.go"
Task: "TradeSql in backend/internal/persistence/dao/trade_sql.go"
Task: "LedgerSql in backend/internal/persistence/dao/ledger_sql.go"
Task: "QuoteSql in backend/internal/persistence/dao/quote_sql.go"
Task: "Player row lock in backend/internal/persistence/dao/player_sql.go"
Task: "Schema guards in backend/internal/persistence/dao/market_schema_test.go"
```

---

## Story traceability

A story is demonstrable only once its row is complete. Every row needs Phases 1 and 2 first.

| Story | Priority | Tasks | Demonstrable after |
|---|---|---|---|
| US1 — Buy player tokens | P1 | T023, T025, T039, T041 | T047 |
| US2 — Sell player tokens | P1 | T024, T039, T041 | T047 |
| US3 — Inspect and cancel open orders | P1 | T026, T027, T028, T042 | T047 |
| US4 — View portfolio and performance | P2 | T009, T021, T037, T040, T043 | T047 |
| US5 — View transaction history | P2 | T013, T038, T040, T044 | T047 |
| US6 — Market bootstrap at moment zero | P2 | T032, T033, T035, T036, T048 | T048 |
| US7 — Registration with initial balance | P3 | T031, T034 | T036 |

### Independent test per story

- **US1**: register, confirm 1000 credits, buy 10 tokens of a player quoted at 1 credit; the order is `FILLED`, the balance is 990, the position is 10, the market maker drops to 90, and an audit record and a trade exist
- **US2**: holding 10 tokens, sell 4 without a price; the position drops to 6, the balance rises by `4 * quote`, the market maker rises by 4, and the average purchase price of the remaining 6 is unchanged
- **US3**: place a `GTC` buy that cannot cross, see it in `GET /orders` with its remaining quantity, cancel it, and watch the whole reservation return to the available balance
- **US4**: buy 10 at 1 credit and 10 at 2, move the quote to 2, request the portfolio: quantity 20, average price 1.5, valuation 40, result 10
- **US5**: execute one purchase and one sale, then list the transactions: both appear with side, player, quantity, unit price, total and timestamp
- **US6**: boot on an empty database, synchronize, assert one initial quote, 100 superuser tokens and one resting offer per player; boot again and assert nothing changed
- **US7**: register through `POST /auth/register` and confirm the available balance equals `NSP_INITIAL_USER_BALANCE`

---

## Implementation Strategy

### Walking skeleton first

The smallest increment that is worth demonstrating is **US1 end to end**, and it needs the whole of Phases 1 and 2 plus T023, T025, T030, T032, T036, T039, T041, T045, T046, T047: a user buying tokens at the active quote from a market maker that bootstrapped itself. Every other story is an increment on top of that spine.

### Incremental delivery

1. Phase 1 → the domain is green with no database
2. Phase 2 → every query and every schema guard is proven against a real PostgreSQL
3. Phase 3 → buying, selling and cancelling work at the service layer, with the concurrency guarantees asserted
4. Phase 4 → the market exists from the first boot and every new user can trade
5. Phase 5 → the HTTP contract, the published document and the evaluation scenario

### Notes

- One task, one commit, green before the next
- `[P]` means different files and satisfied dependencies, nothing else
- When a task extends a file that already has tests, its tests go in a new file: principle XIV forbids touching the existing ones
- No comments, no docstrings, anywhere

---

## Findings

Three things in the design artifacts needed a decision before the tasks that depend on them. All three are now resolved; the resolution is recorded here and in the artifact that was wrong.

1. **RESOLVED — `research.md` contradicted `data-model.md` on the credit column type.** Decision 5 of `research.md` said "The database column stays `NUMERIC(15, 4)`", while `data-model.md` section 1 says `BIGINT` in scaled units and justifies that choice explicitly. `research.md` was the stale one and has been corrected to `BIGINT` with a reference to section 1 of `data-model.md`. `BIGINT` everywhere. Affects T001 and the whole DAO layer.

2. **RESOLVED — `service/portfolio.go` was in the source layout but in no phase.** The Phase 5 description in `plan.md` now opens with `PortfolioService` over the read-side repository, which is where T037 and T038 already placed it. No task changed.

3. **RESOLVED — the account at registration is atomic with the user, through an additive method.** FR-014 as written scopes atomicity to order submission, execution and cancellation, so registration is not strictly covered; FR-013 requires only that the `INITIAL_GRANT` entries sum to zero. The call is nonetheless to make registration atomic, because a user inserted without an account is a real hole and the repository already has the pattern for it (`repository/player.go` opens the transaction with `BeginTx` and passes the `*sql.Tx` into the DAOs).

   **The method is additive, and this is the part T034 must respect.** A new `InsertTx(ctx context.Context, tx *sql.Tx, user model.User) (model.User, error)` is added to the concrete `*UserSql` struct in `backend/internal/persistence/dao/user_sql.go`. The existing `Insert` method, the existing `UserSql` interface in `backend/internal/persistence/repository/user.go`, and every mock of it in the feature 002 suite are left exactly as they are. The consumer of the new method declares its own narrow interface on its own side, which is the convention the comment at the top of `repository/user.go` already states. Principle XIV therefore holds: no existing test is touched.

   Source layout delta: `backend/internal/persistence/dao/user_sql.go` becomes `[mod]`, additive only. `repository/user.go` is **not** modified.
