# Tasks: Load, Persist, and Display Players from Major European Leagues

**Feature Branch**: `ft/football-data` | **Date**: 2026-09-28 | **Spec**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md)

> **IMPORTANT**: This file was generated from `spec.md`, `plan.md`, `data-model.md`, `contracts/`, `research.md`, and `quickstart.md`.
> Tasks are organized by user story so each story can be implemented and tested independently.
> Do NOT modify task IDs — they are referenced for dependency tracking.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Repository scaffolding, Docker environment, and Go/Node project initialization.

- [ ] T001 Create `backend/db/init.sql` with `players` table DDL (columns: `id BIGSERIAL PK`, `external_id BIGINT NOT NULL UNIQUE`, `name VARCHAR(255) NOT NULL`, `club_name VARCHAR(255) NOT NULL`, `league_name VARCHAR(100) NOT NULL`, `league_code VARCHAR(10) NOT NULL`, `date_of_birth VARCHAR(20)`, `nationality VARCHAR(100)`, `position VARCHAR(50) NOT NULL`, `shirt_number INTEGER`, `active BOOLEAN NOT NULL DEFAULT TRUE`, `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`, `updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`) and `player_audit_logs` table DDL (columns: `id BIGSERIAL PK`, `entity_type VARCHAR(50) NOT NULL`, `entity_id BIGINT NOT NULL`, `action VARCHAR(20) NOT NULL`, `actor VARCHAR(100) NOT NULL`, `diff_old JSONB`, `diff_new JSONB NOT NULL`, `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`) plus all required indexes (`idx_players_active`, `idx_players_league_code`, `idx_players_club_name`, `idx_players_position`, `idx_players_name`, `idx_audit_entity`, `idx_audit_created_at`)
- [ ] T002 Create `backend/docker-compose.yml` with a PostgreSQL 15 Alpine service named `nsp_postgres`, port `5433:5432`, credentials `devuser/devpassword`, database `nsp_db`, and volume-mounted `./db/init.sql` for automatic schema initialization at container startup
- [ ] T003 Create `backend/app.env` with environment variable template containing `NSPPSQLDS`, `NSPFOOTBALLDATAAPIKEY`, and `PORT` (default `8080`) values, documenting each variable
- [ ] T004 [P] Initialize `backend/go.mod` (module `github.com/nicotorboli/2026s2-desapp-NoSePudo/backend`, Go 1.26.5) and add dependencies `github.com/lib/pq` (PostgreSQL driver) and `github.com/rs/zerolog` (structured logging); run `go mod tidy`
- [ ] T005 [P] Initialize `frontend/package.json` with React 18.3.1, TypeScript ~5.7.2, Axios, Vite 6, Vitest 5, `@testing-library/react`, ESLint, and Knip; run `npm install`
- [ ] T006 Create `backend/.golangci.yml` enabling `bodyclose`, `gocritic`, `gocyclo` (max complexity 15), `gosec`, `nilerr`, `noctx`, `rowserrcheck`, `sqlclosecheck`, `unconvert`, and `govet` with `enable-all: true`; create `.golangci-version` file with `v2.12.1`
- [ ] T007 [P] Create `scripts/pre-commit.sh` enforcing `go fmt`, `go vet`, and `golangci-lint run`; create `.githooks/pre-commit` symlink/entry that sources `scripts/pre-commit.sh` for Linux, macOS, and Git Bash on Windows

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core backend infrastructure all user stories depend on. No user story work can begin until this phase is complete.

**⚠️ CRITICAL**: All user story phases depend on this phase being complete.

