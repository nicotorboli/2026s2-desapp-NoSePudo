# Review: p2p-token-session (calibration baseline)

- Branch: `f43cf73` (merged P2P feature) → base `9660724`
- Diff: 33 files, +4218 / -33 (over the 600-line gate; reviewed area by area on purpose)
- Date: 2026-11-16
- Verdict: **block** — one money defect

## P0 — fix before merge

### 1. P2P price boundary drops the credit scale by 4 orders of magnitude

- Locations: `backend/internal/controller/p2p.go:51` and `backend/internal/controller/dto/p2p.go` (response),
  contract `backend/api/openapi.yaml:1859` and `1900-1907`.
- Evidence: `model.Credits` is an int64 scaled by `CreditScale = 10000`
  (`backend/internal/model/credits.go:3-5`). Market orders parse a decimal string into that scale
  (`controller/dto/market.go:70`). The P2P controller does `Price: model.Credits(request.PricePerToken)`
  from a raw integer, and the response echoes `int64(session.Price)` as `pricePerToken` / `totalAmount`.
  The contract declares plain `integer/int64` with no unit note, and the UI renders the value as
  "créditos" (`P2PCreatePage.tsx:157,241`, `P2PSessionPage.tsx:333`) while defaulting the input to
  `10000` (`P2PCreatePage.tsx:22`).
- Execution path: `POST /p2p/sessions {quantity: 5, pricePerToken: 3}` → `Price = 3` scaled units
  → accepting buyer is debited 15 scaled units (0.0015 credits) while the page shows 15 credits.
  `chk_trades_total_consistent` passes because both sides use the same unscaled value.
- Fix: pick one unit at the boundary. Either `model.CreditsFromUnits(pricePerToken)` on input and
  divide on output, or move the contract to a decimal string like market orders and convert in the
  DTO. Add the unit to the OpenAPI description either way.
- Confidence: high
- Area: contract + data (found independently by both reviewers)

## P1 — should fix

### 1. Expiry job discards every error inside the transaction

- `backend/internal/persistence/repository/p2p.go:400-431`: `posErr`, the result of
  `ReleaseReservedTokens`, the result of `UpdateStatus`, and `tx.Commit()` are all assigned to `_`.
- Two outcomes: a session becomes `EXPIRED` while `reserved_quantity` still holds the tokens
  (seller loses them permanently), or tokens are released while the row stays `PENDING`
  (offer appears live, then `Accept` fails with an unmapped error).
- Fix: `defer tx.Rollback()`, check each write, and return or collect a per-session error so
  `service/p2p_expiration.go` can log it.
- Confidence: high on the code path; trigger is a DB error, untested.

### 2. Position and account locks are taken in role order, not id order

- `repository/p2p.go:189,202` locks seller position then buyer position; `:239,252` locks buyer
  account then seller account. Two users accepting each other's offers lock the same rows in
  opposite order → PostgreSQL deadlock, unmapped → 500.
- Fix: lock the two rows in ascending `user_id` order before mutating.
- Confidence: high on the inversion; not reproduced.

### 3. Unknown player maps to 500 instead of the documented 404

- `controller/p2p.go:124` → `service/p2p.go:65` → `repository.ErrNotFound`
  (`persistence/repository/player.go:75`), but `p2pStatusFor` matches `service.ErrNotFound`,
  a different value. Falls through to `default` → 500, while `openapi.yaml:1092` documents 404.
- Same hole for `repository.ErrPlayerInactive` at `repository/p2p.go:73`.
- Fix: translate repository sentinels in the service, as `service/market.go:545` already does.
- Confidence: high.

### 4. Non-UUID token reaches Postgres and returns 500

- `Get`/`Accept`/`Cancel` only check `token != ""` (`controller/p2p.go:65,87,108`); the DAO compares
  against a `UUID` column with `lib/pq`, so `GET /p2p/sessions/abc` raises 22P02 and is unmapped.
- Fix: validate the token shape in the controller and return 400/404 before the DAO.
- Confidence: high on behavior, medium on the exact SQLSTATE (not observed).

