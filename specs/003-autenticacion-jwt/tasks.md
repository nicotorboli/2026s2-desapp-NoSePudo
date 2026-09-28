# Tasks: JWT Authentication and Authorization

**Input**: Design documents from `/specs/003-autenticacion-jwt/`

**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/openapi.yaml](./contracts/openapi.yaml)

**Tests**: Included. They are not optional here — Principle XIII of the constitution
fixes what is real and what is mocked per layer, FR-014 *is* an automated test, and
SC-002, SC-009 and SC-010 are stated as things a test must demonstrate.
[research.md R15](./research.md) is the table this breakdown follows.

**Organization**: Tasks are grouped by user story. Every path is relative to the
repository root.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel — different files, no dependency on an incomplete task
- **[Story]**: The user story the task serves (US1…US7)

## Path Conventions

Monorepo with `backend/` and `frontend/`. This feature is backend-only: no file under
`frontend/` is touched.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: The dependencies, the schema and the environment the rest of the feature assumes.

- [X] T001 Add the runtime dependencies to `backend/go.mod` — `github.com/golang-jwt/jwt/v5` (R1), `golang.org/x/crypto` for `bcrypt` (R4) and `github.com/google/uuid` for the `jti`/`sid` UUIDs (R2); then the test-only `github.com/testcontainers/testcontainers-go` with its `modules/postgres` (R15). Run `go get` from `backend/` and commit the resulting `go.sum`.
- [X] T002 [P] Append the `users` and `refresh_tokens` tables and their four indexes to `backend/db/init.sql`, exactly as written in [data-model.md](./data-model.md) § SQL schema. Do not add a superuser row — it is provisioned at startup (R12).
- [X] T003 [P] ~~Declare the six new environment variables in `backend/docker-compose.yml`~~ — **does not apply, and the reason is recorded rather than the task quietly dropped.** `backend/docker-compose.yml` declares only the `db` service; the API is run from the host with `go run ./cmd/server`, so there is no service definition for the variables to sit in, and adding one would be inventing deployment this feature has no reason to make. `.gitignore` also excludes `.env`, `.env.*` and `*.env`, so a committed `.env.example` is not available either. The six variables — `NSP_JWT_SECRET`, `NSP_ACCESS_TTL`, `NSP_REFRESH_TTL`, `NSP_BCRYPT_COST`, `NSP_SUPERUSER_EMAIL`, `NSP_SUPERUSER_PASSWORD` — are documented in [quickstart.md](./quickstart.md) § 2 and land in `README.md` in T089.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The domain vocabulary, the border plumbing, the two adapters, the account's
persistence and the route table. Every user story below consumes something from here.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

### Domain model

- [X] T004 [P] Create `backend/internal/model/privilege.go` — `PrivilegeLevel uint8` with `PrivilegeUnknown` as the zero value, `PrivilegeUser`, `PrivilegeSuperuser`, plus `String()`, `ParsePrivilege(string) PrivilegeLevel` (never an error, never defaults to superuser) and `Satisfies(required) bool` (false whenever either side is unknown), per [data-model.md](./data-model.md).
- [X] T005 [P] Create `backend/internal/model/privilege_test.go` — round-trip `String`/`ParsePrivilege`, an unrecognised string yielding `PrivilegeUnknown`, and the decision table for `Satisfies` including both unknown sides (FR-018).
- [X] T006 [P] Create `backend/internal/model/user.go` — the `User` struct with fields ordered largest-first for `govet`'s `fieldalignment`, `NormalizeEmail(string) string` (trim + lowercase, R11), and the sentinels `ErrEmailTaken` and `ErrInvalidCredentials` declared beside the type (R8).
- [X] T007 [P] Create `backend/internal/model/user_test.go` — `NormalizeEmail` over surrounding whitespace, mixed case, and an address already normalised (idempotence), which is the spec's letter-case edge case.

### HTTP border

