# Review checklist

Operational checklist for reviewing a branch before commit or merge. Every item has a
mechanical check. If an item cannot be verified with a command or by reading one block of
code, report it as a gap instead of guessing.

Run all commands from the repository root. `BASE` defaults to `origin/dev`.

## 0. Scope

- [ ] Diff is reviewable: `git diff --stat "$BASE...HEAD"` stays under 600 changed lines
      and 20 files. Above that, stop and propose a split; a large diff cannot be reviewed
      in one pass.
- [ ] Nothing unrelated changed: read `git diff --name-only "$BASE...HEAD"`.
- [ ] Nothing under `specs/` or `.specify/` was modified:
      `git diff --name-only "$BASE...HEAD" -- specs .specify` must be empty.
- [ ] The branch builds before review: `cd backend && go build ./...` and
      `cd frontend && npm run typecheck`. If either fails, that is the first P0 and the
      review stops there.

## 1. Contract

- [ ] Every operation declares an access level. The counts must match:

      ```bash
      cd backend
      test "$(grep -cE '^    (get|post|put|patch|delete):' api/openapi.yaml)" \
         = "$(grep -cE '^      x-access-level:' api/openapi.yaml)"
      ```

- [ ] Access levels are only the known ones:
      `grep -oE '^      x-access-level: [a-z]+' backend/api/openapi.yaml | awk '{print $2}' | sort -u`
      must yield a subset of `anonymous authenticated renewal superuser`.
- [ ] Contract tests pass: `cd backend && go test ./internal/server/ -run Contract`.
      They need no database.
- [ ] New or changed route exists in `internal/server/routes.go` with an explicit
      `AccessLevel`; `AccessUndeclared` is never reachable.
- [ ] Controller, service, DTO, and route were added together. A route without a DTO or a
      DTO without a route is a P1.
- [ ] Request and response shapes in `openapi.yaml` match the DTO fields and their
      nullability.

## 2. Data and transactions

- [ ] Models carry no JSON tags: `grep -rn 'json:"' backend/internal/model/` must be empty.
      Conversions live in `internal/dto` via `DesdeModelo` / `AModelo`.
- [ ] Every multi-statement write is inside `db.BeginTx`, with rollback on every error path.
      Read the whole method; do not infer it from the first lines.
- [ ] Locking is explicit where two actors can race: `FOR UPDATE` for a single row,
      `SKIP LOCKED` for batch expiry. The lock order is consistent (user then position,
      then trade) to avoid deadlocks.
- [ ] `backend/db/init.sql` changes are paired with `expectedTables` in
      `backend/cmd/server/main.go`. A new table without an entry means the server starts
      against a broken schema.
- [ ] Constraints (FK, unique, check, positive values) live in `init.sql`, not only in Go.
- [ ] Money and token amounts are integers; no `float64` on a price, amount, or balance.
- [ ] Operations that can be retried are idempotent, or the duplicate is rejected by a
      constraint.

## 3. Authorization, input, logging

- [ ] Authorization uses domain data in the service (owner, participant, admin), not only
      the route's access level.
- [ ] The controller validates request shape only; business rules stay in the service.
- [ ] Errors are mapped to status codes consistently; a new sentinel error has a mapping
      and a test-visible status.
- [ ] No credential, token, email, or other PII is logged. New log lines propagate
      `correlation_id`.

## 4. Frontend

- [ ] axios is imported only under `src/api/`:

      ```bash
      grep -rn "from ['\"]axios" frontend/src --include=*.ts --include=*.tsx \
        | grep -v '^frontend/src/api/'
      ```

      must be empty.
- [ ] One typed function per endpoint in `src/api/<module>.ts`; no URL strings in
      components.
- [ ] Styles use BEM classes in the component's CSS file; no inline `style={{`.
- [ ] Loading, empty, error, and unauthorized states exist for every new request.
- [ ] Types match the API: dates are strings, nullable fields are `| null`, and no field is
      assumed present because the mock had it.
- [ ] Countdowns and timers clean up on unmount; no interval leaks.

## 5. Prohibitions

- [ ] Tests were not created or modified unless the task was explicitly a testing task:
      `git diff --name-only "$BASE...HEAD" -- '*_test.go' '*.test.ts' '*.test.tsx'`
      must be empty or justified in the report.
- [ ] No dependency changes without justification:
      `git diff --name-only "$BASE...HEAD" -- go.mod go.sum frontend/package.json frontend/package-lock.json`.
- [ ] Existing strings and comments were not translated, paraphrased, or deleted.
      Additions are fine; rewrites of Spanish text are not.
- [ ] No comments were added to code.
- [ ] No generated or vendored files were hand-edited.

## Severity

| Level | Meaning |
| --- | --- |
| P0 | Data loss or corruption, auth bypass, broken contract, non-compiling code, money miscalculation |
| P1 | Missing error mapping, realistic race, missing UI state, authorization in the wrong layer |
| P2 | Duplication, dead code, inconsistent naming, missing log field |

Do not report formatting, import order, or style that `gofmt`, `golangci-lint`, `eslint`,
or `knip` already cover. If a check cannot be performed (for example, Docker is
unavailable for an integration test), list it under gaps instead of marking it passed.
