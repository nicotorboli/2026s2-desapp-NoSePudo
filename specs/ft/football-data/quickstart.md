# Quickstart Validation Guide: Player Ingestion, Persistence & Catalog UI

**Feature Branch**: `ft/football-data`  
**Date**: 2026-09-28  
**Spec**: [spec.md](./spec.md) | **Plan**: [plan.md](./plan.md)

This guide documents runnable validation scenarios that prove the feature works end-to-end, covering database initialization, external ingestion from football-data.org, public API catalog endpoints, and the React frontend listing and detail pages.

---

## Prerequisites

1. **Go Toolchain**: Go `1.26.x` installed matching [backend/go.mod](../../backend/go.mod).
2. **Node.js & npm**: Node.js `22 LTS` installed.
3. **Docker Engine & Docker Compose**: Active Docker daemon to run PostgreSQL.
4. **football-data.org API Key**: A valid token from [football-data.org](https://www.football-data.org/client/register) set in environment variable `FOOTBALL_DATA_API_KEY`.
5. **PostgreSQL Client / psql** (optional): for inspecting database tables directly.

---

## Scenario 1: Database Initialization & Table Setup

### Objective
Verify that PostgreSQL starts with the required `players` and `player_audit_logs` tables and indexes.

### Validation Steps
1. Start the PostgreSQL container:
   ```bash
   cd backend
   docker compose up -d db
   ```
2. Verify container health:
   ```bash
   docker compose ps
   ```
3. Inspect the database schema:
   ```bash
   docker exec -it nsp_postgres psql -U devuser -d nsp_db -c "\d players"
   docker exec -it nsp_postgres psql -U devuser -d nsp_db -c "\d player_audit_logs"
   ```

### Expected Outcome
- Container `nsp_postgres` is healthy on port `5433`.
- Table `players` contains all required columns: `id`, `external_id`, `name`, `club_name`, `league_name`, `league_code`, `date_of_birth`, `nationality`, `position`, `shirt_number`, `active`, `created_at`, `updated_at`.
- Table `player_audit_logs` contains immutable audit columns and indexes as defined in [data-model.md](./data-model.md).

---

## Scenario 2: Synchronize External Player Data (Backend Ingestion)

### Objective
Fetch active players for the 5 European leagues (Premier League, Bundesliga, La Liga, Serie A, Ligue 1) from football-data.org, persist them in PostgreSQL, and generate audit entries.

### Validation Steps
1. Set the API key environment variable and start the backend server:
   ```bash
   cd backend
   export NSPPSQLDS="user=devuser password=devpassword host=127.0.0.1 port=5433 dbname=nsp_db sslmode=disable"
   export FOOTBALL_DATA_API_KEY="<your-api-key>"
   go run cmd/server/main.go
   ```
2. Trigger the player synchronization in a separate terminal:
   ```bash
   curl -X POST http://localhost:8080/players/sync
   ```
3. Verify stored player count in PostgreSQL:
   ```bash
   docker exec -it nsp_postgres psql -U devuser -d nsp_db -c "SELECT league_name, COUNT(*) FROM players WHERE active = TRUE GROUP BY league_name;"
   ```
4. Verify audit entries:
   ```bash
   docker exec -it nsp_postgres psql -U devuser -d nsp_db -c "SELECT action, COUNT(*) FROM player_audit_logs GROUP BY action;"
   ```

### Expected Outcome
- The sync endpoint returns HTTP 200 with JSON summary (`totalProcessed`, `totalUpdated`, `status: "success"`).
- Active players are populated across all 5 leagues.
- Repeated sync runs do not create duplicate player records (idempotent upsert).
- Audit logs contain `INSERT` or `UPDATE` records for each processed player.

---

## Scenario 3: Verify Public Catalog & Detail REST API

### Objective
Validate that `GET /players` returns the summary DTO (name, club, league, position only) with pagination and filters, while `GET /players/{id}` returns the full attribute detail DTO.

### Validation Steps

#### 3.1 Paginated Catalog List (Summary DTO)
```bash
curl -s "http://localhost:8080/players?page=1&limit=5" | jq .
```
**Expected Outcome**:
HTTP 200 containing pagination metadata (`page`, `limit`, `total`, `totalPages`) and an array of items where each item contains **only**:
- `id`
- `name`
- `club`
- `league`
- `position`

#### 3.2 Filtering by League, Club, and Position
```bash
# Filter by League
curl -s "http://localhost:8080/players?league=PL&limit=5" | jq .

# Filter by Position
curl -s "http://localhost:8080/players?position=Goalkeeper&limit=5" | jq .

# Combined filter
curl -s "http://localhost:8080/players?league=PL&position=Midfielder&club=Arsenal" | jq .
```
**Expected Outcome**:
HTTP 200 containing only players matching all supplied filter criteria.

#### 3.3 Player Detail Endpoint (Full DTO)
```bash
# Fetch player with ID 1
curl -s "http://localhost:8080/players/1" | jq .
```
**Expected Outcome**:
HTTP 200 returning full player profile:
- `id`, `externalId`, `name`, `club`, `league`, `leagueCode`, `position`, `dateOfBirth`, `nationality`, `shirtNumber`, `active`, `createdAt`, `updatedAt`.
- Missing optional values (e.g. unassigned shirt number) return graceful nulls without crashing.

---

## Scenario 4: Frontend Catalog & Detail Page Walkthrough

### Objective
Verify the React + TypeScript frontend displays the player catalog with filtering and pagination, and allows viewing individual player details on a dedicated page.

### Validation Steps
1. Install dependencies and start Vite dev server:
   ```bash
   cd frontend
   npm install
   npm run dev
   ```
2. Open `http://localhost:5173` in a web browser.
3. **Catalog Listing Page**:
   - Verify table/grid renders players displaying name, current team (club), league, and position.
   - Use the filter bar to select a league (e.g., "La Liga") and verify the list updates.
   - Type in the position filter (e.g., "Defender") or search input (e.g., player name) and verify reactive filtering.
   - Click page 2 in the pagination controls and verify subsequent players load.
4. **Player Detail Page**:
   - Click on any player row or "View Details" button.
   - Verify the URL transitions to `/players/:id` (or detailed view).
   - Verify all player attributes are displayed: name, club, league, position, date of birth, nationality, shirt number (or "Unassigned"), active status badge, and last updated date.
   - Click the "Back to Catalog" button; verify return to the player catalog with filters preserved.

### Expected Outcome
- UI meets all styling and functional requirements using BEM CSS classes.
- Zero direct axios calls outside `src/api/players.ts`.

---

## Scenario 5: External Service Downtime Resiliency

### Objective
Prove that player catalog browsing operates with 100% availability during external provider downtime.

### Validation Steps
1. Unset the external API key or disconnect network connectivity to `football-data.org`:
   ```bash
   export FOOTBALL_DATA_API_KEY="invalid-dummy-key"
   ```
2. Browse the frontend player catalog (`http://localhost:5173`) and apply filters.
3. Click through to view player detail pages.

### Expected Outcome
- Catalog and player detail pages load immediately from PostgreSQL with zero errors presented to the user.
- 0 outgoing network requests to football-data.org occur during user catalog browsing (SC-002, SC-004).

---

## Scenario 6: Automated Quality Gates & Tests

### Objective
Ensure all backend and frontend unit tests, integration tests, and linters pass cleanly.

### Validation Steps
1. Run backend tests under race detector:
   ```bash
   cd backend
   go test -v -race ./...
   ```
2. Run backend pre-commit hook:
   ```bash
   ./scripts/pre-commit.sh
   ```
3. Run frontend quality gates:
   ```bash
   cd frontend
   npm test
   npm run typecheck
   npm run lint
   ```

### Expected Outcome
- All backend test packages report `ok`.
- Linter reports zero issues.
- Frontend test suite passes, and TypeScript compiler reports zero errors.
