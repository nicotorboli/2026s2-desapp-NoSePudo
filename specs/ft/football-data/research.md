# Research & Technical Decisions: Player Ingestion, Persistence & Catalog UI

**Feature Branch**: `ft/football-data`  
**Date**: 2026-09-28  
**Feature Spec**: [spec.md](./spec.md)

## 1. External Football Data Provider Integration (football-data.org)

### Decision
Integrate with the [football-data.org API v4](https://www.football-data.org/documentation/api) using a custom adapter in `internal/adapters/footballdata/`.
- **Base URL**: `https://api.football-data.org/v4/`
- **Authentication**: Header `X-Auth-Token: <API_KEY>`, loaded via environment variable `NSPFOOTBALLDATAAPIKEY`.
- **Target Competitions & Codes**:
  - Premier League (England): `PL` (Competition ID: 2021)
  - Bundesliga (Germany): `BL1` (Competition ID: 2002)
  - La Liga (Spain): `PD` (Competition ID: 2014)
  - Serie A (Italy): `SA` (Competition ID: 2019)
  - Ligue 1 (France): `FL1` (Competition ID: 2015)
- **Ingestion Traversal**:
  1. Retrieve participating teams for each competition via `GET /v4/competitions/{code}/teams`.
  2. For each team returned, retrieve full squad members via `GET /v4/teams/{team_id}` (or directly from `teams[i].squad` if populated by the subscription tier).
  3. Extract player attributes: `id` (external ID), `name`, `position`, `dateOfBirth`, `nationality`, `shirtNumber`, associated with the club name and league.

### Rationale
- football-data.org v4 is the canonical provider requested by the user and documented in the feature specification.
- Competitions `PL`, `BL1`, `PD`, `SA`, `FL1` represent the top five European leagues required by FR-001.
- Storing the external person `id` enables idempotent upserts and re-synchronization tracking across runs.

### Alternatives Considered
- *Single-tier competition teams squad payload*: Some API responses do not populate the `squad` array inside `/competitions/{code}/teams`. The adapter will check if `squad` is non-empty; if empty, it falls back to querying `GET /v4/teams/{team_id}`.
- *API-Football (RapidAPI) or OpenFooty*: Rejected because football-data.org was explicitly specified in the requirements.

---

## 2. Rate Limiting, Throttling & External Downtime Resiliency

### Decision
Implement client-side rate limiting and exponential backoff inside `internal/adapters/footballdata/`:
- **Rate Limit Pacing**: The free tier of football-data.org allows 10 requests per minute (~1 request every 6 seconds). The adapter provides a configurable request delay (`time.Duration`, default 6s in live mode, 0 in mocked tests).
- **HTTP 429 Handling**: Detect HTTP 429 responses, inspect the `X-RequestCounter-Reset` or `Retry-After` header if present, and back off gracefully. If retries are exhausted, abort the sync cycle cleanly within a database transaction, leaving existing stored players intact.
- **Cache & Offline Resiliency (Constitution Principle XII & FR-008)**: All catalog reads (`GET /players`, `GET /players/{id}`) query the local PostgreSQL database exclusively. Zero outgoing HTTP calls are made during end-user browsing. If football-data.org is completely down, user experience is unaffected.

### Rationale
- Fulfills SC-002 (0 external service calls during browsing), SC-004 (100% data browsable during external outage), and edge case handling for HTTP 429 rate limits.
- Meets Constitution Principle XII (Resiliencia: caché y scheduler).

### Alternatives Considered
- *Unthrottled concurrent requests*: Would immediately trigger HTTP 429 from football-data.org and fail synchronization.
- *Proxy caching layer*: Unnecessary complexity since PostgreSQL acts as the local system of record.

---

## 3. Database Schema, Data Integrity & Ingestion Reconciliation

### Decision
Define a robust schema in PostgreSQL using standard `database/sql` (no ORM, per Constitution Principle II) with two tables: `players` and `player_audit_logs`.

#### `players` Table Schema:
- `id` (BIGSERIAL PRIMARY KEY)
- `external_id` (BIGINT UNIQUE NOT NULL)
- `name` (VARCHAR(255) NOT NULL)
- `club_name` (VARCHAR(255) NOT NULL)
- `league_name` (VARCHAR(100) NOT NULL)
- `league_code` (VARCHAR(10) NOT NULL)
- `date_of_birth` (DATE NULL)
- `nationality` (VARCHAR(100) NULL)
- `position` (VARCHAR(50) NOT NULL)
- `shirt_number` (INTEGER NULL)
- `active` (BOOLEAN NOT NULL DEFAULT TRUE)
- `created_at` (TIMESTAMPTZ NOT NULL DEFAULT NOW())
- `updated_at` (TIMESTAMPTZ NOT NULL DEFAULT NOW())

#### Indexes:
- `idx_players_external_id` (UNIQUE)
- `idx_players_active`
- `idx_players_league_code`
- `idx_players_club_name`
- `idx_players_position`

#### Re-synchronization Strategy & Reconciliation:
1. Synchronization runs within a transaction (`database/sql.Tx`).
2. Incoming rostered players are upserted via `INSERT INTO players (...) VALUES (...) ON CONFLICT (external_id) DO UPDATE SET ...`.
3. Players who were previously active in the synced leagues but are absent from all current squad rosters are updated to `active = FALSE`, recording an audit event. They are never deleted.
4. An immutable audit record is written to `player_audit_logs` tracking changes per Constitution Principle X.

### Rationale
- Satisfies FR-002, FR-003, FR-010, SC-001, and SC-005.
- Indexes optimize filtering by league, club, position, and active status per Constitution Principle XI (Integridad de datos).
- Soft deactivation preserves historical data and avoids cascade deletions.

### Alternatives Considered
- *Hard deletion of departed players*: Rejected by FR-003 and SC-005, which mandate preserving departed players and marking them inactive.
- *Separate normalized tables for clubs and leagues*: Adds unnecessary join overhead for the required read catalog. Denormalizing club name and league name into `players` matches the bounded context while maintaining simplicity.

---

## 4. Backend Layering & Strict DTO Separation (Constitution Principles III, IV, V)

### Decision
Adhere strictly to the clean layered architecture specified in Constitution Principles III, IV, and V:
- **`internal/model/`**: Contains pure domain models (`Player`, `PlayerFilter`, `PageResult[T]`, `AuditLog`). No struct tags (`json:`, `db:`) are permitted on domain models.
- **`internal/dto/`**: Standalone package containing Data Transfer Objects (DTOs) used to transfer data between architectural layers (e.g., HTTP edge to controller, controller to service, service to controller/adapters) without coupling layers to concrete internal domain implementations:
  - `PlayerListItemResponse`: Contains only `{ id, name, club, league, position }` per the user requirement ("only the name, current team, league and position").
  - `PlayerDetailResponse`: Contains full attributes `{ id, externalId, name, club, league, leagueCode, position, dateOfBirth, nationality, shirtNumber, active, createdAt, updatedAt }`.
  - `PlayerFilterDTO`: Transfers filtering and pagination parameters across layers.
  - `PaginatedResponse[T]`: Generic wrapper for paginated transfer data.
  - Explicit conversion functions `DesdeModelo` and `AModelo`. Different contexts use distinct DTOs per Constitution Principle V.
- **`internal/persistence/dao/`**:
  - `PlayerSql`: Public DAO receiving `*sql.DB`, executing SQL queries and scanning results into models.
  - `AuditSql`: Public DAO executing append-only inserts for audit logs.
- **`internal/persistence/repository/`**:
  - `PlayerRepository`: Coordinates `PlayerSql` and `AuditSql`. Translates database models to domain entities and enforces transactional boundaries.
- **`internal/service/`**:
  - `PlayerService`: Business logic for querying catalog players with filters and pagination, and fetching single player details, accepting and producing DTOs/models across layers.
  - `PlayerSyncService`: Business logic for orchestrating data retrieval from the external adapter and persistence in the repository.
- **`internal/adapters/footballdata/`**: Implements the external provider integration.
- **`internal/controller/`**: Exposes REST endpoints, consumes `internal/dto`, and routes requests:
  - `GET /players`: Returns paginated list of `PlayerListItemResponse`.
  - `GET /players/{id}`: Returns detailed `PlayerDetailResponse`.
  - `POST /players/sync`: Triggers synchronization process (admin/superuser).

### Rationale
- Fulfills the explicit user requirement: list view with only name, current team, league, and position; detail view with all attributes.
- DTOs live in their own dedicated package (`internal/dto/`), enabling clean data transfer across layers (controller, service, adapters) rather than being confined inside the controller.
- Upholds Constitution Principle V: "Request y respuesta son DTOs distintos, y cada contexto en que se devuelve un modelo tiene su propio DTO: no se reusa uno agregándole campos opcionales."
- Propagates `context.Context` across all layers per Principle III.

### Alternatives Considered
- *Returning the same DTO for list and detail with omitted fields*: Explicitly prohibited by Constitution Principle V.
- *Calling DAO directly from controller or service*: Explicitly prohibited by Constitution Principle III.

---

## 5. Frontend Architecture, Routing & Styling (Constitution Principle VI)

### Decision
Implement the frontend in React + TypeScript using Vite:
- **API Abstraction Module (`frontend/src/api/players.ts`)**:
  - Axios is imported only inside the api module. Private helper `get` / `post`.
  - Exported functions: `getPlayers(params: PlayerFilterParams): Promise<PaginatedPlayersDTO>`, `getPlayerById(id: number): Promise<PlayerDetailDTO>`.
  - TypeScript types strictly mirror the backend DTOs.
- **Pages**:
  - `PlayerListPage`: Features filter controls (league dropdown, club search/select, position filter, active status toggle, text search), players table/grid displaying name, club, league, position, and pagination controls.
  - `PlayerDetailPage`: Displays complete player profile (name, club, league, position, date of birth, nationality, shirt number, active badge, last updated timestamp), with friendly fallbacks for missing/unassigned values and navigation back to catalog.
- **Routing & State**:
  - Simple client-side page routing (e.g. view state or React Router) allowing seamless navigation between `/` (list) and `/players/:id` (detail).
- **Styling**:
  - Component-specific CSS files adopting the BEM naming methodology (e.g., `player-list__table`, `player-card__attribute--highlight`).

### Rationale
- Directly implements the user request: list page with name, current team, league, position, and filtering functions; player detail page with all attributes.
- Strictly adheres to Constitution Principle VI (React + TypeScript, axios in abstraction module only, BEM CSS, no direct axios in UI components).

### Alternatives Considered
- *Using third-party UI component libraries (Material UI, Tailwind, AntD)*: Plain CSS with BEM is mandated by Constitution Principle VI ("los estilos se manejan con archivos CSS por página/componente, usando clases y siguiendo el arquetipo BEM").
- *Single-page modal for details*: Separate dedicated page view fulfills the user requirement ("a player detail page that list all the attributes of that one") and supports direct URL linking and clean testing.

---

## 6. Testing Strategy Across All Layers (Constitution Principle XIII)

### Decision
1. **Unit Tests**:
   - `model`: Domain validation and filter evaluation without DB or network.
   - `service`: Business orchestration tested with mock `PlayerRepository` and mock `FootballDataAdapter`.
   - `controller`: Contract tests validating HTTP status codes, headers, and DTO conversions.
   - `adapters`: Tested against recorded JSON fixtures (no live external network calls).
2. **Integration Tests**:
   - `repository` and `dao`: Tested with PostgreSQL database queries verifying correct SQL execution, transaction handling, upsert behavior, and soft-deactivation.
3. **Frontend Tests**:
   - Tested using Vitest and `@testing-library/react`.
   - `PlayerListPage.test.tsx`: Tests rendering players, filter inputs, page switching, empty states, and clicking a player to navigate to detail.
   - `PlayerDetailPage.test.tsx`: Tests rendering all player attributes, fallback indicators ("Unassigned"), and back navigation.
   - `api/players.test.ts`: Verifies API client calls with correct query parameters.

### Rationale
- Satisfies Constitution Principle XIII (Coverage of happy path, negative cases, boundary values, and mock boundaries).