- [ ] T008 Create `backend/internal/configuration/configuration.go` with a `Config` struct and `Load() (Config, error)` function reading `NSPPSQLDS` (PostgreSQL DSN), `NSPFOOTBALLDATAAPIKEY` (external API key), and `PORT` (default `"8080"`) from environment variables; return a descriptive error if any required variable is missing
- [ ] T009 Create `backend/internal/logger/logger.go` with `Init(level slog.Level) *slog.Logger` initializing a `log/slog` logger backed by a Zerolog handler; log structured JSON to stdout; propagate `context.Context` carrying a correlation ID key `"correlation_id"`
- [ ] T010 Create `backend/internal/model/player.go` defining pure domain structs `Player` (fields: `ID int64`, `ExternalID int64`, `Name string`, `ClubName string`, `LeagueName string`, `LeagueCode string`, `DateOfBirth *string`, `Nationality *string`, `Position string`, `ShirtNumber *int`, `Active bool`, `CreatedAt time.Time`, `UpdatedAt time.Time`), `PlayerFilter` (fields: `League string`, `Club string`, `Position string`, `Search string`, `IncludeInactive bool`, `Page int`, `Limit int`), and `PageResult[T any]` (fields: `Items []T`, `Page int`, `Limit int`, `Total int64`, `TotalPages int`) — **no struct tags on any field** (Constitution Principle V)
- [ ] T011 Create `backend/internal/model/audit.go` defining pure domain struct `AuditLog` (fields: `ID int64`, `EntityType string`, `EntityID int64`, `Action string`, `Actor string`, `DiffOld map[string]any`, `DiffNew map[string]any`, `CreatedAt time.Time`) — **no struct tags** (Constitution Principle V)
- [ ] T012 Create `backend/internal/httphandler/utils.go` with generic `Encode[T any](w http.ResponseWriter, status int, v T) error` (JSON marshal, set Content-Type and status) and `Decode[T any](r *http.Request) (T, error)` (JSON unmarshal body with size limit); create `backend/internal/httphandler/wrapper.go` with `Wrap(fn func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc` that calls `fn`, logs errors with correlation ID from context, and writes `{"error": "...", "message": "..."}` JSON with appropriate HTTP status code
- [ ] T013 Create `backend/internal/middleware/container.go` with middleware functions: `CorrelationID(next http.Handler) http.Handler` (generates UUID, stores in `context.Context` and sets `X-Correlation-ID` response header), `RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler` (logs method, path, status, and latency), and `CORS(next http.Handler) http.Handler` (sets permissive CORS headers for development)
- [ ] T014 Create `backend/internal/server/routes.go` defining `RegisterRoutes(mux *http.ServeMux, cc controller.Container, mc middleware.Container)` that registers routes `GET /players`, `GET /players/{id}`, and `POST /players/sync` with middleware chain applied; create `backend/internal/server/server.go` with `NewServer(addr string, handler http.Handler) *http.Server` and `Run(ctx context.Context) error` with graceful shutdown on `SIGTERM`/`SIGINT`
- [ ] T015 Create `backend/cmd/server/main.go` wiring the full dependency graph: `configuration.Load()` → `sql.Open("postgres", dsn)` → `dao.NewContainer(db)` → `repository.NewContainer(daoContainer)` → `service.NewContainer(repoContainer, adapterClient)` → `controller.NewContainer(serviceContainer)` → `server.NewServer` → `server.Run(ctx)` — all via constructor injection per Constitution Principle IV; log startup errors and exit with code 1

**Checkpoint**: Foundation ready — all containers and wiring exist; user story implementation can begin.

---

## Phase 3: User Story 1 — Browse Active Players by League and Profile Attributes (Priority: P1) 🎯 MVP

**Goal**: Public catalog of active players with name, club, date of birth, nationality, position, and shirt number; server-side pagination; filters by league, club, and position; graceful fallbacks for missing values.

**Independent Test**: Pre-populate the `players` table with fixture rows; start the backend; `GET /players` returns HTTP 200 with paginated items containing only `id`, `name`, `club`, `league`, `position`; `GET /players/1` returns HTTP 200 with full profile including `dateOfBirth`, `nationality`, `shirtNumber` (null if unassigned); `GET /players?league=PL` returns only Premier League players; no external HTTP calls are made during any of these operations.

### Implementation for User Story 1

