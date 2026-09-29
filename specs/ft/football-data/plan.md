# Implementation Plan: Load, Persist, and Display Players from Major European Leagues

**Branch**: `ft/football-data` | **Date**: 2026-09-28 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/ft/football-data/spec.md`

## Summary

Implement external ingestion, local PostgreSQL persistence, and a full-stack user catalog for active football players across the top five European competitions (Premier League, Bundesliga, La Liga, Serie A, and Ligue 1).

1. **Ingestion & Integration**: Consume the `football-data.org` API v4 via a specialized adapter in `internal/adapters/footballdata/`, authenticated using `NSPFOOTBALLDATAAPIKEY` passed in the `X-Auth-Token` header. Handle free-tier rate limits (10 req/min) with pacing, exponential backoff, and graceful recovery.
2. **Persistence & Data Integrity**: Store player records in PostgreSQL table `players` using `database/sql` without an ORM. Support idempotent upserting on `external_id`, soft-deactivate departed players (`active = false`), and record immutable transaction history in `player_audit_logs` tracking changes and diffs (Constitution Principle X).
3. **Backend API & Strict Layering**: Expose public read endpoints `GET /players` (paginated, filtered catalog returning summary DTOs with name, current team, league, and position only) and `GET /players/{id}` (full detail DTO), plus an administrative endpoint `POST /players/sync`. Strictly adhere to Constitution Principles III, IV, and V (independent `internal/dto/` package for transferring data between architectural layers without leaking domain models, separate boundary DTOs for list and detail contexts, pure domain models without struct tags, context propagation, constructor dependency injection wired in `cmd/`).
4. **Frontend React + TypeScript**: Implement a responsive web application adhering to Constitution Principle VI:
   - Encapsulate all HTTP communication in `frontend/src/api/players.ts` using axios privately.
   - Catalog List Page: renders players displaying only name, current team, league, and position, with reactive filtering (by league, club, position, and search) and server-side pagination.
   - Player Detail Page: displays all attributes of the selected player with non-breaking fallback indicators ("Unassigned", "-") and navigation back to catalog.
   - Component-level CSS adopting BEM conventions.

---

## Technical Context

**Language/Version**:
- Backend: Go 1.26.5 (governed by `backend/go.mod`)
- Frontend: Node.js 22 LTS, TypeScript ~5.7.2, React 18.3.1 (governed by `frontend/package.json`)

**Primary Dependencies**:
- Backend: Go standard library (`net/http`, `database/sql`, `log/slog`, `context`), PostgreSQL driver `github.com/lib/pq`, Zerolog (log handler)
- Frontend: React 18, TypeScript, Axios (inside API abstraction module only), Vite 6, Vitest 5, `@testing-library/react`
- Static Analysis: `golangci-lint v2.12.1` (`backend/.golangci.yml`), `eslint`, `knip`

**Storage**:
- PostgreSQL 15 Alpine (dockerized via `backend/docker-compose.yml` on port 5433)
- Tables: `players` and `player_audit_logs`, with indexes on `active`, `league_code`, `club_name`, `position`, and `name`.

**Testing**:
- Backend: `go test -v -race ./...` across model, service, controller, repository, and adapter layers. Mocks at layer boundaries; adapter tests use fixture payloads.
- Frontend: `npm test` (Vitest + React Testing Library) testing catalog listing, filtering, pagination, and detail views.
- CI / Hook: Repository-managed `scripts/pre-commit.sh` enforcing formatting, vet, and golangci-lint.

**Target Platform**:
- Linux server / Docker container environment (Linux x86_64, macOS, Windows via Git Bash).

**Project Type**:
- Full-stack web application monorepo (Go API in `backend/` + React SPA in `frontend/`).

**Performance Goals**:
- Public catalog query response latency < 2 seconds under standard load (SC-003).
- 0 external network requests during end-user catalog browsing (SC-002).
- Zero downtime impact: 100% catalog availability during external provider downtime (SC-004).

**Constraints**:
- Backend web endpoints implemented solely with standard library `net/http` (no web frameworks).
- Database access exclusively through `database/sql` (no ORM).
- Rate-limited external calls (10 req/min free tier).
- Strict separation between domain models (no struct tags) and DTOs (Principle V).
- DTOs organized into their own independent `internal/dto/` package to transfer data between layers.
- Distinct DTOs for list view (summary) and detail view (full attributes).
- Frontend CSS scoped per component using BEM methodology (no utility-first frameworks).

**Scale/Scope**:
- Top 5 European leagues (PL, BL1, PD, SA, FL1), ~100 clubs, ~2,500 active player profiles.

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Requirement | Satisfied By | Status |
|---|---|---|---|
| **I. Backend en Go con net/http** | REST API exposed only via `net/http`; dockerized app available | Standard `net/http` mux and routing in `internal/server/`; Dockerized app via Compose | PASS |
| **II. Persistencia en PostgreSQL** | PostgreSQL connected via standard `database/sql` without ORM; dockerized | `database/sql` in `internal/persistence/dao/`; dockerized service in `backend/docker-compose.yml` | PASS |
| **III. Arquitectura en capas** | Strict layer separation: `server`, `middleware`, `httphandler`, `logger`, `controller`, `service`, `persistence` (with `repository` and `dao`), `model`, `adapters`; `context.Context` propagated across all I/O | Repository coordinates DAOs; models cross repository boundary; context propagated to all DB and HTTP calls | PASS |
| **IV. Inyección de dependencias** | All dependencies injected via constructors; interfaces at consumer; graph assembled in `cmd/` | All layers receive interfaces via constructors; `cmd/server/main.go` wires full graph | PASS |
| **V. DTOs en el borde** | DTOs distinct from domain models; domain models lack struct tags; distinct DTOs for list vs detail; `DesdeModelo`/`AModelo` conversions; `httphandler` encode/decode | `PlayerListItemResponse` (name, club, league, position only) and `PlayerDetailResponse` (all fields); independent `internal/dto/` package for data transfer between layers; `DesdeModelo` in `dto/` package; no tags on `model.Player` | PASS |
| **VI. Frontend React + TypeScript** | React + TS; axios only inside API abstraction module; types mirror DTOs; BEM CSS | `frontend/src/api/players.ts` encapsulates axios; DTO interfaces mirrored; CSS with BEM per component | PASS |
| **VII. API REST documentada con OpenAPI** | Mandatory OpenAPI specification artifact kept synchronized | Documented in `specs/ft/football-data/contracts/openapi.yaml` | PASS |
| **VIII. Seguridad: JWT y validación** | Public catalog endpoints; authenticated/admin for sync; strict input validation | Public access to catalog; input query parameters validated; sync operation restricted | PASS |
| **IX. Observabilidad** | Structured `log/slog` with Zerolog; correlation ID in context; request latency and external call logging | Correlation ID in context, latency and error logging in `httphandler` and adapter | PASS |
| **X. Auditoría transaccional** | Immutable append-only audit trail with author, timestamp, old/new diffs | `player_audit_logs` table populated on insert, update, and deactivation | PASS |
| **XI. Integridad de datos** | Database operations transactional by default; schema indexed on query keys | Ingestion runs in `sql.Tx`; indexes on `active`, `league_code`, `club_name`, `position`, `name` | PASS |
| **XII. Resiliencia: caché y scheduler** | Reads served from local store; external failures do not affect browsing | 100% of read requests served from PostgreSQL; zero calls to football-data.org during catalog read | PASS |
| **XIII. Testing** | Unit, integration, and e2e testing; layer-specific mocking; boundary value coverage | Unit tests for model, service, controller; adapter tests against JSON fixtures; Vitest for frontend | PASS |
| **XIV. Semánticas** | Specification directory and identifier match active git branch | Branch is `ft/football-data`, directory is `specs/ft/football-data` | PASS |

---

## Project Structure

### Documentation (this feature)

```text
specs/ft/football-data/
├── spec.md                     # Feature specification
├── plan.md                     # Implementation plan (/speckit-plan output)
├── research.md                 # Technical research, decisions, and alternatives
├── data-model.md               # Domain models, SQL schema, DTOs, and state machine
├── quickstart.md               # Runnable local and end-to-end validation scenarios
└── contracts/
    ├── openapi.yaml            # OpenAPI 3.0 specification for REST endpoints
    ├── footballdata-external-api.md # Contract for external football-data.org integration
    └── frontend-api.md         # Frontend client module and UI component contracts
