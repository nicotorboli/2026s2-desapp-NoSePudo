# Handoff — agent workflow

Branch: `fix/p2p-price-scale` · Last update: 2026-11-16 · Language: English (per `AGENTS.md`)

Everything below is uncommitted. Read this first, then `AGENTS.md`, then the file for the task
you are picking up.

## 1. What exists and is verified

| Artifact | State | Notes |
| --- | --- | --- |
| `docs/review.md` | new, verified | Review checklist with mechanical checks. Commands were run against the tree and pass. |
| `.opencode/agents/reviewer.md` | new, verified | Read-only subagent. Shell denied, `speckit-*` skills denied, `steps: 40`. |
| `.opencode/skills/review-changes/SKILL.md` + `references/report-template.md` | new, discovered | Skill ID `review-changes` is loaded by OpenCode. |
| `AGENTS.md` | edited | Added Language, Spec-kit (co-workers only), Agent workflow. |
| `.gitignore` | edited | `.opencode/*` with `agents/` and `skills/` re-included so skills can be committed. |
| `plans/p2p-token-session/review.md` | new | Calibration review of the merged P2P feature: 1 P0 + 6 P1. |
| `plans/p2p-price-scale-fix/{plan,implementation}.md` | new | P0 task record. |
| P2P price fix | implemented, verified | Backend + OpenAPI + frontend + tests. See section 3. |
| 3 test files | new, passing | `dto/p2p_test.go`, `controller/p2p_test.go`, `P2PCreatePage.test.tsx`. |

Verification already run on this branch:

```bash
cd backend && go build ./...
go test ./internal/controller/... ./internal/model/... ./internal/dto/... \
  ./internal/middleware/... ./internal/httphandler/...
go test ./internal/server/ -run Contract
./scripts/pre-commit.sh                 # 0 issues

cd frontend && npm run typecheck && npm run lint && npm run knip && npm test
```

Results: backend targeted packages pass, contract 3/3, pre-commit clean, frontend 16 files /
58 tests pass. Do not redo this unless you change code.

## 2. Open decisions (do these before slice 2, they are small)

1. **Route test map.** `go test ./internal/server/` fails in
   `TestDeclaredLevelsMatchTheSpecification` (`internal/server/routes_test.go:151`) because its
   hardcoded map lacks the four P2P routes. Reproduces on `dev`; not caused by the price fix.
   Decision: add the four P2P entries to the map, or record why they stay out. The test's own
   message treats this as an exposure decision.
2. **Existing P2P data.** The data check could not run: no Docker daemon, port 5433 closed.
   Once Docker is available:

   ```bash
   cd backend && docker compose up -d
   docker compose exec -T db psql -U devuser -d nsp_db -c \
     "SELECT count(*) FROM p2p_sessions;"
   docker compose exec -T db psql -U devuser -d nsp_db -c \
     "SELECT count(*) FROM market_trades WHERE p2p_session_id IS NOT NULL;"
   ```

   Empty → nothing to do. Non-empty → stop and decide remediation before deploying the fix;
   the pre-fix rows store unscaled integers and intent cannot be inferred.
3. **Commit split.** The tree mixes three concerns. Suggested commits:
   - review tooling: `.gitignore`, `AGENTS.md`, `docs/review.md`, `.opencode/agents/reviewer.md`,
     `.opencode/skills/review-changes/**`
   - review report: `plans/p2p-token-session/review.md`
   - price fix: `backend/`, `frontend/`, `plans/p2p-price-scale-fix/**`

## 3. Slice 2 — context docs and design skills

Goal: make design decisions cheap by giving the agent the context it currently re-derives from
~18k lines of backend code, then add two skills that use that context.

### 3.1 Docs (new, English)

| File | Content | Sources |
| --- | --- | --- |
| `docs/architecture-backend.md` | layer table with responsibilities; one real request walkthrough (P2P create: route → controller → service → repository → dao → pg); "where does this change go?" table; package map; known asymmetries (two `dto` packages, repository vs service sentinels) | `backend/internal/*`, `backend/cmd/server` |
| `docs/domain.md` | Credits scale and arithmetic; accounts and double-entry ledger; positions and reservations; orders/trades; quotes and scores; P2P state machine; invariants and owning package per concept | `backend/internal/model` (38 files) |
| `docs/errors.md` | the three sentinel namespaces and the translation rule (repository → service → controller); error → status table per controller | 51 `errors.New` declarations, `marketStatusFor`, `p2pStatusFor` |
| `docs/ui.md` | palette and tokens (`index.css`); Rubik; BEM convention (block = folder in kebab-case); 12 components and 7 pages with one-line responsibility; routing in `App.tsx`; single breakpoint at 768px; loading/empty/error/unauthorized patterns; rules (axios only in `src/api/`, no inline styles, Spanish copy) | `frontend/src` |

Each doc header must include the command that regenerates its inventory.

### 3.2 ADRs (new, `docs/adr/`)

Five one-pagers, ~15 lines each: layered flow; consumer-defined interfaces; DTO boundary and no
JSON tags; credits as scaled int64 plus the wire-boundary rule; explicit access levels and JWT
middleware.