- [ ] T016 [P] [US1] Create `backend/internal/dto/player_list.go` with `PlayerListItemResponse` struct (fields: `ID int64 \`json:"id"\``, `Name string \`json:"name"\``, `Club string \`json:"club"\``, `League string \`json:"league"\``, `Position string \`json:"position"\``) and function `PlayerListItemDesdeModelo(m model.Player) PlayerListItemResponse` mapping `m.ID`, `m.Name`, `m.ClubName`, `m.LeagueName`, `m.Position`
- [ ] T017 [P] [US1] Create `backend/internal/dto/player_detail.go` with `PlayerDetailResponse` struct (all fields with json tags: `id`, `externalId`, `name`, `club`, `league`, `leagueCode`, `position`, `dateOfBirth *string`, `nationality *string`, `shirtNumber *int`, `active bool`, `createdAt string`, `updatedAt string`) and function `PlayerDetailDesdeModelo(m model.Player) PlayerDetailResponse` formatting `CreatedAt`/`UpdatedAt` as `time.RFC3339`
- [ ] T018 [P] [US1] Create `backend/internal/dto/pagination.go` with `PaginatedResponse[T any]` struct (fields: `Items []T \`json:"items"\``, `Page int \`json:"page"\``, `Limit int \`json:"limit"\``, `Total int64 \`json:"total"\``, `TotalPages int \`json:"totalPages"\``)
- [ ] T019 [P] [US1] Create `backend/internal/dto/player_filter.go` with `PlayerFilterDTO` struct (fields: `League string`, `Club string`, `Position string`, `Search string`, `IncludeInactive bool`, `Page int`, `Limit int`) and conversion functions `PlayerFilterDesdeQuery(r *http.Request) (PlayerFilterDTO, error)` (parse query params `league`, `club`, `position`, `search`, `includeInactive`, `page` ≥ 1, `limit` 1–100 default 20, return 400-friendly error on invalid values) and `PlayerFilterAModelo(d PlayerFilterDTO) model.PlayerFilter`
- [ ] T020 [US1] Create `backend/internal/persistence/dao/player_sql.go` defining interface `PlayerDAO` with methods `ListPlayers(ctx context.Context, f model.PlayerFilter) ([]model.Player, int64, error)` and `GetPlayerByID(ctx context.Context, id int64) (model.Player, error)` and `UpsertPlayer(ctx context.Context, tx *sql.Tx, p model.Player) (int64, error)` and `DeactivateMissingPlayers(ctx context.Context, tx *sql.Tx, activeExternalIDs []int64, leagueCode string) ([]int64, error)`; implement `PlayerSql` struct with `*sql.DB`; `ListPlayers` builds parameterized SQL with optional `WHERE active = TRUE` (unless `IncludeInactive`), `AND league_code = $n`, `AND club_name ILIKE $n`, `AND position ILIKE $n`, `AND name ILIKE $n` clauses, `LIMIT`/`OFFSET` pagination, and a separate `COUNT(*)` query; `GetPlayerByID` fetches single row by `id` returning `model.ErrNotFound` on no rows; `UpsertPlayer` issues `INSERT INTO players (...) VALUES (...) ON CONFLICT (external_id) DO UPDATE SET name=$n, club_name=$n, ..., updated_at=NOW()`; `DeactivateMissingPlayers` issues `UPDATE players SET active=FALSE, updated_at=NOW() WHERE league_code=$1 AND active=TRUE AND external_id != ALL($2) RETURNING id`
- [ ] T021 [P] [US1] Create `backend/internal/persistence/dao/audit_sql.go` defining interface `AuditDAO` with method `InsertAuditLog(ctx context.Context, tx *sql.Tx, log model.AuditLog) error`; implement `AuditSql` struct; SQL: `INSERT INTO player_audit_logs (entity_type, entity_id, action, actor, diff_old, diff_new) VALUES ($1,$2,$3,$4,$5,$6)` with `diff_old` and `diff_new` JSON-marshaled as JSONB
- [ ] T022 [P] [US1] Create `backend/internal/persistence/dao/container.go` with `Container` struct holding `PlayerDAO` and `AuditDAO` interfaces and `NewContainer(db *sql.DB) Container` constructor instantiating `PlayerSql` and `AuditSql`
- [ ] T023 [US1] Create `backend/internal/persistence/repository/player.go` defining interface `PlayerRepository` with methods `ListPlayers(ctx, filter model.PlayerFilter) (model.PageResult[model.Player], error)` and `GetPlayerByID(ctx, id int64) (model.Player, error)` and `SavePlayers(ctx, players []model.Player, leagueCode, actor string) error`; implement `PlayerRepositoryImpl` composing `dao.PlayerDAO` and `dao.AuditDAO`; `SavePlayers` opens `sql.Tx`, upserts each player recording `INSERT`/`UPDATE` audit logs (diff old/new), calls `DeactivateMissingPlayers` writing `DEACTIVATE` audit logs for each returned deactivated ID, commits transaction — all inside a single `sql.Tx`
- [ ] T024 [P] [US1] Create `backend/internal/persistence/repository/container.go` with `Container` struct holding `PlayerRepository` interface and `NewContainer(dc dao.Container) Container` constructor
- [ ] T025 [US1] Create `backend/internal/service/player.go` defining interface `PlayerService` with methods `ListPlayers(ctx context.Context, filter dto.PlayerFilterDTO) (dto.PaginatedResponse[dto.PlayerListItemResponse], error)` and `GetPlayerByID(ctx context.Context, id int64) (dto.PlayerDetailResponse, error)`; implement `PlayerServiceImpl` accepting `repository.PlayerRepository`; `ListPlayers` calls `repo.ListPlayers` with `dto.PlayerFilterAModelo(filter)`, maps each `model.Player` using `dto.PlayerListItemDesdeModelo`, wraps in `dto.PaginatedResponse`; `GetPlayerByID` calls `repo.GetPlayerByID` and maps with `dto.PlayerDetailDesdeModelo`
- [ ] T026 [P] [US1] Create `backend/internal/service/container.go` with `Container` struct holding `PlayerService` and `PlayerSyncService` interfaces and `NewContainer(rc repository.Container, adapter footballdata.Client) Container` constructor
- [ ] T027 [US1] Create `backend/internal/controller/player.go` defining `PlayerController` struct accepting `service.PlayerService`; implement `GetPlayers(w http.ResponseWriter, r *http.Request)` parsing filter params via `dto.PlayerFilterDesdeQuery`, calling `svc.ListPlayers`, encoding response with `httphandler.Encode`; implement `GetPlayerByID(w http.ResponseWriter, r *http.Request)` parsing `id` path param (return 400 on non-integer), calling `svc.GetPlayerByID` (return 404 on not found), encoding response; propagate `context.Context` to all service calls
- [ ] T028 [P] [US1] Create `backend/internal/controller/container.go` with `Container` struct holding `PlayerController` and `SyncController` and `NewContainer(sc service.Container) Container` constructor