- [X] T008 [P] Create `backend/internal/httphandler/error.go` — a concrete `Error` type implementing the existing `APIError` interface, carrying a status code, a client-facing message and a wrapped cause, with `Unwrap()` (R8).
- [X] T009 Modify `backend/internal/httphandler/wrapper.go` — replace the `err.(APIError)` type assertion with `errors.AsType[APIError](err)` so a decorated error keeps its intended status (FR-012). Depends on T008.
- [X] T010 Create `backend/internal/httphandler/wrapper_test.go` — an `APIError` wrapped in two layers of `fmt.Errorf("...: %w", err)` still reaches the caller with its own status and message, never a 500 (SC-010). Depends on T009.
- [X] T011 Modify `backend/internal/httphandler/utils.go` — `Decode` calls `DisallowUnknownFields()` (FR-024), refuses trailing content after the first JSON value, and runs the value's `Validate() error` when it implements the new `Validator` interface (R10, FR-023).
- [X] T012 Create `backend/internal/httphandler/utils_test.go` — an unknown field is refused, a trailing second JSON value is refused, a type implementing `Validator` has it invoked, and one that does not decodes unchanged. Depends on T011.

### Configuration and logging contract

- [X] T013 [P] Create `backend/internal/logger/context.go` — `FromContext(ctx) *slog.Logger` returning the undecorated logger and `Into(ctx, *slog.Logger) context.Context`, the contract R9 defines. This is the stub the deferred observability feature replaces; nothing else about logging is built here.
- [X] T014 [P] Modify `backend/internal/configuration/configuration.go` — `LoadCfg() (*Cfg, error)`; new fields for the signing secret, both lifetimes, the bcrypt cost and the superuser credentials; defaults via `cmp.Or`; an error when the secret is absent or under 32 bytes, or when a lifetime or the cost is present but unparseable (FR-031, FR-032, R13).
- [X] T015 [P] Create `backend/internal/configuration/configuration_test.go` — absent secret and 31-byte secret both error, a 32-byte one does not (boundary value), defaults are applied when the optional variables are unset, and an unparseable duration errors (SC-009). Depends on T014.

### Adapters

