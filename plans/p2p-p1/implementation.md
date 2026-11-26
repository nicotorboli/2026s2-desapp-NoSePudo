# P2P P1 remediation — implementation

Date: 2026-11-25
Source: `plans/p2p-token-session/review.md` (P1 items 1–6)
Working tree: current worktree, no branch, no commit.

## Modified files

### Backend

1. `backend/internal/persistence/repository/p2p.go`
   - `ExpirePending` now runs one transaction per session and collects failures with
     `errors.Join`; a new `expireSession` helper defers `tx.Rollback`, checks
     `ReleaseReservedTokens`, `PositionSql.Update`, `P2PSessionSql.UpdateStatus`, the `EXPIRE`
     audit insert, and `tx.Commit`.
   - A missing seller position (`model.ErrPositionNotFound`) is treated as "nothing to release";
     any other error aborts the session transaction. A session is never marked `EXPIRED` with the
     reservation left in place, nor are tokens released while the row stays `PENDING`.
   - `transferTokens` locks the seller and buyer positions in ascending `user_id` order.
     `lockOrCreateBuyerPosition` behavior is preserved; when the buyer position is created it is
     resolved before the seller lock (buyer id lower path), keeping the ascending order.
   - `settleBalances` locks the buyer and seller accounts in ascending `user_id` order before
     mutating either balance. Amounts and authorization are unchanged.

2. `backend/internal/service/p2p.go`
   - Added `translateP2PPlayerError`, mapping `repository.ErrNotFound` to `service.ErrNotFound`
     and `repository.ErrPlayerInactive` to `service.ErrPlayerInactive` (same pattern as
     `translateMarketError` in `service/market.go`).
   - `Create` and `Get` pass player lookup failures through that translation, so `p2pStatusFor`
     produces the documented Spanish 404 instead of a 500.
   - Follow-up: the result of `s.repo.Create` in `Create` is also passed through
     `translateP2PPlayerError`. `P2PRepository.Create` re-checks the player row and returns
     `repository.ErrNotFound` / `repository.ErrPlayerInactive`; without this, those paths still
     produced a 500 instead of the documented 404.

3. `backend/internal/controller/p2p.go`
   - Added `p2pTokenFromPath`: rejects an empty token with the existing message and parses the
     token with `github.com/google/uuid`, returning `400 {"error":"token inválido"}` before the DAO.
   - `Get`, `Accept`, and `Cancel` use the helper instead of checking only `token != ""`.

4. `backend/api/openapi.yaml`
   - Added a `400` response (`Token inválido`) to `GET /p2p/sessions/{token}`,
     `POST /p2p/sessions/{token}/accept`, and `POST /p2p/sessions/{token}/cancel`.
   - No `x-access-level`, route, or schema change.

### Frontend

5. `frontend/src/pages/P2PCreatePage/P2PCreatePage.tsx`
   - `loadPositions` no longer depends on `selectedPlayerId`; it uses the functional update
     `setSelectedPlayerId((previous) => (previous === '' ? available[0].playerId : previous))`,
     so selecting a player neither refetches the portfolio nor replaces the form.
   - `isLoading` is initialized from `isAuthenticated` and now rendered.
   - Added the three branches from `docs/designs/2026-11-25-p2p-create-states.md`:
     loading (`role="status"`, `aria-live="polite"`, "Cargando tus posiciones...") while the
     portfolio is loading with no error, empty only after a successful load with no positions,
     otherwise the form. A failed load keeps the error alert and does not render the empty card.
   - The error alert now carries `role="alert"`.

6. `frontend/src/pages/P2PCreatePage/P2PCreatePage.css`
   - Added `.p2p-create__loading` next to `.p2p-create__empty`, using the page's secondary text
     colour (`#64748b`).

7. `frontend/src/pages/P2PSessionPage/P2PSessionPage.tsx`
   - The `COMPLETED` card branches on `session.buyer?.id === user?.id` and on `isSeller`.
     The buyer sees "Compra Completada" / "Ya completaste esta compra…"; the seller sees
     "Oferta Completada" / "Tu oferta fue aceptada…"; the previous
     "completada por otro usuario" text is kept only for uninvolved viewers.

### Plan

8. `plans/p2p-p1/implementation.md` — this file.

## Verification

Run from the repository root unless noted.

| Command | Result |
| --- | --- |
| `cd backend && go build ./...` | pass (no output) |
| `cd backend && go test ./internal/server/` | pass — `ok .../internal/server 0.012s` |
| `cd backend && make test` | pass — 12 unit packages, 240 tests, 0 failed, 0 skipped |
| `./scripts/pre-commit.sh` | pass — gofmt, `go vet`, golangci-lint v2.14.0, 0 issues |
| `cd frontend && npm run typecheck` | pass |
| `cd frontend && npm run lint` | pass |
| `cd frontend && npm run knip` | pass |
| `cd frontend && npm run test` | pass — 16 test files, 58 tests |

### Follow-up re-run (extra `repository.Create` translation)

| Command | Result |
| --- | --- |
| `cd backend && go build ./...` | pass (no output) |
| `cd backend && go test ./internal/server/` | pass — `ok .../internal/server (cached)` |
| `cd backend && make test` | pass — 12 unit packages, 240 tests, 0 failed, 0 skipped |
| `./scripts/pre-commit.sh` | pass — gofmt, `go vet`, golangci-lint v2.14.0, 0 issues |

No Docker-related command was run. No test, CI, `specs/`, `.specify/`, or `db/init.sql` change was
made. No dependency was added.

The working tree already contained the Task 0 documentation edits (`plans/agent-workflow/*`,
`plans/test-runner/implementation.md`, deletion of `plans/agent-workflow/handoff.md`); they were
left untouched.

## Open points / human decision

- No P2P automated tests exist. The expiry error paths and the lock ordering are therefore not
  exercised by the suite; adding tests is out of scope for this feature task.
- `docs/errors.md` still lists the P2P 404 and malformed-token mappings under "Known gaps". The code
  is now fixed; the document was intentionally not edited to keep this change scoped to the six P1
  items. Updating it is a documentation follow-up.
- The malformed-token 400 is documented inline as `Token inválido`; it is not reusing the
  `ValidationFailed` component because that component describes malformed request bodies.