**Backend US1 Checkpoint**: `GET /players` and `GET /players/{id}` return correct DTOs from PostgreSQL with no external calls.

### Frontend Implementation for User Story 1

- [ ] T029 [US1] Create `frontend/src/api/client.ts` configuring an Axios instance with `baseURL` from `import.meta.env.VITE_API_BASE_URL` (default `http://localhost:8080`); export the instance as default for use only within `src/api/`
- [ ] T030 [US1] Create `frontend/src/api/players.ts` exporting TypeScript interfaces `PlayerListItemDTO`, `PaginatedPlayersDTO`, `PlayerDetailDTO`, `PlayerFilterParams` mirroring backend DTOs exactly; export async functions `getPlayers(params?: PlayerFilterParams): Promise<PaginatedPlayersDTO>` and `getPlayerById(id: number): Promise<PlayerDetailDTO>` using the private Axios client; **Axios must not be imported anywhere outside `src/api/`**
- [ ] T031 [P] [US1] Create `frontend/src/components/PlayerFilter/PlayerFilter.tsx` with filter controls: league dropdown (options: All, Premier League, Bundesliga, La Liga, Serie A, Ligue 1 with codes PL/BL1/PD/SA/FL1), club name text input, position dropdown (All, Goalkeeper, Defender, Midfielder, Attacker), and search input; emit `onChange(params: PlayerFilterParams)` callback on any change; BEM CSS in `PlayerFilter.css` using classes `.player-filter`, `.player-filter__group`, `.player-filter__input`, `.player-filter__select`, `.player-filter__button`
- [ ] T032 [P] [US1] Create `frontend/src/components/Pagination/Pagination.tsx` accepting props `page`, `totalPages`, `total`, and `onPageChange(page: number)`; render Previous/Next buttons (disabled when at first/last page), current page and total pages indicator; BEM CSS in `Pagination.css` using classes `.pagination`, `.pagination__button`, `.pagination__button--disabled`, `.pagination__info`
- [ ] T033 [P] [US1] Create `frontend/src/components/PlayerTable/PlayerTable.tsx` accepting `players: PlayerListItemDTO[]` and `onSelectPlayer(id: number)` props; render a table/grid showing **only** `name`, `club`, `league`, `position` columns per each player row; clicking a row calls `onSelectPlayer(player.id)`; BEM CSS in `PlayerTable.css`
- [ ] T034 [US1] Create `frontend/src/pages/PlayerListPage/PlayerListPage.tsx` composing `PlayerFilter`, `PlayerTable`, and `Pagination`; manage local state for `filters` and `page`; call `getPlayers` on mount and on filter/page change; show loading spinner during fetch; show empty-state indicator (e.g. "No players found") when `items` is empty; show friendly error message on API failure; BEM CSS in `PlayerListPage.css` using `.player-list-page`, `.player-list-page__header`, `.player-list-page__title`, `.player-list-page__content`, `.player-list-page__empty`
- [ ] T035 [P] [US1] Create `frontend/src/components/PlayerCard/PlayerCard.tsx` for Player Detail page displaying all player attributes: name, active status badge (`.player-card__badge--active` or `.player-card__badge--inactive`), club, league, leagueCode, position, `dateOfBirth` (formatted as `YYYY-MM-DD` or `"—"` if null), `nationality` (or `"Unknown"` if null), `shirtNumber` (formatted as `"#7"` or `"Unassigned"` if null), `externalId`, `createdAt`, `updatedAt`; BEM CSS in `PlayerCard.css` using `.player-card`, `.player-card__header`, `.player-card__badge--active`, `.player-card__badge--inactive`, `.player-card__field`, `.player-card__label`, `.player-card__value`
- [ ] T036 [US1] Create `frontend/src/pages/PlayerDetailPage/PlayerDetailPage.tsx` reading player `id` from route params; call `getPlayerById(id)` on mount; render `PlayerCard` with full attributes; show loading state during fetch; show not-found error message if API returns 404; include a "Back to Catalog" button navigating to the list page; BEM CSS in `PlayerDetailPage.css` using `.player-detail-page` and `.player-card` blocks
- [ ] T037 [US1] Create `frontend/src/App.tsx` wiring React Router routes: `/` or `/players` → `PlayerListPage`, `/players/:id` → `PlayerDetailPage`; create `frontend/src/main.tsx` mounting the React app at `#root`
- [ ] T038 [P] [US1] Create `frontend/src/components/Navbar/Navbar.tsx` with site navigation (app name/logo and link back to catalog); BEM CSS in `Navbar.css`

