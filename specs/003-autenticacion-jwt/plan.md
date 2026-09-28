# Implementation Plan: JWT Authentication and Authorization

**Branch**: `003-autenticacion-jwt` | **Date**: 2026-09-28 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/003-autenticacion-jwt/spec.md`

## Summary

The backend gains an account concept and a signed, self-contained session credential
that governs access to every endpoint it exposes. Accounts are created with a
normalised email and a bcrypt-hashed password; signing in returns a short-lived HS256
access token and a longer-lived, persisted, single-use refresh token. A new
`internal/middleware` layer verifies the credential before any business logic runs,
publishes the acting account into the request `context`, and is applied from a route
table in which every route declares its access level — anonymous, authenticated,
renewal or superuser — with the zero value invalid, so an undeclared endpoint cannot
reach production, and with an automated test that fails when a route is registered
outside that table.

One gap in the current code is fixed here: the HTTP error wrapper does not unwrap
decorated errors, so an authentication refusal raised below the controller surfaces as a
500. It is a one-line change and FR-012 and SC-010 are requirements of this spec.

Two further gaps are **not** fixed here and are tracked as an external dependency: the
logger writes human-readable console output rather than JSON, and there is no correlation
identifier at all, even though FR-026 assumes one already travels with every request.
Both belong to Principle IX and to a separate observability feature, not yet specified.
See [Dependencies](#dependencies) below for what this leaves blocked.

## Technical Context

**Language/Version**: Go 1.26.5 (`backend/go.mod` is the source of truth)

**Primary Dependencies**: `net/http` and `database/sql` from the standard library
(Principles I and II forbid a web framework and an ORM); `github.com/golang-jwt/jwt/v5`
for JWT; `golang.org/x/crypto/bcrypt` for password hashing; `github.com/lib/pq` as the
Postgres driver; `github.com/rs/zerolog` behind `log/slog`. New test-only dependency:
`github.com/testcontainers/testcontainers-go` with its `postgres` module, required by
Principle XIII for repository tests.

**Storage**: PostgreSQL 15 in Docker. Two new tables, `users` and `refresh_tokens`,
added to `backend/db/init.sql`.

**Testing**: `go test -race ./...` with the standard `testing` package and hand-written
mocks, the pattern already established in `internal/service/player_test.go`. Repository
tests run against a real Postgres started by testcontainers. Controller tests are
checked against `contracts/openapi.yaml`. The spec's acceptance scenarios are
implemented as e2e tests through `server.NewServer`.

**Target Platform**: Linux server in Docker, and local execution on the developer's
machine.

**Project Type**: Web service (Go REST API) inside a monorepo whose frontend this
feature does not touch.

**Performance Goals**: SC-007 — 95% of sign-in attempts answered in under one second
under the demonstration load. bcrypt dominates that cost, so the hashing cost factor is
injected configuration (FR-031) and is tuned to stay inside the budget.

**Constraints**: An access credential is verified by signature and expiry only, with no
per-request database lookup; what bounds the exposure is its short lifetime. Refresh
tokens are persisted precisely so they can be revoked before expiring. No rate limiting
and no lockout, accepted explicitly by the spec. TLS is a deployment concern and is not
built here.

**Scale/Scope**: First delivery. Five endpoints — account creation, sign-in, renewal,
sign-out, player catalog — one superuser account, and a user population in the tens for
the demonstration.

## Constitution Check

*GATE: checked before Phase 0 and re-checked after Phase 1 design. Result: **PASS**.*

| Principle | How this plan satisfies it |
|---|---|
| I. Go con `net/http` | No web framework. Middleware is plain function composition over the existing `httphandler.Endpoint` type and over `http.Handler`. Routing stays on `http.ServeMux` with method-aware patterns. |
| II. PostgreSQL con `database/sql` | `users` and `refresh_tokens` are read and written with hand-written SQL in DAOs. No ORM, no query builder. |
| III. Arquitectura en capas | New code lands in packages the constitution already names: `model` (User, PrivilegeLevel, RefreshToken), `persistence/dao` + `persistence/repository` (one repository per domain concept), `service` (Auth), `controller` + `controller/dto`, `middleware` (authentication, authorization, and the Actor they produce), `server` (route table). `context.Context` is propagated through every I/O layer, as it already is. |
| IV. Inyección de dependencias | Every new collaborator is received by constructor behind an interface declared next to its consumer. Concrete types — JWT adapter, bcrypt adapter, two DAOs, two repositories, the auth service, the auth controller, the middlewares — are instantiated only in the per-layer `NewContainer` functions that `cmd` wires together. |
| V. DTOs en el borde | Request and response DTOs are distinct types per operation; the domain `User` carries no serialization tags and never crosses the border. Conversion is `DesdeModelo` / `AModelo` inside the DTO. Encoding and decoding go through `httphandler.Encode` / `Decode`. |
| VI. Frontend | Not touched. The spec scopes this feature to the backend API boundary. |
| VII. OpenAPI | This plan produces [contracts/openapi.yaml](./contracts/openapi.yaml), documenting the four new endpoints **and** the pre-existing `GET /players`, which had never been documented. It is published to `backend/api/openapi.yaml` as the maintained artifact. |
| VIII. Seguridad JWT | HS256 with an explicit algorithm allowlist, verified in middleware before any business logic. The middleware puts the actor in the `context` and can demand a privilege level. Every route declares an access level and the zero value is invalid. Controls that depend on domain data — sign-out revoking a specific refresh token — compare the body's subject against the `context` actor inside the service. |
| IX. Observabilidad | **Partially deferred.** This feature does what is its own: the authentication middleware publishes the actor so that authenticated operations can name it, refusals are logged with their reason, and no password, token, signing secret or email ever reaches a log event. The JSON output and the correlation identifier belong to the observability feature this one depends on — see [Dependencies](#dependencies). |
| X. Auditoría transaccional | Out of scope here — no domain state changes beyond account creation and refresh-token lifecycle, and the spec requires no audit trail yet. The new rows carry the timestamps a later audit feature will need. |
| XI. Integridad de datos | Refresh-token rotation — invalidating the presented token and issuing its replacement — runs inside a single transaction. Reuse detection revokes the whole family in one statement. `users.email` carries a unique index; `refresh_tokens` is indexed by `user_id` and by expiry. |
| XII. Resiliencia | Not applicable: this feature reads no external data source and runs no batch process. |
| XIII. Testing | Unit tests per layer with the collaborator below mocked; repository tests against a real Postgres via testcontainers; adapter tests against the real libraries with fixed inputs and no network; controller tests against the OpenAPI contract including error codes; the spec's acceptance scenarios as e2e tests. Cases are chosen by equivalence class, boundary value — the expiry instant is tested one tick before, exactly at, and one tick after — and decision table where a rule combines conditions. |
| XIV. Semánticas | The specification directory is `specs/003-autenticacion-jwt/` and the active branch is `003-autenticacion-jwt`. They match. |

One deviation is recorded in Complexity Tracking below: the delivery of the new schema
through `db/init.sql` instead of a migration tool.

The placement of the JWT and bcrypt wrappers in `internal/adapters` was a deviation in an
earlier draft of this plan and is no longer one. Principle III described adapters as
"integración con servicios externos", which is narrower than the layer it actually names:
in ports and adapters, an adapter is any concrete implementation of a port against a
technology — a filesystem, a clock, a hashing function — and nothing in the pattern
requires a network on the other side. The principle was amended to say so, so that the
next adapter that is not a network service does not reopen the question.

## Project Structure

### Documentation (this feature)

```text
specs/003-autenticacion-jwt/
├── plan.md              # This file
├── research.md          # Phase 0 output — the decisions and why
├── data-model.md        # Phase 1 output — entities, schema, state transitions
├── quickstart.md        # Phase 1 output — how to run and validate the feature
├── contracts/
│   └── openapi.yaml     # Phase 1 output — the API contract
├── checklists/
│   └── requirements.md  # From /speckit-specify
├── spec.md
└── tasks.md             # Phase 2 output — created by /speckit-tasks, not here
```

### Source Code (repository root)

```text
backend/
├── api/
│   └── openapi.yaml                      # NEW — published contract, Principle VII
├── cmd/server/
│   └── main.go                           # MODIFIED — fallible config, schema check, calls EnsureSuperuser
├── db/
│   └── init.sql                          # MODIFIED — users, refresh_tokens
├── internal/
│   ├── adapters/                         # NEW package
│   │   ├── container.go                  # NEW
│   │   ├── jwt.go                        # NEW — HS256 issue/verify
│   │   ├── jwt_test.go                   # NEW
│   │   ├── password.go                   # NEW — bcrypt hash/compare
│   │   └── password_test.go              # NEW
│   ├── configuration/
│   │   └── configuration.go              # MODIFIED — secret, lifetimes, cost, superuser; fails fast
│   ├── controller/
│   │   ├── auth.go                       # NEW — register, login, refresh, logout
│   │   ├── auth_test.go                  # NEW
│   │   ├── container.go                  # MODIFIED — Auth controller
│   │   └── dto/
│   │       ├── auth.go                   # NEW — request/response DTOs + Validate/AModelo
│   │       └── auth_test.go              # NEW
│   ├── httphandler/
│   │   ├── error.go                      # NEW — concrete APIError carrying a cause
│   │   ├── utils.go                      # MODIFIED — DisallowUnknownFields + Validator (FR-023, FR-024)
│   │   └── wrapper.go                    # MODIFIED — errors.AsType unwrap (FR-012)
│   ├── middleware/
│   │   ├── container.go                  # MODIFIED — holds the middlewares
│   │   ├── authentication.go             # NEW — verify credential, publish actor
│   │   ├── authorization.go              # NEW — demand a privilege level
│   │   ├── actor.go                      # NEW — Actor: who the verified credential names
│   │   └── context.go                    # NEW — actor in context, typed key
│   ├── model/
│   │   ├── user.go                       # NEW — User, email normalisation, ErrEmailTaken
│   │   ├── privilege.go                  # NEW — PrivilegeLevel, parsing, zero value invalid
│   │   └── refresh_token.go              # NEW — RefreshToken, its state, its sentinels
│   ├── persistence/
│   │   ├── dao/
│   │   │   ├── container.go              # MODIFIED
│   │   │   ├── user_sql.go               # NEW
│   │   │   ├── refresh_token_sql.go      # NEW — rotation inside one transaction
│   │   │   └── schema_sql.go             # NEW — MissingTables, for the startup check
│   │   └── repository/
│   │       ├── container.go              # MODIFIED
│   │       ├── user.go                   # NEW
│   │       ├── user_test.go              # NEW — testcontainers
│   │       ├── refresh_token.go          # NEW
│   │       └── refresh_token_test.go     # NEW — testcontainers
│   ├── server/
│   │   ├── routes.go                     # MODIFIED — routes() describes, buildMux registers
│   │   ├── access.go                     # NEW — AccessLevel, zero value invalid
│   │   ├── routes_test.go                # NEW — the automated check (FR-014, SC-002)
│   │   └── server_test.go                # MODIFIED
│   └── service/
│       ├── auth.go                       # NEW — register, login, refresh, logout, EnsureSuperuser
│       ├── auth_test.go                  # NEW
│       └── container.go                  # MODIFIED
└── test/
    └── e2e/
        └── auth_test.go                  # NEW — the spec's acceptance scenarios
