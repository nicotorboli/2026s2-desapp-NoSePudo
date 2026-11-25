# Handoff — agent workflow

Branch: `rf/reestructurar-agents-workflow` · Last update: 2026-11-25 · Language: English

Read this first, then `AGENTS.md`, then the file for the task you are picking up.

## 1. What exists

| Artifact | State | Notes |
| --- | --- | --- |
| `docs/review.md` | committed | Review checklist with mechanical checks |
| `.opencode/agents/reviewer.md` | committed | Read-only reviewer subagent |
| `.opencode/skills/review-changes/**` | committed | Skill ID `review-changes` |
| `AGENTS.md` | committed, then edited | Language, spec-kit, Agent workflow; two design-skill rows added on 2026-11-25 |
| `plans/p2p-token-session/review.md` | committed | Calibration review: fixed P0 plus open P1/P2 backlog |
| `plans/p2p-price-scale-fix/**` | committed | P0 record and implementation |
| P2P price fix | committed | Backend, OpenAPI, frontend, and three test files |
| Backend test runner | committed (`e540a9f`) | `backend/Makefile` + `backend/scripts/*`; see `plans/test-runner/implementation.md` |
| Slice 2 docs and ADRs | **uncommitted** | `docs/architecture-backend.md`, `docs/domain.md`, `docs/errors.md`, `docs/ui.md`, `docs/adr/0001…0006` |
| Design skills | **uncommitted** | `design-backend` and `design-frontend`, both discovered by OpenCode |
| Calibration designs | **uncommitted** | `docs/designs/2026-11-25-p2p-price-units.md`, `docs/designs/2026-11-25-p2p-create-states.md` |
| Route-map fix | **uncommitted** | `backend/internal/server/routes_test.go`: P2P controller plus four routes |

The last commit is `e540a9f` (test runner). The slice-2 docs, skills, calibration designs, and
the route-map fix are in the working tree; no commits were requested for them.

## 2. Verified on 2026-11-25

```bash
cd backend && go build ./...
cd backend && go test ./internal/server/        # green, including the route-map fix
cd backend && make test                        # 12 packages, 240 tests, green
./scripts/pre-commit.sh                        # 0 issues
```

The test runner's unit, formatting, merge, HTML, threshold, and failure paths were verified by
hand. See `plans/test-runner/implementation.md` for the exact checks.

## 3. Open blockers

1. **Docker is unavailable.** This blocks:
   - `make test-integration`, `make test-e2e`, `make test-all`, `make coverage`, `make coverage-ci`;
   - the coverage merge comparison against the CI one-shot number;
   - the P2P data check below.
2. **Slice 2 is uncommitted.** Decide the split before merging:
   - route-map fix: `backend/internal/server/routes_test.go`
   - docs and ADRs: `docs/architecture-backend.md`, `docs/domain.md`, `docs/errors.md`, `docs/ui.md`, `docs/adr/`, `plans/agent-workflow/slice-2-implementation.md`
   - skills and calibration: `.opencode/skills/design-*/`, `docs/designs/`, the two `AGENTS.md` rows
   - working state: `plans/agent-workflow/handoff.md`, `plans/test-runner/implementation.md`

Once Docker is available:

```bash
cd backend
make test-integration
make test-all
make coverage
make coverage-ci
```

Compare the `make coverage` total with `make coverage-ci`. They should match; a difference points
at the merge logic, not at coverage itself.

## 4. P2P data check — still pending

The price fix changed the wire boundary. Pre-fix rows stored the unscaled request integer and
intent cannot be inferred. Before deploying:

```bash
cd backend && docker compose up -d
docker compose exec -T db psql -U devuser -d nsp_db -c "SELECT count(*) FROM p2p_sessions;"
docker compose exec -T db psql -U devuser -d nsp_db -c \
  "SELECT count(*) FROM market_trades WHERE p2p_session_id IS NOT NULL;"
```

Empty → nothing to do. Non-empty → stop and decide remediation before deploying. Do not
backfill automatically.

## 5. Slice 2 results

- `docs/architecture-backend.md` — 126 non-test Go files, layered table, P2P create walkthrough,
  change-placement table, package map, asymmetries.
- `docs/domain.md` — 26 model files: credits, ledger, positions, orders/trades, quotes/scores,
  P2P state machine, invariant table.
- `docs/errors.md` — 77 `errors.New` sites across ten packages, ownership table, translation
  rule, `marketStatusFor` / `p2pStatusFor` tables, known gaps.
- `docs/ui.md` — tokens, 15 components (11 + 4 admin), 7 pages, view routing, one 768px
  breakpoint, state matrix, rules with exceptions.