**US1 Checkpoint**: A visitor can open the browser, view the paginated player catalog filtered by league/club/position, click a player, and see all profile attributes — all served from PostgreSQL with no external calls.

---

## Phase 4: User Story 2 — Synchronize and Persist External Player Data Locally (Priority: P2)

**Goal**: `POST /players/sync` retrieves active squads from football-data.org for all five leagues, persists them locally via idempotent upsert, soft-deactivates departed players, records audit logs, and returns a sync summary response.

**Independent Test**: With an empty `players` table, call `POST /players/sync` (requires mock/real API key); verify `players` table has rows for all 5 leagues; call `POST /players/sync` again and verify no duplicate rows; verify `player_audit_logs` contains `INSERT` records for first sync and `UPDATE` records where attributes changed.

### Implementation for User Story 2

- [ ] T039 [US2] Create `backend/internal/adapters/footballdata/dto.go` with Go structs matching the football-data.org v4 API JSON response shapes: `CompetitionTeamsResponse` (fields: `Competition struct{ Name, Code string }`, `Teams []TeamPayload`), `TeamPayload` (fields: `ID int64`, `Name string`, `Squad []PlayerPayload`), `TeamDetailResponse` (fields: `ID int64`, `Name string`, `Squad []PlayerPayload`), `PlayerPayload` (fields: `ID int64`, `Name string`, `Position string`, `DateOfBirth *string`, `Nationality *string`, `ShirtNumber *int`); all fields with `json:` struct tags for unmarshaling
- [ ] T040 [US2] Create `backend/internal/adapters/footballdata/client.go` defining interface `Client` with method `FetchLeaguePlayers(ctx context.Context, leagueCode string) ([]model.Player, error)`; implement `FootballDataClient` struct holding `httpClient *http.Client`, `apiKey string`, and `requestDelay time.Duration`; constructor `NewClient(apiKey string, delay time.Duration) *FootballDataClient`; implement two private methods: `fetchCompetitionTeams(ctx, code string) (CompetitionTeamsResponse, error)` calling `GET /v4/competitions/{code}/teams` with `X-Auth-Token` header, `fetchTeamDetail(ctx, teamID int64) (TeamDetailResponse, error)` calling `GET /v4/teams/{teamID}` with same auth header; in `FetchLeaguePlayers` iterate all 5 leagues' teams, use `squad` field if non-empty or fall back to `fetchTeamDetail`; respect `requestDelay` between requests (pacing for 10 req/min free tier); on HTTP 429 read `Retry-After` header and retry up to 3 times with exponential backoff, then return sentinel error `ErrRateLimitExceeded`; apply domain mapping rules (position normalization: `"Offence"` → `"Attacker"`, `"Defence"` → `"Defender"`, `"Midfield"` → `"Midfielder"`, `"Goalkeeper"` stays; `shirtNumber=0` → nil)
- [ ] T041 [US2] Create `backend/internal/service/player_sync.go` defining interface `PlayerSyncService` with method `SyncPlayers(ctx context.Context, actor string) (dto.SyncPlayersResponse, error)`; implement `PlayerSyncServiceImpl` accepting `footballdata.Client` and `repository.PlayerRepository`; for each of the 5 league codes (`PL`, `BL1`, `PD`, `SA`, `FL1`) call `adapter.FetchLeaguePlayers(ctx, code)` then `repo.SavePlayers(ctx, players, code, actor)`; accumulate `totalProcessed` and `totalUpdated` from results; on `ErrRateLimitExceeded` log warning and return partial success without data corruption
- [ ] T042 [P] [US2] Create `backend/internal/dto/sync.go` with `SyncPlayersResponse` struct (fields: `Status string \`json:"status"\``, `Message string \`json:"message"\``, `TotalProcessed int \`json:"totalProcessed"\``, `TotalUpdated int \`json:"totalUpdated"\``, `TotalDeactivated int \`json:"totalDeactivated"\``)
- [ ] T043 [US2] Create `backend/internal/controller/sync.go` with `SyncController` struct accepting `service.PlayerSyncService`; implement `SyncPlayers(w http.ResponseWriter, r *http.Request)` that calls `svc.SyncPlayers(ctx, "system/sync")`, encodes `dto.SyncPlayersResponse` as JSON with status 200, or returns 429 on `ErrRateLimitExceeded`, or 500 on any other error; propagate `context.Context`
- [ ] T044 [P] [US2] Create `backend/internal/adapters/footballdata/client_test.go` with unit tests that load JSON fixture files from a `testdata/` directory (recording real API response shapes) and assert: (a) league players are correctly mapped to `model.Player`; (b) position normalization rules are applied (`"Offence"` → `"Attacker"`, etc.); (c) `shirtNumber=0` is stored as `nil`; (d) `dateOfBirth=null` is stored as `nil *string`; no live network calls are made