Unresolved, decide during the slice: `internal/dto` vs `internal/controller/dto`. Document
reality as a sixth ADR or update `AGENTS.md`; do not consolidate packages now.

### 3.3 Skills (new, `.opencode/skills/`)

`design-backend` and `design-frontend`, each with `SKILL.md` and
`references/decision-template.md`. They run in the **primary** session because they ask
questions; they are not subagents.

Shared procedure:

1. Read only `AGENTS.md` + 3 relevant docs + `specs/<feature>/spec.md` if it exists.
2. At most 3 questions, each with a stated default.
3. Present 2–3 options with repo-specific trade-offs.
4. Write `docs/designs/<YYYY-MM-DD>-<topic>.md`; never overwrite without asking.
5. Stop. No code, no `plans/` writes, no builds.

Backend template: invariants · data/constraints · transactions and lock order · HTTP contract
delta (method, path, access level, DTOs, units) · authorization matrix · error mapping ·
concurrency/idempotency · migration/rollback · observability · rejected options · file checklist
· out of scope.

Frontend template: route and placement · component tree (reuse vs create) · state ownership ·
`src/api` functions with signatures · states matrix · BEM blocks and tokens · 768px behavior ·
accessibility · new copy · file checklist · non-goals.

Hard rules: output ≤ 1 page plus the checklist; cite `path:line`; never read
`specs/*/{plan,tasks,research}.md`; never invoke `speckit-*`.

### 3.4 `AGENTS.md`

Add two rows to the Agent workflow table pointing at the two design skills and `docs/designs/`.

### 3.5 Validation for slice 2

- Run `design-backend` on the P2P price fix (already implemented) and compare its decision doc
  with `plans/p2p-price-scale-fix/plan.md`. Missing the unit boundary means the template lacks a
  section.
- Run `design-frontend` on the portfolio loading-state P1; expect one page, component tree,
  states matrix, ≤ 6 files.
- Reject any design doc over one page plus checklist; tighten the template.
- Check each doc's inventory command against the tree.

Estimated effort: docs + ADRs ≈ 2 h, skills ≈ 1 h, calibration ≈ 30 min.

## 4. Backlog found by the review, not scheduled

| Sev | Finding | Location |
| --- | --- | --- |
| P1 | Expiry job discards every error, including `tx.Commit()` | `backend/internal/persistence/repository/p2p.go:400-431` |
| P1 | Position/account locks taken in role order, not id order | `backend/internal/persistence/repository/p2p.go:189,202,239,252` |
| P1 | Unknown player maps to 500 instead of 404 | `backend/internal/controller/p2p.go:124`, `backend/internal/service/p2p.go:65` |
| P1 | Non-UUID token returns 500 | `backend/internal/controller/p2p.go:65,87,108` |
| P1 | Portfolio empty state renders during load and after failure | `frontend/src/pages/P2PCreatePage/P2PCreatePage.tsx:181` |
| P1 | COMPLETED card tells the buyer the trade was taken by someone else | `frontend/src/pages/P2PSessionPage/P2PSessionPage.tsx:246-254` |
| P2 | 11 items (constraints, duplicate index, missing `SKIP LOCKED`, audit errors, retry idempotency, OpenAPI nullability, email exposure, refetch loop, clipboard, missing CSS, added comments) | see `plans/p2p-token-session/review.md` |

Do not fix these as part of slice 2. Each gets its own task if you take it.

## 5. How to resume

1. `git status` — expect the three groups in section 2.3 to be uncommitted.
2. Read `AGENTS.md` (language and spec-kit rules), then `docs/review.md`.
3. For the review workflow: say "review this branch". Base defaults to `origin/dev`; the skill
   refuses diffs over 600 lines / 20 files and writes `plans/<task>/review.md`.
4. For slice 2: start with `docs/architecture-backend.md`, because the backend design skill
   depends on it; `docs/ui.md` is independent and can be done in parallel.
5. Verify with the command block in section 1.
6. `go test ./...` will fail in Docker-backed packages (`service`, `persistence/repository`)
   without the daemon, and in `internal/server` until decision 2.1 is resolved. That is expected;
   do not silence it.

## 6. Constraints that are easy to violate

- New agent-facing docs, plans, skills, and reports are English. Product strings, UI copy,
  existing test names, and existing comments stay Spanish and are never translated.
- `specs/` and `.specify/` are read-only co-worker input; never create spec directories or
  invoke `speckit-*`.
- Tests were allowed for the price fix only, on explicit request. The default remains: no test
  edits in a feature task.
- The reviewer subagent has no shell and cannot edit. Findings still need human triage.
- `formatCredits` was not changed; the new helper is `formatCreditDecimal` (trims zeros, comma).
- The P2P service, repository, and DAO work in `model.Credits` and were intentionally untouched.

## 7. Definition of done for the whole workflow project

- Review skill produces a report a human can triage in under 10 minutes on a normal-sized diff.
- Both design skills produce a one-page decision plus a file checklist, with no code.
- Docs are accurate enough that `design-backend` does not need to open persistence or service
  packages to propose a change.
- Co-workers' speckit flow still works untouched.
