# P2P P2 remediation — implementation

Date: 2026-11-25
Source: `plans/p2p-token-session/review.md` (P2 backlog), `/home/cristian/.opencode/plan/p1-p2-remediation.md` §3
Working tree: current worktree, no branch, no commit. Builds on the uncommitted P1 changes.

## Modified files

### Backend

1. `backend/db/init.sql`
   - `p2p_sessions` gains two table-level checks:
     `chk_p2p_distinct_parties` (`buyer_id IS NULL OR buyer_id <> seller_id`) and
     `chk_p2p_buyer_consistency` (`(status = 'COMPLETED') = (buyer_id IS NOT NULL)`).
     `CANCELLED` / `EXPIRED` rows keep `buyer_id` NULL and satisfy the second check.
   - `idx_p2p_sessions_token` removed; `DROP INDEX IF EXISTS idx_p2p_sessions_token;` replaces it
     so an existing volume loses the redundant index on a manual re-run. The `token` column's
     `UNIQUE` constraint (implicit `p2p_sessions_token_key`) is untouched.
   - `idx_trades_p2p` is now `CREATE UNIQUE INDEX`; a `DROP INDEX IF EXISTS idx_trades_p2p;`
     precedes it so a manually re-run script replaces the previous non-unique partial index.
   - Reset-dependent statements (see "Schema reset" below): both `CHECK` constraints, the token
     index drop, and the trade-index drop/recreate.

2. `backend/internal/persistence/dao/p2p_session_sql.go`
   - `ListPendingExpired` now takes `tx *sql.Tx`, rejects a nil transaction with
     `errLockNeedsTransaction` (same guard as `LockByToken`), and runs
     `ORDER BY s.expires_at ASC FOR UPDATE OF s SKIP LOCKED` on that transaction.
     `FOR UPDATE OF s` locks only `p2p_sessions`, not the joined `users` rows.

3. `backend/internal/persistence/dao/trade_sql.go`
   - New `GetByP2PSessionID(ctx, tx, sessionID) (model.Trade, error)`: single-row select on
     `market_trades.p2p_session_id`, scanning the same columns/optional fields as
     `ListByParticipant`. Uses `querierFor` so it works with a transaction or the pool.

4. `backend/internal/persistence/repository/p2p.go`
   - `Create`, `Accept`, `Cancel`: `InsertAuditLog` failures are no longer discarded; they are
     wrapped (`registrar auditoría p2p create/accept/cancel`) and returned, rolling back the
     transaction. `ExpirePending` was left as P1 wrote it (it already checks the `EXPIRE` audit).
   - `Accept`: after `LockByToken`, if the session is `COMPLETED` and `BuyerID` equals the caller,
     it returns the existing trade through `GetByP2PSessionID` instead of `409`. Any other buyer,
     or any other status, follows the previous path (`Accept` then the existing error mapping).
     The returned trade goes through the unchanged `P2PTradeResponseDesdeModelo` in the controller.
   - `ExpirePending`: one batch transaction:
     `BeginTx` → `ListPendingExpired(ctx, tx, now)` (rows locked `SKIP LOCKED`) → for each session
     `SAVEPOINT p2p_expire_session` → `expireSession(ctx, tx, token, now)` →
     `RELEASE SAVEPOINT`; on any per-session error `ROLLBACK TO SAVEPOINT`, collect the error, and
     continue with the next session. The final `Commit` error is joined with the per-session errors.
     A savepoint open/rollback/release failure aborts the batch and returns everything collected so
     far (the deferred `Rollback` clears the transaction).
   - `expireSession` no longer owns a transaction: it receives `*sql.Tx`, keeps the `IsLive`
     re-check via `LockByToken`, releases the seller reservation when the position exists, updates
     the session status, and writes the checked `EXPIRE` audit row. `ErrPositionNotFound` still
     means "nothing to release"; any other error aborts only that session's savepoint.

5. `backend/internal/service/p2p_expiration.go`
   - `Start` checks `started` under `mu` and returns early, so a second call neither starts a second
     loop nor closes `done` twice. The context/cancel pair is only created on the first call.

6. `backend/api/openapi.yaml`
   - `P2PSessionResponse.buyer`: `nullable: true` replaced with
     `anyOf: [P2PUserSummary, {type: 'null'}]`, matching the DTO (`*P2PUserSummary`, always emitted,
     null until accepted).
   - `P2PSessionResponse.completedAt`: `nullable: true` replaced with
     `anyOf: [{type: string, format: date-time}, {type: 'null'}]`. The DTO omits the field while
     pending (`omitempty`) and emits a timestamp once terminal; both are valid under `anyOf`.
   - No route, operation, response code, or `x-access-level` change; the P1 `400` entries remain.

### Frontend