**US2 Checkpoint**: `POST /players/sync` populates the database; subsequent `GET /players` calls serve data from PostgreSQL with zero external calls.

---

## Phase 5: User Story 3 — Resilient Local Data Availability During External Downtime (Priority: P3)

**Goal**: When football-data.org is unreachable or returns errors, all previously synchronized player data remains fully browsable; sync failures are logged cleanly without corrupting local data.

**Independent Test**: Populate `players` table with fixture data; set `NSPFOOTBALLDATAAPIKEY` to an invalid key; call `GET /players`, `GET /players?league=PL`, and `GET /players/1` — all return HTTP 200 with correct data. Call `POST /players/sync` — it returns a logged error response (429 or 500) while `players` table remains intact and unchanged.

### Implementation for User Story 3

- [ ] T045 [US3] Update `backend/internal/adapters/footballdata/client.go` to ensure all sync failure paths (network timeout, HTTP 5xx, `ErrRateLimitExceeded` after exhausted retries) return descriptive errors without panicking; add `context.Context` deadline/timeout enforcement so a timed-out sync does not leave the database in an inconsistent state; verify `sql.Tx.Rollback()` is called on any error path in `PlayerRepositoryImpl.SavePlayers`
- [ ] T046 [US3] Update `backend/internal/service/player_sync.go` to log each sync failure per league using structured `slog` with correlation ID from context, including the league code, error type, and retry count; ensure partial sync results (some leagues succeeded, some failed) preserve all previously persisted data and do not roll back successful leagues
- [ ] T047 [US3] Update `backend/internal/controller/sync.go` to return HTTP 429 with `{"error":"Rate Limit Exceeded","message":"External provider rate limit hit; existing data is intact"}` when `ErrRateLimitExceeded`; return HTTP 500 with `{"error":"Sync Failed","message":"..."}` on other errors; never expose raw Go error messages to the HTTP response
- [ ] T048 [P] [US3] Validate frontend resilience: update `frontend/src/pages/PlayerListPage/PlayerListPage.tsx` and `frontend/src/pages/PlayerDetailPage/PlayerDetailPage.tsx` to handle API errors gracefully — show a user-friendly error banner (e.g. "Unable to load players. Please try again later.") without crashing the page; verify via browser that catalog loads normally when the backend is running with an invalid API key