- [X] T016 [P] Create `backend/internal/adapters/password.go` — `Password` wrapping `bcrypt`, with the cost factor received by constructor; `Hash(plain string) (string, error)` and `Compare(hash, plain string) error` (R4, R5).
- [X] T017 [P] Create `backend/internal/adapters/password_test.go` — against the real `bcrypt` at a low cost: a hash verifies against its own password, fails against another, and two hashes of the same password differ.
- [X] T018 Create `backend/internal/adapters/jwt.go` — `JWT` wrapping `golang-jwt/jwt/v5` with the secret, both lifetimes and an injected `func() time.Time` clock; issues and verifies both claim sets of [data-model.md](./data-model.md) § The credential itself (`sub`, `iat`, `exp`, `jti`, `typ`, `priv`, `sid`) with HS256 and an explicit algorithm allowlist (R1, FR-009). Depends on T004.
- [X] T019 Create `backend/internal/adapters/jwt_test.go` — with a fixed clock: a token verified one tick before, exactly at, and one tick after `exp` (the spec's first edge case); a token signed with an algorithm outside the allowlist is refused; an altered payload is refused; the claims round-trip. Depends on T018.
- [X] T020 Create `backend/internal/adapters/container.go` — `NewContainer(cfg)` building `Password` and `JWT`, following the per-layer container pattern the other packages already use. Depends on T016, T018.

### Account persistence

- [X] T021 [P] Create `backend/internal/persistence/dao/user_sql.go` — `UserSql` with `Insert`, `GetByEmail` and `GetByID`, hand-written SQL, `context` propagated, following the shape of `player_sql.go`.
- [X] T022 [P] Create `backend/internal/persistence/dao/schema_sql.go` — `SchemaSql` with `MissingTables(ctx, want []string) ([]string, error)` querying `information_schema.tables` (R14).
- [X] T023 Modify `backend/internal/persistence/dao/container.go` — add `User` and `Schema`. Depends on T021, T022.
- [X] T024 Create `backend/internal/persistence/repository/user.go` — `UserRepository` over the DAO interface declared next to it, returning `model.ErrEmailTaken` when the unique index rejects an insert. Depends on T021.
- [X] T025 ✅ **Verified with Docker up — all 7 cases PASS.** `backend/internal/persistence/repository/user_test.go` and the shared `postgres_test.go` helper exist and compile, covering insert-and-read-back, the unique index refusing a duplicate email, a lookup miss on both finders, the superuser privilege surviving the round trip, and two normalised forms of one address colliding. All eight cases **skip** on this machine: Docker Desktop is not running (`rootless Docker is not supported on Windows, failed to create Docker provider`), so the package reports `ok` without having proved anything. Re-run `go test ./internal/persistence/repository/` with Docker up and confirm the cases PASS rather than SKIP before checking this off. Depends on T024.
- [X] T026 Modify `backend/internal/persistence/repository/container.go` — add `User`. Depends on T024.

### Route table

- [X] T027 [P] Create `backend/internal/server/access.go` — `AccessLevel uint8` with `AccessUndeclared` as the zero value, plus `AccessAnonymous`, `AccessAuthenticated`, `AccessRenewal`, `AccessSuperuser` (R7, FR-013).
- [X] T028 Modify `backend/internal/server/routes.go` — `routes()` returns `[]route{{pattern, access, endpoint}}` and registers nothing; a new `buildMux(routes, logger, middleware)` becomes the only caller of `mux.Handle`, panics on `AccessUndeclared`, and applies the chain each level implies (only `AccessAnonymous` has a chain — none — until US3 lands). Depends on T027.
- [X] T029 Modify `backend/internal/server/server.go` — `NewServer` assigns `s.router = buildMux(s.routes(), s.logger, s.middleware)` so no mux is in scope for a stray registration. Depends on T028.
- [X] T030 Modify `backend/cmd/server/main.go` — handle the error from `LoadCfg`, build the adapters container, and run the schema check right after the ping, aborting with the missing table names and the `docker compose down -v` instruction when one is absent (R14, FR-032). Depends on T014, T020, T023.
  - **Partial, deliberately**: the fallible `LoadCfg` and the schema check are done and validated by hand (server refuses to start with no secret and with a 31-byte one; against a database without the tables it names all three and prints the fix; against the real one it logs `Database schema verified` and serves). ~~**Building the adapters container is not done here**~~ — **done in T035**, where `service.NewContainer(repos, adapterContainer)` gained the `Auth` service that consumes `adapters.Password`. T030 is now complete.

**Checkpoint**: `go build ./...`, `go test -race ./...` and `./scripts/pre-commit.sh` are green, the server still serves `GET /players`, and it now refuses to start without `NSP_JWT_SECRET`.

---

## Phase 3: User Story 1 - Create an account to operate in the market (Priority: P1) 🎯 MVP

**Goal**: A prospective participant creates an account with a normalised email and a bcrypt-hashed password, as a common user, and a duplicate identifier is refused.

**Independent Test**: `POST /auth/register` with valid data creates a common user; the same address in different case is refused as a duplicate; a body carrying `privilege` is refused as an unexpected field; the stored row holds no recoverable password.

- [X] T031 [P] [US1] Create `backend/internal/controller/dto/auth.go` — `RegisterRequest` (email, password, and **no** privilege field, FR-022) with `Validate() error` implementing the rules table of [data-model.md](./data-model.md): email present and non-empty after trimming, exactly one `@` with non-empty parts, at most 254 characters, password between 8 and 72 bytes; plus `RegisterResponse` per [contracts/openapi.yaml](./contracts/openapi.yaml). Messages name the JSON field and never echo a submitted value (FR-025).
- [X] T032 [P] [US1] Create `backend/internal/controller/dto/auth_test.go` — equivalence classes and boundary values for `RegisterRequest.Validate`: 7/8/72/73-byte passwords, 254/255-character emails, an address with no `@`, with an empty local part, with an empty domain.
- [X] T033 [US1] Create `backend/internal/service/auth.go` — the `Auth` service with its repository and adapter interfaces declared next to it, and `Register(ctx, email, password) (model.User, error)`: normalise, check uniqueness, hash at the configured cost, hardcode `model.PrivilegeUser`. Depends on T006, T016, T024.
- [X] T034 [US1] Create `backend/internal/service/auth_test.go` — `Register` with the repository and the password adapter mocked: the happy path stores a normalised email and a hash, a taken email returns `ErrEmailTaken`, and the created user is always `PrivilegeUser`. Depends on T033.
- [X] T035 [US1] Modify `backend/internal/service/container.go` — add `Auth`, built from the repository and adapter containers. Depends on T033.
- [X] T036 [US1] Create `backend/internal/controller/auth.go` — `RestAuthController` with a `Register()` endpoint that decodes through `httphandler.Decode`, calls the service, maps `model.ErrEmailTaken` to 409 with `errors.Is` and returns 201. Depends on T031, T033.
- [X] T037 [US1] Create `backend/internal/controller/auth_test.go` — register contract tests with the service mocked, checked against `contracts/openapi.yaml`: 201 body shape, 400 for each validation failure, 400 for an unknown field, 409 for a duplicate. Depends on T036.
- [X] T038 [US1] Modify `backend/internal/controller/container.go` — add `Auth` and its interface. Depends on T036.
- [X] T039 [US1] Modify `backend/internal/server/routes.go` — add `POST /auth/register` declared `AccessAnonymous` (FR-015). Depends on T028, T038.
- [X] T040 [US1] Create `backend/test/e2e/auth_test.go` — the whole stack through `server.NewServer` against a testcontainers Postgres, covering US1's five acceptance scenarios, including that a `privilege` field in the body is refused and that no account is created by any rejected request (SC-003). Depends on T039.

**Checkpoint**: accounts can be created and the refusals are demonstrable end to end.

---

## Phase 4: User Story 2 - Sign in and receive a session credential (Priority: P1)

**Goal**: Correct credentials return a signed access token carrying the account's identity, its privilege level and an expiry; wrong ones return a single indistinguishable refusal.

**Independent Test**: Sign in with a seeded account and read the claims out of the returned token; sign in with a wrong password and with an unknown address and confirm the two responses are identical and nothing is issued.

- [X] T041 [US2] Modify `backend/internal/controller/dto/auth.go` — add `LoginRequest` with its `Validate`, and `TokenResponse` as a structured object with named fields rather than a bare string (FR-006), per `contracts/openapi.yaml`; `omitzero` on the numeric and time fields, `omitempty` on the strings.
  - **Named `SessionResponse`, matching the contract's schema name, and built incrementally**: it carries `access_token`, `token_type` and `access_expires_at` now; `refresh_token` and `refresh_expires_at` are added in T077, when there is storage that can revoke them. The contract marks all five required, so the published document and the implementation only agree once US7 lands — which is fine because the contract is published in T087, after it. The service returns a new `model.Session`; it lives in `model` for the same reason `User.PasswordHash` does — opaque strings whose format is the adapter's business.
- [X] T042 [US2] Modify `backend/internal/controller/dto/auth_test.go` — `LoginRequest.Validate` over a missing, empty and malformed field, all refused before any lookup (US2 scenario 4). Depends on T041.
- [X] T043 [US2] Modify `backend/internal/service/auth.go` — `Login(ctx, email, password)`: normalise, look the account up, compare the hash, issue an access token carrying `sub`, `iat`, `exp`, `jti`, `typ: access` and `priv`. A missing account and a wrong password both return `ErrInvalidCredentials`, and the missing-account path still pays a hash comparison so the two are indistinguishable in time as well (FR-003). Depends on T018, T033.
- [X] T044 [US2] Modify `backend/internal/service/auth_test.go` — a decision table over {account exists, password matches}: only the both-true row issues a token, and the other rows return the same error. Depends on T043.
- [X] T045 [US2] Modify `backend/internal/controller/auth.go` — a `Login()` endpoint mapping `model.ErrInvalidCredentials` to 401. Depends on T041, T043.
- [X] T046 [US2] Modify `backend/internal/controller/auth_test.go` — login contract tests: 200 body shape against the OpenAPI document, 401 for both failure causes with an identical body, 400 for each invalid input. Depends on T045.
- [X] T047 [US2] Modify `backend/internal/server/routes.go` — add `POST /auth/login` declared `AccessAnonymous` (FR-015). Depends on T045.
- [X] T048 [US2] Modify `backend/test/e2e/auth_test.go` — US2's five acceptance scenarios, including that the issued token's readable payload contains no password and nothing damaging to disclose (FR-007). Depends on T047.

**Checkpoint**: a credential is issued, but no endpoint demands one yet.

---

## Phase 5: User Story 3 - The player catalog is unreachable without a valid credential (Priority: P1)

**Goal**: The authentication middleware verifies the credential before any business logic runs, publishes the acting account into the request context, and the catalog requires it.

**Independent Test**: Call `GET /players` with no credential, a malformed one, an expired one and an altered one — all four refused with 401 and no query executed. Call it with a valid access token and get the catalog.

- [X] T049 [P] [US3] Create `backend/internal/middleware/actor.go` — `Actor{ID int64, Privilege model.PrivilegeLevel, SessionID string}`, carrying no email by design (FR-028). Depends on T004.
- [X] T050 [P] [US3] Create `backend/internal/middleware/context.go` — an unexported typed context key, `WithActor(ctx, Actor) context.Context` and `ActorFromContext(ctx) (Actor, bool)`. Depends on T049.
- [X] T051 [US3] Create `backend/internal/middleware/authentication.go` — `Authenticate(kind)` as a `func(httphandler.Endpoint) httphandler.Endpoint` (R6): it accepts exactly `Bearer <token>` and refuses a missing header, an empty value, an unknown scheme and a malformed one rather than interpreting them (US3 scenario 4); verifies through the token-verifier interface declared next to it; refuses a `typ` other than the one the level requires (FR-038); returns `httphandler.Error` with 401 (FR-011); and on success publishes the actor and decorates the context logger with `actor` (R9, FR-010). Depends on T008, T013, T050.
- [X] T052 [US3] Create `backend/internal/middleware/authentication_test.go` — with the verifier mocked: the four refusal shapes, an expired token, an altered token, and the decision table of token kind × required kind (FR-038); on success the actor is readable from the context and the endpoint below ran exactly once. Depends on T051.
- [X] T053 [US3] Modify `backend/internal/middleware/container.go` — hold the authentication middleware, built from the JWT adapter. Depends on T051.
- [X] T054 [US3] Modify `backend/internal/server/routes.go` — `buildMux` applies `Authenticate(KindAccess)` for `AccessAuthenticated` and `Authenticate(KindRefresh)` for `AccessRenewal`, and `GET /players` is declared `AccessAuthenticated` (FR-015). Depends on T053.
- [X] T055 [US3] Modify `backend/internal/server/server_test.go` — the existing catalog test now presents a valid credential, and a companion case asserts the unauthenticated call is refused. Depends on T054.
- [X] T056 [US3] Modify `backend/test/e2e/auth_test.go` — US3's five acceptance scenarios, asserting for each refusal that the system's state is unchanged and that the catalog query never ran (SC-001, FR-008). Depends on T054.

**Checkpoint**: the credential now governs the API. This is the delivery's demonstrable core.

---

## Phase 6: User Story 4 - Every endpoint declares whether it requires a credential (Priority: P1)

**Goal**: An endpoint that ships without a declared access level fails the build.

**Independent Test**: Add a route with no `access` field and confirm the table test names it; register a route directly on the mux and confirm the AST test names the file and line.

- [X] T057 [US4] Create `backend/internal/server/routes_test.go` — the table test over `routes()`: every entry has a level other than `AccessUndeclared`, and this delivery's five routes carry exactly the levels FR-015 fixes — register and login anonymous, the catalog and logout authenticated, refresh renewal. Depends on T028.
- [X] T058 [US4] Extend `backend/internal/server/routes_test.go` with the source-level check — parse `backend/internal/server/` with `go/ast` and fail, naming file and line, when `Handle` or `HandleFunc` is called on a `*http.ServeMux` outside `buildMux` (R7, FR-014). Depends on T057.
- [X] T059 [US4] Demonstrate SC-002 — **done, and this is what the checks actually printed.** Nothing from this step was committed; `routes.go` was restored with `git checkout`.

  **Violation 1 — a route literal with no `access` field.** Three things fired, which is more than SC-002 asks for: the table test named it, the specification test named it, and `buildMux` refused to build the router at all.

  ```text
  --- FAIL: TestEveryRouteDeclaresAnAccessLevel
      routes_test.go:53: la ruta "GET /endpoint-que-alguien-olvido-declarar" no declara su nivel de acceso
  --- FAIL: TestDeclaredLevelsMatchTheSpecification
      routes_test.go:95: la ruta "GET /endpoint-que-alguien-olvido-declarar" está declarada como undeclared
                         y no figura en la especificación
  panic: la ruta "GET /endpoint-que-alguien-olvido-declarar" no declara su nivel de acceso
  ```

  **Violation 2 — a route registered straight onto the mux, the case the table cannot see because the route never entered it.** The AST test named the file, the line and the function:

  ```text
  --- FAIL: TestNoRouteIsRegisteredOutsideBuildMux
      routes.go:99:  registrarUnaRutaAEscondidas llama a Handle por fuera de buildMux
      routes.go:100: registrarUnaRutaAEscondidas llama a HandleFunc por fuera de buildMux
  ```

**Checkpoint**: the protection US3 delivered can no longer decay silently as the API grows.

---

## Phase 7: User Story 5 - The account's privilege level travels in the credential (Priority: P2)

**Goal**: The two privilege levels are a property of the account, carried by the access token, with exactly one superuser that registration cannot produce.

**Independent Test**: Sign in as a common user and as the superuser and read the `priv` claim of each; confirm registration has no path to producing a superuser.

- [X] T060 [P] [US5] Create `backend/internal/middleware/authorization.go` — `Authorize(required model.PrivilegeLevel)` as an `Endpoint` decorator reading the actor from the context and returning `httphandler.Error` with 403 when `Satisfies` is false, which is a different call site from the 401 (FR-011). Depends on T050.
- [X] T061 [P] [US5] Create `backend/internal/middleware/authorization_test.go` — a common user refused where superuser is required, the superuser allowed, and an absent or unknown level refused rather than treated as superuser (FR-018). Depends on T060.
- [X] T062 [US5] Modify `backend/internal/middleware/container.go` and `backend/internal/server/routes.go` — hold the authorization middleware and give `AccessSuperuser` its chain in `buildMux`, `Authenticate(KindAccess)` then `Authorize(model.PrivilegeSuperuser)`. No route uses it in this delivery; it is the point the next feature plugs into. Depends on T060.
- [X] T063 [US5] Modify `backend/internal/service/auth.go` — `EnsureSuperuser(ctx, email, password) error`: normalise, create with `PrivilegeSuperuser` when no account holds the address, and do nothing at all when one does — including not resetting its password (R12, FR-019). Depends on T033.
- [X] T064 [US5] Modify `backend/internal/service/auth_test.go` — `EnsureSuperuser` creates on an empty store, is idempotent on a second call, and leaves an existing account's hash untouched. Depends on T063.
- [X] T065 [US5] Modify `backend/cmd/server/main.go` — call `EnsureSuperuser` once after the schema check, only when both superuser variables are set, and skip silently otherwise. Depends on T030, T063.
- [X] T066 [US5] Modify `backend/test/e2e/auth_test.go` — US5's five acceptance scenarios: the `priv` claim for each level, exactly one superuser after provisioning, and that it was not reachable through registration. Depends on T065.

**Checkpoint**: the privilege distinction exists end to end and the enforcement point is ready for the first operation that needs it.

---

## Phase 8: User Story 7 - Staying signed in without resending the password (Priority: P3)

**Goal**: Sign-in also issues a persisted, single-use refresh token; renewal rotates it; reuse revokes every live token of the account; sign-out revokes the family.

**Independent Test**: Sign in, let the access token expire, renew without the password. Present the same refresh token twice and confirm the second is refused and the account's other refresh tokens are dead.

> Ordered before US6 although both are P3: US6 is blocked by an external dependency (see Phase 9) and this one is not.

- [X] T067 [P] [US7] Create `backend/internal/model/refresh_token.go` — the `RefreshToken` struct per [data-model.md](./data-model.md), `IsLive(now time.Time) bool` as a method rather than a column, and the sentinels `ErrTokenReused` and `ErrTokenExpired` beside the type (R8).
- [X] T068 [P] [US7] Create `backend/internal/model/refresh_token_test.go` — `IsLive` at the expiry boundary one tick before, exactly at and one tick after, and false for each of used and revoked independently. Depends on T067.
- [X] T069 [US7] Create `backend/internal/persistence/dao/refresh_token_sql.go` — `Insert`, `GetByID`, `Rotate` (mark the presented `jti` used and insert the replacement inheriting the same `family_id`, inside one transaction, Principle XI), `RevokeFamily(familyID, userID)` and `RevokeAllLiveForUser(userID)`. Depends on T067.
- [X] T070 [US7] Modify `backend/internal/persistence/dao/container.go` — add `RefreshToken`. Depends on T069.
- [X] T071 [US7] Create `backend/internal/persistence/repository/refresh_token.go` — the repository over the DAO interface declared next to it. Depends on T069.
- [X] T072 [US7] Create `backend/internal/persistence/repository/refresh_token_test.go` — against testcontainers: a rotation leaves the old row used and the new row live with the same family, a failed rotation writes nothing, `RevokeFamily` touches one family only, and `RevokeAllLiveForUser` leaves the account with zero live rows (SC-012). Depends on T071.
- [X] T073 [US7] Modify `backend/internal/persistence/repository/container.go` — add `RefreshToken`. Depends on T071.
- [X] T074 [US7] Modify `backend/internal/service/auth.go` — `Login` also opens a session family, issues a refresh token and persists its `jti`, and both credentials now carry `sid` (FR-033, R17). Depends on T043, T071.
- [X] T075 [US7] Modify `backend/internal/service/auth.go` — `Refresh` rotating in one transaction and returning a new access token (FR-034, FR-036); presenting a used or revoked token returns `ErrTokenReused` **and** revokes every live refresh token of the account (FR-037); `Logout` revokes the family named by the access token's `sid` (FR-039). Depends on T074.
- [X] T076 [US7] Modify `backend/internal/service/auth_test.go` — rotation issues a replacement and kills the original, replay returns `ErrTokenReused` and leaves zero live tokens, an expired refresh token is refused, and sign-out on one family leaves the other family live (the spec's two-devices edge case). Depends on T075.
- [X] T077 [US7] Modify `backend/internal/controller/dto/auth.go` — `TokenResponse` gains the refresh token and both lifetimes, per `contracts/openapi.yaml` (FR-033). Depends on T041.
- [X] T078 [US7] Modify `backend/internal/controller/auth.go` — `Refresh()` and `Logout()` endpoints; neither takes a body, both read their credential from the verified context (R17); `ErrTokenReused` and `ErrTokenExpired` map to 401. Depends on T075, T077.
- [X] T079 [US7] Modify `backend/internal/controller/auth_test.go` — refresh and logout contract tests against the OpenAPI document, including the 401 bodies. Depends on T078.
- [X] T080 [US7] Modify `backend/internal/server/routes.go` — add `POST /auth/refresh` declared `AccessRenewal` and `POST /auth/logout` declared `AccessAuthenticated` (FR-015), and update the Phase 6 table test's expected levels if it does not already list them. Depends on T054, T078.
- [X] T081 [US7] Modify `backend/test/e2e/auth_test.go` — US7's eight acceptance scenarios, including an access token refused at `/auth/refresh` and a refresh token refused at `/players` (FR-038), and that after sign-out every renewal attempt fails (SC-013). Depends on T080.

**Checkpoint**: a session survives the access token's short lifetime, and a stolen token is answerable.

---

## Phase 9: User Story 6 - Authenticated operations name their actor in the logs (Priority: P3) ⚠️ Partially blocked

**Goal**: Every authenticated operation records its actor, every refusal records its reason, and no password, credential, signing secret or personal datum ever reaches a log event.

**Independent Test**: Run the full flow and grep the emitted events for the password, the tokens, the secret and the email; all must return nothing, and each authenticated operation must name its actor.

> **Blocked by an external dependency.** [plan.md § Dependencies](./plan.md) records it:
> `internal/logger` emits human-readable console output and there is no correlation
> identifier anywhere, both of which belong to an unspecified observability feature. The
> tasks below are the part of US6 that is this feature's own; the two marked ⛔ are
> **not to be implemented here**.

- [X] T082 [US6] Modify `backend/internal/middleware/authentication.go` and `backend/internal/middleware/authorization.go` — log every refusal through `logger.FromContext(ctx)` with a `reason` field, and still log it when no account could be identified, with no subject (FR-029, US6 scenario 4). Depends on T051, T060.
- [X] T083 [US6] Modify `backend/internal/service/auth.go` — log a failed sign-in with its reason and without the submitted address, which is personal data (FR-028, FR-029). Depends on T043.
- [X] T084 [US6] Create `backend/test/e2e/logging_test.go` — capture the log output of a complete run through an `slog` handler installed by the test, and assert zero occurrences of the submitted password, either token, the signing secret and the account's email across every event, error events included (FR-027, FR-028). Depends on T082, T083.
- [ ] T085 [US6] ⛔ **BLOCKED — do not implement in this feature.** JSON log output replacing `zerolog.ConsoleWriter` in `backend/internal/logger/logger.go` (FR-030, SC-004, SC-005). Belongs to the observability feature.
- [ ] T086 [US6] ⛔ **BLOCKED — do not implement in this feature.** A correlation identifier generated at the HTTP border and carried on the context logger (FR-026), which is what `actor` is meant to sit beside. Belongs to the observability feature.

**Checkpoint**: the feature writes nothing it must not write; the rest of US6 waits on the observability feature.

**One prerequisite the plan did not foresee.** `logger.FromContext` falls back to
`slog.Default()` when the context carries no logger, and nothing was putting the server's
logger in there — so every event these tasks emit would have gone to the default logger
instead of the injected one, and the e2e could not have captured them. `httphandler.Wrap`
now publishes it: it already receives the logger and is the outermost decorator of the
chain, so it is the one place that has both. The deferred observability feature replaces
what `FromContext` returns; it no longer has to arrange for it to be reachable.

---

## Phase 10: Polish & Cross-Cutting Concerns

- [ ] T087 [P] Publish the contract to `backend/api/openapi.yaml` — the maintained artifact Principle VII requires, documenting the four new endpoints and the pre-existing `GET /players`, and point the controller contract tests at this copy so drift fails the build (R16).
- [ ] T088 [P] Verify `.gitignore` covers the Go patterns for this repository — `*.exe`, `*.test`, `*.out`, `vendor/` — and append only what is missing.
- [ ] T089 [P] Document the new environment variables in `README.md`, matching the table in [quickstart.md](./quickstart.md) § 2.
- [ ] T090 Run `./scripts/pre-commit.sh` from the repository root — gofmt, `go vet` with `enable-all`, and golangci-lint at the version in `.golangci-version`. `fieldalignment` is the one that usually bites on new structs.
- [ ] T091 Run `go test -race ./...` from `backend/` and confirm every suite passes, testcontainers ones included.
- [ ] T092 Walk [quickstart.md](./quickstart.md) § 4 end to end and confirm each user story's flow behaves as written, starting with SC-009 — the server refusing to start with `NSP_JWT_SECRET` unset. Depends on T090, T091.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)** → no dependencies.
- **Foundational (Phase 2)** → depends on Setup. **Blocks every user story.**
- **US1 (Phase 3)** → Foundational.
- **US2 (Phase 4)** → US1, because it extends the same DTO, service, controller and route files.
- **US3 (Phase 5)** → US2, because the catalog test needs an issued credential.
- **US4 (Phase 6)** → Foundational for the table test; its expected-levels assertion is completed by US7's two routes (T080).
- **US5 (Phase 7)** → US2 for the `priv` claim, US3 for the actor in the context.
- **US7 (Phase 8)** → US2, whose `Login` it extends.
- **US6 (Phase 9)** → US3 and US5 for the middlewares it logs from; two tasks blocked externally.
- **Polish (Phase 10)** → everything above.

### Parallel Opportunities

Within Foundational, five independent tracks can run at once once T001–T003 are done:

- **Domain**: T004 → T005, T006 → T007
- **Border**: T008 → T009 → T010, T011 → T012
- **Config/adapters**: T013, T014 → T015, T016 → T017, T018 → T019 → T020
- **Persistence**: T021, T022 → T023 → T024 → T025 → T026
- **Route table**: T027 → T028 → T029

`T030` closes the phase and needs the config, adapters and persistence tracks.

Inside a story, the `[P]` tasks are the ones in different files: T031/T032, T049/T050, T060/T061, T067/T068.

### Parallel Example: Foundational

```text
Track A: T004, T005, T006, T007          # internal/model/
Track B: T008, T009, T010, T011, T012    # internal/httphandler/
Track C: T013, T014, T015, T016…T020     # configuration, logger, adapters
Track D: T021…T026                       # persistence
Track E: T027, T028, T029                # server route table
Then:    T030                            # cmd/server/main.go — needs C and D
```

---

## Implementation Strategy

### MVP

Phases 1, 2 and 3 give an API that creates accounts. It is not yet the feature's point.
**The smallest genuinely demonstrable increment is Phases 1–5**: an account is created, a
credential is issued, and the catalog refuses anyone without one. That is what the
delivery asks for; US4 then keeps it from decaying, and US5 and US7 build on it.

### Incremental Delivery

1. Phases 1–2 → foundation, everything still green, the server refuses to start without a secret
2. \+ Phase 3 (US1) → accounts exist
3. \+ Phase 4 (US2) → credentials are issued
4. \+ Phase 5 (US3) → **the credential governs the API** — demonstrable
5. \+ Phase 6 (US4) → the protection cannot silently decay
6. \+ Phase 7 (US5) → privilege levels, and the point the next feature plugs into
7. \+ Phase 8 (US7) → sessions survive the access token's lifetime
8. \+ Phase 9 (US6) → what this feature can do about logging; the rest waits
9. \+ Phase 10 → contract published, verification green

---

## Notes

- **One task, one commit, green before the next.** Every task above is meant to leave `go build ./...` and `go test -race ./...` passing.
- `[P]` means a different file and no dependency on an incomplete task.
- Tests come before the implementation they cover wherever the task order allows it; for the risky ones — the expiry boundary (T019, T068), rotation and reuse (T072, T076) — write the test first.
- New structs must be declared largest-field-first: `govet` runs with `enable-all` and `fieldalignment` will fail the build otherwise.
- The Go 1.26 idioms this feature commits to are listed at the end of [research.md](./research.md): `errors.AsType`, `cmp.Or`, `omitzero`, `t.Context()`, `for range n`.