### 5. Portfolio empty state renders during load and after failure

- `P2PCreatePage.tsx:181`: only `positions.length === 0` is checked; `isLoading` is set but never
  read, and the catch leaves `positions` empty. A seller who owns tokens is told they own none for
  the whole request; on failure the error alert and the empty card render together.
- Fix: separate loading / empty / error branches.
- Confidence: high.

### 6. COMPLETED card tells the actual buyer the trade was taken by someone else

- `P2PSessionPage.tsx:246-254` ignores `isSeller` and `session.buyer`; the token stays in the URL,
  so reloading after purchase renders "completada por otro usuario" to the buyer.
- Fix: branch on `session.buyer.id === user.id` and on `isSeller`, keep the third-party text only
  for uninvolved viewers.
- Confidence: high.

## P2 — backlog

- `db/init.sql:308-331` — P2P invariants only in Go: no buyer ≠ seller check, no status/buyer
  consistency check, no unique index on `market_trades.p2p_session_id`.
- `db/init.sql:326` — `idx_p2p_sessions_token` duplicates the implicit unique index on `token`.
- `dao/p2p_session_sql.go:129` — expiry batch has no `SKIP LOCKED`, contrary to
  `plans/p2p-token-session/modifications.md`.
- `repository/p2p.go:95,167,359,419` — audit insert errors discarded.
- `repository/p2p.go:138` — `Accept` retry after a lost response returns 409 to the buyer who
  already bought.
- `openapi.yaml:1888,1915` — `nullable: true` is not valid OpenAPI 3.1; `buyer` is documented
  non-nullable but the DTO emits null.
- `service/p2p_expiration.go:38` — `Start` twice panics on closing a closed channel.
- `openapi.yaml:1866` — `P2PUserSummary` exposes email to any authenticated token holder.
- `P2PCreatePage.tsx:43` — effect refetches portfolio on every select change.
- `P2PCreatePage.tsx:113` — clipboard call neither awaited nor caught; timeout not cleaned up.
- `P2PCreatePage.tsx:55` — `__title` / `__description` classes have no CSS rules.
- Six Spanish comments added in `P2PSessionPage.tsx` (violates the no-new-comments rule).

## Verified OK

- Contract: 32 operations and 32 `x-access-level` entries; P2P routes match
  `routes.go` verbatim; all four are `authenticated`; no `AccessUndeclared`.
- Data: no JSON tags in `model/`; all money is int64/Credits, no float; `Create`, `Accept`,
  `Cancel` each open a transaction with rollback; `FOR UPDATE` on session/position/account;
  `p2p_sessions` in `expectedTables`; FKs and positive-value checks present.
- Authorization: cancellation checks the seller against the loaded session in the service.
- Frontend: axios only in `src/api/`; four API functions match four operations; countdown cleaned
  up on unmount; accept disabled while submitting; BEM classes resolve in `P2PSessionPage.css`.
- Prohibitions: no test, dependency, or lockfile changes in the code paths reviewed.

## Gaps

- 33 files / 4218 lines: the scope gate in `docs/review.md` correctly rejects this diff. This run
  split it by area as a calibration exercise.
- No builds, typecheck, lint, or tests were run (reviewers are read-only by design).
- No P2P tests exist at all, so the expiry, deadlock, and price-scale paths have no automated
  coverage.
- UUID error text and deadlock timing inferred from code, not observed.
- `specs/006-transaccion-p2p-tokens/**` was authored by the branch itself; scope note, not a
  finding.

## Files to open

1. `backend/internal/controller/p2p.go` — price scale (P0), token validation, error mapping
2. `backend/internal/controller/dto/p2p.go` — response units, `buyer` nullability
3. `backend/api/openapi.yaml:1848-1866` and `1896-1910` — request/response schemas
4. `backend/internal/persistence/repository/p2p.go:138-262, 380-435` — locking and expiry
5. `frontend/src/pages/P2PCreatePage/P2PCreatePage.tsx:30-60, 175-185`
6. `frontend/src/pages/P2PSessionPage/P2PSessionPage.tsx:240-260`
