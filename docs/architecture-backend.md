# Backend architecture

Structural map of `backend/`. Use it to answer "which layer owns this?" before reading code.
Every claim below is anchored to a file; line numbers are from the current tree.

Inventory (126 non-test Go files: 1 under `cmd/`, 125 under `internal/`):

```bash
find backend/cmd backend/internal -name '*.go' -not -name '*_test.go' | sort
```

The same command returns the files listed in the package map in section 4.

## 1. Layers

| Path | Responsibility | May import | Must not do |
| --- | --- | --- | --- |
| `cmd/server` | Load configuration, open the pool, verify the schema, build containers, start and stop background jobs. | Everything, as the composition root (`cmd/server/main.go:69-159`). | Contain business rules or SQL. |
| `internal/server` | Route table, access-level enum, mux assembly, HTTP server lifecycle. | `controller`, `middleware`, `httphandler`, `model`, `logger`, `server/docs` (`server/server.go:11-13`, `server/routes.go:8-10`). | Touch the database or validate a payload. |
| `internal/middleware` | Authentication, authorization, actor propagation, correlation ID, CORS, access log. | `adapters` (claims and token kind), `httphandler`, `logger`, `model` (privilege) (`middleware/authentication.go:9-11`). | Query the database or own domain authorization. |
| `internal/httphandler` | The only JSON encode/decode path, the `Endpoint` type, the `APIError` wrapper. | `logger` only (`httphandler/wrapper.go:4-9`). | Import `model`, repositories, or services. |
| `internal/controller` | Parse path/query/body, call one service, translate errors to HTTP status. | `internal/dto`, `controller/dto`, `service`, `model`, `middleware`, `httphandler` (`controller/market.go:9-13`). | Open transactions or hold business rules. |
| `internal/controller/dto` | Request/response shapes for auth, market, P2P; input validation and decimal parsing. | `model`, `service` (`controller/dto/market.go:11-12`). | Touch persistence or `net/http`. |
| `internal/service` | Business rules, orchestration, authorization against domain data, response assembly for player/sync/score. | `model`, `dto`, `adapters`, `logger`, `persistence/repository` (`service/market.go:3-11`). | Import `net/http` or write SQL. |
| `internal/model` | Domain types, invariants, enumerations, pure calculations. | Standard library only (`model/credits.go:1`, `model/matching.go:3`). | Import any other internal package, carry JSON tags, or perform I/O. |
| `internal/persistence/repository` | Transactions, lock ordering, multi-DAO workflows, sentinel translation. | `model`, `persistence/dao`, `database/sql` (`persistence/repository/p2p.go:3-11`). | Know about HTTP, DTOs, or services. |
| `internal/persistence/dao` | One SQL statement per method, row scanning, driver-error translation. | `model`, `database/sql` (`persistence/dao/p2p_session_sql.go:3-10`). | Begin transactions, decide business outcomes, or import repositories. |
| `internal/adapters` | JWT, bcrypt, football-data.org, WhoScored clients. | `configuration`, `model`, third-party libraries (`adapters/container.go:4-8`). | Leak provider payloads past their own DTOs. |
| `internal/dto` | Player, scoring, sync, ranking, pagination DTOs and credit rendering. | `model`, `net/http` (`dto/player_filter.go:22`). | Import `service` or persistence. |
| `internal/logger` | `slog` handler with Zerolog output and correlation ID from context. | Standard library plus `zerolog` (`logger/logger.go:4-8`). | Import domain packages. |
| `internal/configuration` | Environment parsing and validation. | Standard library (`configuration/configuration.go:3-10`). | Read the database. |
| `internal/server/docs` | Embedded Swagger UI and raw `/openapi.yaml`, mounted outside the route table. | `logger` (`server/docs/docs.go:8-14`). | Register business endpoints. |

## 2. Walkthrough: P2P session creation

A seller posts a session; the seller's tokens are reserved before the row is inserted.

1. **Route.** `POST /p2p/sessions` is declared `AccessAuthenticated` and bound to
   `s.controllers.P2P.Create()` (`backend/internal/server/routes.go:161-165`).
   `buildMux` wraps the endpoint (`backend/internal/server/routes.go:188`); `chainFor` adds
   `RequireAccessToken` for this level (`backend/internal/server/routes.go:198-199`).
2. **Middleware.** The access token is read, verified, type-checked, and turned into an
   `Actor` that is published on the context (`middleware/authentication.go:50-80`).
3. **Controller.** `Create` takes the actor, decodes the body, validates the request, parses
   the decimal price, and calls the service (`controller/p2p.go:30-57`).
4. **Request DTO.** `CreateP2PSessionRequest.Validate` enforces a positive player and quantity
   (`controller/dto/p2p.go:24-35`); `ParsedPrice` converts the decimal string through
   `CreditsDesdeDecimal` (`controller/dto/p2p.go:37-46`, `controller/dto/market.go:42-76`).
5. **Service.** `P2PService.Create` re-checks quantity and price, loads the player, rejects an
   inactive player, builds a `PENDING` session expiring in five minutes, and calls the
   repository (`service/p2p.go:57-90`).
