# AGENTS.md

Monorepo for **NoSePudo**: Go backend in `backend/`, React + TypeScript + Vite frontend in `frontend/`. All docs, comments, and tests are in Spanish.

This file is the operative guidance for working in this repo and supersedes `.specify/memory/constitution.md` — do not rely on the constitution to implement features.

## Architecture (backend)

- Strict layered structure: `cmd/` (wires concrete implementations only) + `internal/{server,middleware,httphandler,logger,controller,service,model,persistence/{repository,dao},adapters,dto}`. Flow: controller → service → repository → dao → adapters.
- Stdlib only for the core: `net/http` and `database/sql`. No ORMs or web frameworks.
- Constructor DI everywhere (`New...`), no global state; `context.Context` must be the first parameter of every I/O function.
- Models never carry `json` tags; conversions live in `internal/dto` via `DesdeModelo`/`AModelo`. All JSON encode/decode goes through `internal/httphandler`.
- `backend/api/openapi.yaml` is the maintained API contract. A contract test (`internal/server/contract_test.go`) fails the build if a route lacks `x-access-level` or mismatches the code — any new or changed endpoint must update the OpenAPI spec in the same commit.

## Conventions

- **Auth**: JWT HS256 (`golang-jwt/jwt/v5`) + bcrypt. `internal/middleware` extracts and injects the actor and privilege level into `context.Context`; every route must declare an explicit access level (absence is an error, never a public endpoint). Validate inputs at the controller/middleware; authorize against domain data in the service.
- **Logging**: structured JSON via `log/slog` with the Zerolog handler, centralized in `internal/logger`; propagate `correlation_id`; never log credentials, tokens, or PII.
- **Persistence**: wrap multi-statement operations in `db.BeginTx` transactions; FK/unique constraints live in `db/init.sql`.
- **Testing**: prefer integration tests against real Postgres via testcontainers; mocks are allowed only for external services, never for the database or persistence. Layer scoping: `model` = pure in-memory unit tests, `service`/`repository` = testcontainers, `adapters` = stubs/fixtures.
- **Features and tests must not be mixed.** A feature implementation plan must NOT create or edit existing tests unless explicitly commanded; tests are run only to check for regressions. A testing plan must NOT modify implementation code to relax or make tests pass unless explicitly commanded.
- **Strings and comments are frozen.** Do not modify, paraphrase, or delete user-provided strings or comments. Do not add comments unless explicitly requested.
- **Post-implementation mini plan**: after implementing a feature, write one or more short files under `plans/[plan_name]/` listing every modification made.
- **Frontend**: all HTTP calls must go through `frontend/src/api/` (one typed function per endpoint); importing `axios` anywhere else is forbidden. Styles use BEM classes in per-page/component CSS files.

## Commands

Backend (from `backend/`):

```bash
docker compose up -d        # Postgres on port 5433 (devuser/devpassword/nsp_db)
go run ./cmd/server         # refuses to start without NSPPSQLDS, NSP_JWT_SECRET (>=32 bytes), or missing DB tables
go build ./...
go test -race ./...
```

Frontend (from `frontend/`): `npm run typecheck`, `npm run lint`, `npm run knip`, `npm run test`, `npm run test:coverage`, `npm run dev`.

Full backend check identical to CI: `./scripts/pre-commit.sh` from the repo root (gofmt + `go vet` + golangci-lint at the version pinned in `.golangci-version`).

## Gotchas

- **Pre-commit hook** (`.githooks/pre-commit`) auto-formats staged Go files (`gofmt -w`, `golangci-lint --fix`, including `fieldalignment` struct-field reordering) and re-stages them; it may leave unstaged collateral edits in other files.
- **Docker is required** for repository, service, and e2e tests (testcontainers). Tests must NOT silently skip and report `ok` when Docker is unavailable — missing Docker must fail loudly.
- `go test -race` needs cgo (a C compiler); on Windows without one, run without `-race`. CI runs with `-race` on Linux.
- **Schema changes**: `db/init.sql` only runs on an empty Postgres volume. After editing it: `docker compose down -v && docker compose up -d`.
- golangci-lint is pinned in `.golangci-version` and enforced by `pre-commit.sh` (exact version match); notable rules: `gocyclo` max complexity 15, `fieldalignment` (order struct fields by type size).

## CI

`.github/workflows/ci.yml` runs backend (`./scripts/pre-commit.sh`, then `go test -v -race -coverpkg=./...` with coverage) and frontend (`npm ci`, typecheck, lint, knip, `test:coverage`) jobs; both upload coverage artifacts consumed by a separate SonarQube job (`sonar-project.properties`). Local verification should match these steps.