```

**Structure Decision**: The monorepo already separates `backend/` and `frontend/`, and
this feature is backend-only. Inside the backend no new layer is invented: every file
above lands in a package the constitution already names. The single new package is
`internal/adapters/`, which the constitution lists but which the codebase had not
needed until now.

## Dependencies

This feature depends on an **observability feature that does not exist yet** and whose
scope is deliberately left undecided here. Two things it must deliver:

1. **JSON log output.** `internal/logger` currently builds a `zerolog.ConsoleWriter`,
   which emits human-readable lines. FR-030 requires machine-parseable events.
2. **A correlation identifier**, generated at the HTTP border when the client does not
   supply one, carried in the `context`, and present on every event of the operation —
   Principle IX states this and FR-026 assumes it already happens.

**The contract this feature needs from it** is one function:

```go
func logger.FromContext(ctx context.Context) *slog.Logger
```

returning a logger already decorated with the correlation identifier. Given that, this
feature's authentication middleware adds `actor` to it and every layer logs through it,
which is what makes FR-026 nearly free — a layer that logs at all logs the actor and the
correlation identifier, with neither threaded through its signatures. Until it exists,
`logger.FromContext` can return the undecorated logger and nothing else changes.

**What is blocked until then**, stated plainly rather than quietly deferred:

| Blocked | Why |
|---|---|
| **US6** in full | Its entire premise is inspecting the emitted log events |
| FR-026 | The actor can be published, but there is no correlation identifier to sit beside |
| FR-030 | Console output is not machine-parseable |
| SC-004 | Scanning a run for leaked secrets needs parseable events |
| SC-005 | Same, for attributing operations to their actor |

**What is not blocked**: everything else. US1, US2, US3, US4, US5 and US7 — account
creation, sign-in, the protected catalog, the access-level declaration, privilege levels
and session renewal — are unaffected, along with FR-027, FR-028 and FR-029, which are
obligations about what this feature must *not* write and what it must log, and hold
whatever the output format is.

US6 is P3, the lowest priority in the spec, and the spec itself says the system is already
protected without it. So this does not block the delivery; it blocks one story and four
criteria, and it is recorded here so the gap is not discovered at the demonstration.

## Phase 0 — Research

Complete. See [research.md](./research.md): seventeen decisions, each with its rationale
and the alternatives rejected. No `NEEDS CLARIFICATION` markers remain.

## Phase 1 — Design & Contracts

Complete.

- [data-model.md](./data-model.md) — the entities the spec names, their fields and
  validation rules, the SQL schema, and the refresh token's state transitions.
- [contracts/openapi.yaml](./contracts/openapi.yaml) — the five endpoints with their
  declared access level, request and response shapes, and every error code.
- [quickstart.md](./quickstart.md) — how to start the stack, the environment variables
  the service now refuses to start without, and the runnable flows that demonstrate
  each user story.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|---|---|---|
| The new schema ships as statements appended to `backend/db/init.sql` rather than as a versioned migration | `init.sql` is the mechanism the repository already uses, it is mounted into the Postgres container's entrypoint, and the same file seeds the testcontainers instance used by repository tests. One mechanism, one source of truth. Its one bad symptom — a stale volume failing three layers down with `relation "users" does not exist` — is closed by the startup schema check (R14), which refuses to start and names the command that fixes it, exactly as FR-032 does for the signing secret. | A migration tool (goose, migrate, tern) is a new dependency and a new operational concept the constitution does not name, for a database that currently has one table and a development volume that can be recreated. It becomes worth it when there is data worth preserving; the startup check buys the time to get there without the confusing failure in between. The residual cost is recorded: applying the new schema locally requires `docker compose down -v`, which the startup check names and quickstart.md explains. |