```

### Source Code (repository root)

```text
backend/
├── cmd/
│   └── server/
│       └── main.go             # Dependency injection assembly and server lifecycle
├── db/
│   └── init.sql                # PostgreSQL DDL for players and player_audit_logs tables
├── internal/
│   ├── adapters/
│   │   └── footballdata/
│   │       ├── client.go       # football-data.org HTTP client with auth and rate limiting
│   │       ├── client_test.go  # Adapter unit tests against recorded JSON fixtures
│   │       └── dto.go          # External API JSON response payloads
│   ├── configuration/
│   │   └── configuration.go    # Environment configuration loader (DSN, port, API key)
│   ├── controller/
│   │   ├── container.go        # Controller registry container
│   │   ├── player.go           # PlayerController (GetPlayers, GetPlayerByID)
│   │   ├── player_test.go      # Controller unit tests validating HTTP status & DTO handling
│   │   └── sync.go             # SyncController (SyncPlayers)
│   ├── dto/
│   │   ├── pagination.go       # PaginatedResponse[T] generic paginated transfer container
│   │   ├── player_detail.go    # PlayerDetailResponse (all attributes) + DesdeModelo/AModelo
│   │   ├── player_filter.go    # PlayerFilterDTO for querying and transferring filter criteria
│   │   ├── player_list.go      # PlayerListItemResponse (name, club, league, position only)
│   │   └── sync.go             # SyncPlayersResponse
│   ├── httphandler/
│   │   ├── utils.go            # Encode / Decode JSON helpers
│   │   └── wrapper.go          # Standardized HTTP error and logging wrapper
│   ├── logger/
│   │   └── logger.go           # Zerolog slog initialization
│   ├── middleware/
│   │   └── container.go        # Middleware handlers (correlation ID, logging, CORS)
│   ├── model/
│   │   ├── player.go           # Pure domain models (Player, PlayerFilter, PageResult)
│   │   └── audit.go            # Domain model (AuditLog)
│   ├── persistence/
│   │   ├── dao/
│   │   │   ├── container.go    # DAO registry container
│   │   │   ├── player_sql.go   # Player SQL queries (SELECT, UPSERT, DEACTIVATE)
│   │   │   ├── player_sql_test.go
│   │   │   └── audit_sql.go    # Audit log SQL queries (INSERT append-only)
│   │   └── repository/
│   │       ├── container.go    # Repository registry container
│   │       ├── player.go       # PlayerRepository implementation coordinating DAOs & Tx
│   │       └── player_test.go  # Repository unit tests with mocked DAOs
│   ├── server/
│   │   ├── routes.go           # HTTP routing mux (/players, /players/{id}, /players/sync)
│   │   └── server.go           # Server startup and shutdown logic
│   └── service/
│       ├── container.go        # Service registry container
│       ├── player.go           # PlayerService (ListPlayers, GetPlayerByID)
│       ├── player_test.go      # Service unit tests with mocked repository
│       └── player_sync.go      # PlayerSyncService (orchestrates adapter & repository sync)
├── docker-compose.yml          # PostgreSQL container configuration
└── app.env                     # Local environment variable template