**US3 Checkpoint**: Catalog reads always succeed from PostgreSQL; sync failures are logged and return clean error responses; no data corruption occurs.

---

## Phase 6: Unit Tests (Constitution Principle XIII)

**Purpose**: Cover all layers with unit tests as mandated by the constitution.

- [ ] T049 [P] Create `backend/internal/persistence/dao/player_sql_test.go` with integration-style unit tests using a real PostgreSQL connection (or test doubles): verify `ListPlayers` applies all filter combinations, `GetPlayerByID` returns `ErrNotFound` on missing ID, `UpsertPlayer` inserts on first call and updates on second call (idempotent), `DeactivateMissingPlayers` sets `active=false` only for players not in `activeExternalIDs`
- [ ] T050 [P] Create `backend/internal/persistence/repository/player_test.go` with unit tests mocking `PlayerDAO` and `AuditDAO`; verify `SavePlayers` calls `UpsertPlayer` for each player, writes `INSERT` audit logs for new players, writes `UPDATE` audit logs for changed players, calls `DeactivateMissingPlayers`, writes `DEACTIVATE` audit logs for returned IDs, and commits `sql.Tx`; verify rollback on any DAO error
- [ ] T051 [P] Create `backend/internal/service/player_test.go` with unit tests mocking `repository.PlayerRepository`; verify `ListPlayers` returns correct `PaginatedResponse` shape and that `PlayerListItemResponse` contains only `id`, `name`, `club`, `league`, `position`; verify `GetPlayerByID` returns `PlayerDetailResponse` with all fields; verify error propagation on repository errors
- [ ] T052 [P] Create `backend/internal/controller/player_test.go` with HTTP-level unit tests using `httptest.NewRecorder()`; verify `GET /players` returns HTTP 200 with paginated JSON matching `PaginatedPlayerListResponse` schema; verify `GET /players/{id}` returns HTTP 200 with full `PlayerDetailResponse` schema; verify `GET /players/{id}` returns HTTP 404 when player not found; verify `GET /players?page=abc` returns HTTP 400
- [ ] T053 [P] Create `backend/internal/controller/sync_test.go` with HTTP-level unit tests: verify `POST /players/sync` returns HTTP 200 with `SyncPlayersResponse` JSON on success; verify HTTP 429 on `ErrRateLimitExceeded`; verify HTTP 500 on generic error
- [ ] T054 [P] Create `frontend/src/api/players.test.ts` with Vitest unit tests mocking Axios; verify `getPlayers()` calls the correct endpoint with correct query params; verify `getPlayerById(id)` calls `/players/{id}`; verify response is correctly typed
- [ ] T055 [P] Create `frontend/src/pages/PlayerListPage/PlayerListPage.test.tsx` with React Testing Library tests: render with mocked `getPlayers` returning fixture data; assert player names, clubs, leagues, and positions appear in the DOM; simulate league filter change and assert `getPlayers` is called with updated params; simulate page navigation and assert page param changes; assert empty state renders when `items` is empty
- [ ] T056 [P] Create `frontend/src/pages/PlayerDetailPage/PlayerDetailPage.test.tsx` with React Testing Library tests: render with mocked `getPlayerById` returning fixture player; assert all attributes render (including `"Unassigned"` for null shirt number, `"—"` for null dateOfBirth, `"Unknown"` for null nationality); assert the "Back to Catalog" button navigates to the list page; assert not-found error message when API returns 404

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Quality gates, observability, OpenAPI compliance, and integration validation.