- `docs/adr/0001…0006` — layered flow, consumer interfaces, DTO boundary, credits scale, access
  levels, DTO package split.
- `design-backend` / `design-frontend` — primary-session skills, one-page output plus checklist.
- `AGENTS.md` — two Agent workflow rows.

Reality corrections found while writing the docs, worth keeping in mind:

- The frontend has 15 component folders, not 12; `docs/ui.md` says so.
- `CreditsADecimal` is duplicated in `internal/dto` and `internal/controller/dto`; only the latter
  has `CreditsDesdeDecimal`.
- 630 hard-coded colour literals across 21 of 23 CSS files; the new docs record this as debt.
- `README.md:93-95` claims tests skip without Docker; they fail loudly in `TestMain`. Left
  untouched because existing Spanish docs are frozen.

## 6. Calibration results

| Skill | Input | Output | Verdict |
| --- | --- | --- | --- |
| `design-backend` | The implemented P2P price fix | `docs/designs/2026-11-25-p2p-price-units.md` | Independently states decimal string at the wire, scaled `int64` inside, no service/repository change, OpenAPI and frontend deltas, data caveat, rejected options. |
| `design-frontend` | The `P2PCreatePage` loading-state P1 | `docs/designs/2026-11-25-p2p-create-states.md` | One page, component tree, states matrix, two files in the checklist, accessibility and copy included. |

Both are about 68–73 lines above the checklist, inside the template's "about one page" target.
No template change was needed. A useful side effect: the frontend calibration surfaced that
`loadPositions` depends on `selectedPlayerId`, so a naive loading branch would replace the form
on every selection; the design records the stable-callback fix.

## 7. Backlog — not scheduled

| Sev | Finding | Location |
| --- | --- | --- |
| P1 | Expiry job discards every error, including `tx.Commit()` | `backend/internal/persistence/repository/p2p.go:400-431` |
| P1 | Position/account locks taken in role order, not id order | `backend/internal/persistence/repository/p2p.go:189,202,239,252` |
| P1 | Unknown player maps to 500 instead of 404 | `backend/internal/controller/p2p.go`, `backend/internal/service/p2p.go:65` |
| P1 | Non-UUID token returns 500 | `backend/internal/controller/p2p.go` |
| P1 | P2P create form renders the empty state during load and after failure | `frontend/src/pages/P2PCreatePage/P2PCreatePage.tsx:186` |
| P1 | COMPLETED card tells the buyer the trade was taken by someone else | `frontend/src/pages/P2PSessionPage/P2PSessionPage.tsx:246-254` |
| P2 | 11 items: constraints, duplicate index, missing `SKIP LOCKED`, audit errors, retry idempotency, OpenAPI nullability, email exposure, refetch loop, clipboard, missing CSS, added comments | see `plans/p2p-token-session/review.md` |

Each gets its own task. Do not fold them into unrelated changes.

## 8. How to resume

1. Read `AGENTS.md`, then `docs/review.md` and the docs relevant to your task.
2. For a review: say "review this branch". Base defaults to `origin/dev`; the skill refuses
   diffs over 600 lines / 20 files and writes `plans/<task>/review.md`.
3. For a design: use `design-backend` or `design-frontend`. They ask at most three questions,
   write `docs/designs/<YYYY-MM-DD>-<topic>.md`, and stop before code.
4. For tests: `cd backend && make test` (unit, no Docker), `make test-integration` (Docker),
   `make coverage` (Docker, merged table plus HTML).
5. Re-run the command block in section 2 only after changing code. Do not redo the verified
   checks to "confirm" them.

`go test ./...` fails in `service` and `persistence/repository` without Docker. That is expected;
never silence or skip it.

## 9. Constraints that are easy to violate

- New agent-facing docs, plans, skills, and reports are English. Product strings, UI copy,
  existing test names, and existing comments stay Spanish and are never translated.
- `specs/` and `.specify/` are read-only co-worker input; never create spec directories or
  invoke `speckit-*`.
- Tests were allowed for the price fix only, on explicit request. The default remains: no test
  edits in a feature task.
- The design skills never write code, builds, or `plans/` files.
- Leave the tree uncommitted until the human chooses the split.

## 10. Definition of done

- `review-changes` produces a report a human can triage in under 10 minutes. Done.
- Both design skills produce a one-page decision plus a file checklist, with no code. Done.
- Docs are accurate enough that `design-backend` proposes a change without opening persistence
  or service internals. Verified by the price-fix calibration.
- Co-workers' speckit flow is untouched. Done.
- Remaining before merge: Docker verification, the data check, and the commit split.