7. `frontend/src/pages/P2PCreatePage/P2PCreatePage.tsx`
   - Added `useRef` and `copyTimeoutRef`.
   - `handleCopyLink` is async, awaits `navigator.clipboard.writeText`, clears a previous pending
     reset timer, schedules the 3 s `copied` reset in `copyTimeoutRef`, and on rejection shows the
     existing error alert (`No se pudo copiar el enlace. Copialo manualmente.`). A successful copy
     clears the alert.
   - A mount-scoped effect clears `copyTimeoutRef` on unmount.
   - `loadPositions` was verified, not changed (P1 already made it stable with
     `useCallback(..., [user])` and a functional `setSelectedPlayerId`).

8. `frontend/src/pages/P2PCreatePage/P2PCreatePage.css`
   - Added `.p2p-create__title` (1.5rem, weight 700, `#1e293b`) and `.p2p-create__description`
     (0.9rem, line-height 1.5, `#64748b`), consistent with `.p2p-create__heading` and the page
     palette. No inline styles were introduced.

9. `frontend/src/pages/P2PSessionPage/P2PSessionPage.tsx`
   - Deleted the six approved `//` lines (countdown, cases 1–5). No other line was touched; the
     file now contains no `//` comments.

### Plan

10. `plans/p2p-p2/implementation.md` — this file.

## Explicitly not changed

- `P2PUserSummary` / session page keep `email` (human decision).
- `docs/errors.md` and other docs were not edited (the "Known gaps" P2P entries are stale, see
  follow-ups).
- P1's `transferTokens` / `settleBalances` lock ordering and the controller token validation were
  left as P1 wrote them.
- No test, dependency, lockfile, CI, `specs/`, or `.specify/` change.

## Schema reset

`backend/db/init.sql` only runs when the Postgres volume is empty. The following parts of this
change are **not** applied to an existing dev volume and need
`docker compose down -v && docker compose up -d` (run by the human, not by this task):

```sql
ALTER TABLE p2p_sessions
    ADD CONSTRAINT chk_p2p_distinct_parties CHECK (buyer_id IS NULL OR buyer_id <> seller_id),
    ADD CONSTRAINT chk_p2p_buyer_consistency CHECK ((status = 'COMPLETED') = (buyer_id IS NOT NULL));

DROP INDEX IF EXISTS idx_p2p_sessions_token;

DROP INDEX IF EXISTS idx_trades_p2p;
CREATE UNIQUE INDEX idx_trades_p2p ON market_trades (p2p_session_id)
    WHERE p2p_session_id IS NOT NULL;
```

The shipped script expresses the constraints inside `CREATE TABLE IF NOT EXISTS p2p_sessions` and
the index changes as `DROP INDEX IF EXISTS` + `CREATE UNIQUE INDEX IF NOT EXISTS`. On an existing
volume a manual re-run of `init.sql` would apply the index changes but not the table constraints,
because `CREATE TABLE IF NOT EXISTS` is a no-op there; a volume reset is the supported path. Before
resetting, the pending P2P data check should still confirm there are no `p2p_sessions` rows and no
`market_trades.p2p_session_id` values (unchanged requirement from the P1/P2 plan).

## Verification

Run from the repository root unless noted. No Docker-related command was run.

| Command | Result |
| --- | --- |
| `cd backend && go build ./...` | pass (no output) |
| `cd backend && go test ./internal/server/` | pass — `ok .../internal/server 0.013s` |
| `cd backend && make test` | pass — 12 packages, 240 tests, 0 failed, 0 skipped |
| `./scripts/pre-commit.sh` | pass — gofmt, `go vet`, golangci-lint v2.14.0, `0 issues` |
| `cd frontend && npm run typecheck` | pass |
| `cd frontend && npm run lint` | pass |
| `cd frontend && npm run knip` | pass |
| `cd frontend && npm run test` | pass — 16 files, 58 tests |
| `python3 -c "yaml.safe_load(...)"` on `openapi.yaml` | pass — both schemas parse as `anyOf` with a `null` branch |

## Human follow-ups

1. **Volume reset** — after reviewing the `init.sql` diff, run the P2P data check and then
   `docker compose down -v && docker compose up -d`. The DB-backed suites
   (`make test-integration`, `make test-e2e`, `make test-all`, coverage targets) were not run here
   and should be re-run against the reset schema before merge.
2. **Stale docs** — `docs/errors.md` still lists the P2P 404 / malformed-token cases as "Known
   gaps" even though P1 fixed them, and it predicates `ListPendingExpired` on `dao.Db`. Docs were
   out of scope; update separately.
3. **Expiry batch under load** — the single batch transaction holds the `SKIP LOCKED` locks until
   commit. There is no P2P automated coverage, so the savepoint isolation and the batch lock window
   are verified by inspection only; a repository/integration test is a separate testing task.
4. **Accept retry** — the idempotent path relies on `market_trades.p2p_session_id` being unique
   (now enforced by the schema) and on the trade row existing for a `COMPLETED` session. If that
   invariant is ever violated, the lookup error surfaces as `500`; no new mapping was added.