6. **Repository.** `P2PRepository.Create` opens a transaction, locks the player row, locks the
   seller position, reserves the tokens, updates the position, inserts the session, writes the
   audit row, and commits (`persistence/repository/p2p.go:54-121`).
7. **DAO.** `P2PSessionSql.Insert` runs one `INSERT ... RETURNING id, token, created_at,
   expires_at` and scans the generated token back into the session
   (`persistence/dao/p2p_session_sql.go:21-62`).
8. **Postgres.** `p2p_sessions` supplies the UUID default, the status/quantity/price checks, and
   the FKs to `users` and `players` (`backend/db/init.sql:308-323`).
9. **Response.** The controller builds `/?token=<token>` and encodes
   `dto.P2PSessionResponseDesdeModelo` (`controller/p2p.go:62-64`,
   `controller/dto/p2p.go:75-110`).

## 3. Where does this change go?

| Change | Start here | Then |
| --- | --- | --- |
| New field on a domain entity | `internal/model/<type>.go` | `db/init.sql` column and constraint, DAO scan/insert, repository mapping, DTO conversion, `api/openapi.yaml`, contract test. |
| New business rule | `internal/model` if it is a pure invariant; `internal/service` if it needs a repository or actor | A model method keeps the rule testable without Postgres. Add a sentinel when the rule can fail. |
| New query | `internal/persistence/dao` (one method, one statement) | `internal/persistence/repository` wrapper, service interface, service call. |
| New error | `model` for a domain invariant, `service` for an orchestration outcome, `controller/dto` for request shape | Map it in the controller's `*StatusFor` function, add the status to `api/openapi.yaml`, record it in `docs/errors.md`. |
| New route | `internal/server/routes.go` with an explicit `AccessLevel` | Controller endpoint, controller-side service interface, OpenAPI operation with `x-access-level`, route test map. |
| New UI state | `docs/ui.md` and the owning page; no backend change | See `design-frontend`; backend only changes if the contract must grow. |
| Several layers at once | Write the contract delta first (`api/openapi.yaml`) | Then model → `init.sql` → DAO → repository → service → controller/DTO → route. Keep the route, DTO, and OpenAPI entry in the same change. |

## 4. Package map

Counts are production `.go` files from the inventory command.

- `cmd/` (1): composition root; no logic beyond wiring, schema check, and shutdown.
- `server` (3): route table, access enum, HTTP lifecycle.
- `middleware` (6): access/refresh token verification, privilege check, actor context, global HTTP middleware.
- `httphandler` (3): `Endpoint`, `Encode`/`Decode`, `APIError` and the panic-free error wrapper.
- `controller` (7): one REST controller per area (auth, player, scoring, sync, market, P2P) plus the container.
- `controller/dto` (3): auth, market, and P2P request/response types; validation and `CreditsDesdeDecimal`.
- `service` (14): domain orchestration, plus the sync scheduler, market maker, P2P expiry job, and containers.
- `model` (26): value types and pure functions for accounts, positions, orders, trades, quotes, scores, P2P, users, sync.
- `persistence/repository` (16): transaction boundaries and cross-DAO workflows.
- `persistence/dao` (21): one SQL statement per method, including the schema probe.
- `adapters` (7): 3 top-level (JWT, password, container) plus 2 football-data and 2 WhoScored files.
- `dto` (13): player, scoring, sync, ranking, pagination DTOs and `CreditsADecimal`.
- `logger` (2): Zerolog-backed `slog` handler and the context helpers.
- `configuration` (1): environment parsing and validation.
- `server/docs` (1): Swagger UI assets and the raw contract endpoint.
- `testsupport` (1) and `testutil` (1): shared test helpers; present in the inventory, not part of the runtime graph.

## 5. Known asymmetries (facts, not proposals)

- **Two DTO packages.** `internal/dto` (13 files) holds player, scoring, sync, ranking, and
  pagination shapes plus `CreditsADecimal` (`dto/credits.go:9`). `internal/controller/dto`
  (3 files) holds auth, market, and P2P shapes plus `CreditsDesdeDecimal`
  (`controller/dto/market.go:42`). Controllers pick one or the other per handler
  (`controller/player.go:9` versus `controller/market.go:9`).
- **Response DTOs are built in different layers.** The player, sync, and score services return
  `internal/dto` responses directly (`service/player.go:75`, `service/player.go:96`,
  `service/sync_run.go:93`, `service/score.go:114`). Auth, market, and P2P controllers build
  `controller/dto` responses at the edge (`controller/auth.go:44`, `controller/market.go:75`,
  `controller/p2p.go:64`).
- **Credit formatting is duplicated.** `CreditsADecimal` exists in `internal/dto`
  (`dto/credits.go:9`) and again in `internal/controller/dto` (`controller/dto/market.go:24`);
  only the latter package has `CreditsDesdeDecimal` (`controller/dto/market.go:42`).
- **Sentinels have five homes.** `model` (24 declarations), `service` (16), `persistence/repository`
  (4), `persistence/dao` (3), and `controller/dto` (10); `middleware` adds 5 package-private
  errors. Names collide across packages: `ErrNotFound`, `ErrPlayerInactive`,
  `ErrInvalidQuantity`, `ErrRateLimitExceeded`. See `docs/errors.md` for the per-layer table.