- [ ] T057 [P] Validate OpenAPI compliance: ensure all three endpoints (`GET /players`, `GET /players/{id}`, `POST /players/sync`) match the schemas defined in `specs/ft/football-data/contracts/openapi.yaml`; verify response field names use camelCase (`externalId`, `shirtNumber`, `dateOfBirth`, `leagueCode`, `totalPages`) matching the OpenAPI spec exactly
- [ ] T058 [P] Add structured observability to `backend/internal/httphandler/wrapper.go`: log every request with `method`, `path`, `status`, `duration_ms`, and `correlation_id` from context; log external adapter calls with `league_code`, `team_id`, `status_code`, and `duration_ms` in `backend/internal/adapters/footballdata/client.go`
- [ ] T059 Add `frontend/vite.config.ts` with a dev proxy so requests to `/players` are forwarded to `http://localhost:8080`; create `.env.example` documenting `VITE_API_BASE_URL`; run `npm run typecheck` and `npm run lint` and fix all issues
- [ ] T060 Run `cd backend && go test -v -race ./...` and fix any race conditions or failing tests; run `golangci-lint run` and fix all lint issues identified by `bodyclose`, `noctx`, `rowserrcheck`, `sqlclosecheck`, `errcheck`, `staticcheck`, and `fieldalignment`
- [ ] T061 Run quickstart validation scenarios from `specs/ft/football-data/quickstart.md`: Scenario 1 (DB init), Scenario 3 (REST API), Scenario 4 (Frontend), Scenario 6 (quality gates); document any deviations

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 completion — **BLOCKS all user stories**
- **User Story 1 (Phase 3)**: Depends on Phase 2 completion
- **User Story 2 (Phase 4)**: Depends on Phase 2 completion — can run in parallel with US1
- **User Story 3 (Phase 5)**: Depends on Phase 4 (US2) completion — US3 is hardening of US2
- **Unit Tests (Phase 6)**: Can be written alongside their respective implementation phases
- **Polish (Phase 7)**: Depends on Phases 3, 4, and 5 completion

### Within User Story 1

- T016–T019 (DTOs) and T020–T022 (DAOs) can be developed in parallel
- T023 (Repository) depends on T020–T022
- T025 (Service) depends on T023
- T027 (Controller) depends on T025
- T029–T030 (Frontend API) can start once backend contracts are known
- T031–T033 (Frontend components) can develop in parallel with T029–T030
- T034–T037 (Frontend pages) depend on their respective components

### Within User Story 2

- T039 (Adapter DTOs) and T042 (Sync DTO) can run in parallel
- T040 (Adapter client) depends on T039
- T041 (SyncService) depends on T040 and T023
- T043 (SyncController) depends on T041

### Parallel Opportunities

```bash
# Phase 1 parallelism:
T004  # Go module init
T005  # npm install
T006  # golangci config
T007  # pre-commit hook

# Phase 2 - foundational models (no deps):
T010  # model/player.go
T011  # model/audit.go

# Phase 3 - US1 DTOs (all parallel):
T016  # dto/player_list.go
T017  # dto/player_detail.go
T018  # dto/pagination.go
T019  # dto/player_filter.go

# Phase 3 - US1 DAOs (all parallel after T010):
T020  # dao/player_sql.go
T021  # dao/audit_sql.go
T022  # dao/container.go

# Phase 3 - US1 Frontend components (all parallel after T030):
T031  # PlayerFilter component
T032  # Pagination component
T033  # PlayerTable component
T035  # PlayerCard component
T038  # Navbar component
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete **Phase 1** (Setup — T001–T007)
2. Complete **Phase 2** (Foundation — T008–T015) — CRITICAL BLOCKER
3. Complete **Phase 3** (US1 — T016–T038)
4. **STOP AND VALIDATE**: Run `GET /players` and `GET /players/{id}` against pre-seeded data; open browser at `http://localhost:5173` and verify catalog renders
5. Deploy/demo US1 independently

### Incremental Delivery

1. **Setup + Foundational** → Backend scaffolding complete
2. **Add US1** → Public catalog browsable → Demo MVP
3. **Add US2** → Sync from football-data.org → Data populates automatically
4. **Add US3** → Resilience hardened → Production-ready
5. **Tests + Polish** → CI/CD clean → Merge-ready

### Parallel Team Strategy

With multiple developers after Phase 2:
- **Developer A**: US1 backend (T016–T028)
- **Developer B**: US1 frontend (T029–T038) — can start once backend contracts are stable
- **Developer C**: US2 adapter + sync (T039–T044)

---

## Notes

- `[P]` tasks operate on different files with no incomplete-task dependencies — safe to parallelize
- `[US1]`, `[US2]`, `[US3]` labels map tasks to their user story for traceability
- Domain models (`internal/model/`) must **never** have `json:` or `db:` struct tags (Constitution Principle V)
- Axios must be imported **only** within `frontend/src/api/` (Constitution Principle VI)
- All SQL operations in sync use `sql.Tx`; rollback on any error (Constitution Principle XI)
- `player_audit_logs` rows are immutable — never UPDATE or DELETE (Constitution Principle X)
- All I/O layers accept and propagate `context.Context` (Constitution Principle III)
- Verify `go test -v -race ./...` passes with zero races before marking any backend task complete