frontend/
├── package.json                # React, TypeScript, Axios, Vitest dependencies
├── src/
│   ├── api/
│   │   ├── client.ts           # Axios instance configuration
│   │   ├── players.ts          # Encapsulated API functions (getPlayers, getPlayerById)
│   │   └── players.test.ts     # API client unit tests
│   ├── components/
│   │   ├── Navbar/
│   │   │   ├── Navbar.tsx
│   │   │   └── Navbar.css
│   │   ├── Pagination/
│   │   │   ├── Pagination.tsx
│   │   │   └── Pagination.css
│   │   ├── PlayerCard/
│   │   │   ├── PlayerCard.tsx
│   │   │   └── PlayerCard.css
│   │   ├── PlayerFilter/
│   │   │   ├── PlayerFilter.tsx
│   │   │   └── PlayerFilter.css
│   │   └── PlayerTable/
│   │       ├── PlayerTable.tsx
│   │       └── PlayerTable.css
│   ├── pages/
│   │   ├── PlayerListPage/
│   │   │   ├── PlayerListPage.tsx # Catalog list with name, club, league, position & filters
│   │   │   ├── PlayerListPage.css
│   │   │   └── PlayerListPage.test.tsx
│   │   └── PlayerDetailPage/
│   │       ├── PlayerDetailPage.tsx # Detail page with all player attributes
│   │       ├── PlayerDetailPage.css
│   │       └── PlayerDetailPage.test.tsx
│   ├── App.tsx                 # Routing and navigation between catalog and detail views
│   └── main.tsx                # React root mount
```

**Structure Decision**:
- Retains the full-stack repository structure (`backend/` for Go API and `frontend/` for React).
- Backend strictly complies with Constitution Principle III, placing domain models in `internal/model/` (free of tags), public DAOs in `internal/persistence/dao/`, coordinated domain repositories in `internal/persistence/repository/`, business services in `internal/service/`, external provider adapter in `internal/adapters/footballdata/`, and data transfer objects in their own dedicated package `internal/dto/` enabling data transfer across layers (HTTP edge, controller, service, persistence, adapters) rather than being confined inside the controller.
- Frontend encapsulates Axios within `frontend/src/api/players.ts` per Constitution Principle VI, organizing pages and components with accompanying BEM CSS files.

---

## Complexity Tracking

No constitution violations detected. All 14 principles pass without exceptions or compromises:
- `database/sql` without ORM fulfills Principle II.
- Context propagation throughout fulfills Principle III.
- Constructor injection and assembly in `cmd/` fulfills Principle IV.
- Distinct DTOs for list and detail contexts in independent `internal/dto` package fulfill Principle V and user requirements.
- Modular Axios encapsulation and BEM CSS fulfill Principle VI.
- Transactional audit log fulfills Principle X.
- Offline PostgreSQL cache fulfills Principle XII.

---

## Post-Design Constitution Check

| Principle | Status | Notes |
|---|---|---|
| **I. Backend en Go con net/http** | PASS | Standard library `net/http` used across all server routing and handler definitions. |
| **II. Persistencia en PostgreSQL** | PASS | `database/sql` with raw SQL queries; Dockerized PostgreSQL 15 in `docker-compose.yml`. |
| **III. Arquitectura en capas** | PASS | Clear layer boundaries: `server` → `controller` → `service` → `repository` → `dao`; `adapters` separated; `context.Context` propagated. |
| **IV. Inyección de dependencias** | PASS | Every layer exposes constructor accepting interfaces; container wiring in `cmd/server/main.go`. |
| **V. DTOs en el borde** | PASS | `PlayerListItemResponse` (name, club, league, position only) vs `PlayerDetailResponse` (all fields); independent `internal/dto/` package for cross-layer data transfer; `DesdeModelo` in `dto/`; domain models have zero tags. |
| **VI. Frontend React + TypeScript** | PASS | Axios isolated in `api/players.ts`; types mirror backend DTOs; BEM CSS files per page and component. |
| **VII. API REST documentada con OpenAPI** | PASS | Documented in `contracts/openapi.yaml`. |
| **VIII. Seguridad: JWT y validación** | PASS | Public read endpoints; sync endpoint restricted; all input parameters strictly validated. |
| **IX. Observabilidad** | PASS | Zerolog-backed `log/slog` logging correlation ID, latency, status, and error details. |
| **X. Auditoría transaccional** | PASS | `player_audit_logs` records author, timestamp, old state, and new state diffs immutably. |
| **XI. Integridad de datos** | PASS | Atomic database transactions during sync; schema indexes on all filtering keys. |
| **XII. Resiliencia: caché y scheduler** | PASS | All user catalog reads satisfied exclusively by PostgreSQL local storage; zero external calls on read. |
| **XIII. Testing** | PASS | Comprehensive testing strategy across unit, integration, and frontend components with specified mock boundaries. |
| **XIV. Semánticas** | PASS | Branch `ft/football-data` strictly corresponds to specification directory `specs/ft/football-data`. |
