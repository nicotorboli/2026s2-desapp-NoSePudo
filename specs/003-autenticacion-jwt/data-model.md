# Phase 1 Data Model: JWT Authentication and Authorization

**Feature**: `003-autenticacion-jwt` | **Date**: 2026-09-28 | **Plan**: [plan.md](./plan.md)

The entities the spec names, the form and the **layer** each one takes, the SQL schema
behind them, and the one state machine this feature introduces. Decisions and their
rationale are in [research.md](./research.md); this document is the shape.

Not all of them are domain models. The spec's Key Entities list names concepts, not
packages: `User`, `PrivilegeLevel` and `RefreshToken` are domain — they cross a
repository boundary and hold invariants of their own — while `Actor` and `AccessLevel`
only mean something at the HTTP border and live there. Each section below says which and
why.

---

## Entities

### User — `internal/model/user.go`

The market participant. The spec's **Account**.

| Field | Type | Notes |
|---|---|---|
| `ID` | `int64` | Assigned by the database. Zero until persisted. |
| `Email` | `string` | Normalised: trimmed and lowercased (R11). Unique. |
| `PasswordHash` | `string` | bcrypt digest. Never leaves the domain, never appears in a DTO, never reaches a log. |
| `Privilege` | `PrivilegeLevel` | Derived from the account, never from client input (FR-016). |
| `Active` | `bool` | Whether the account may operate. Always `true` in this delivery; the column exists because the spec names it as part of the entity. |
| `CreatedAt` | `time.Time` | |

Struct fields are declared largest-first, as `govet`'s `fieldalignment` requires.

**Validation rules** (enforced at the border by the DTO, per R10):

Every `Validate` below is invoked by `httphandler.Decode`, not by the controller, so no
endpoint can skip it (R10).

| Rule | Where | Refusal |
|---|---|---|
| `email` present, non-empty after trimming | `dto.RegisterRequest.Validate` | 400 |
| `email` contains exactly one `@` with a non-empty local and domain part | `dto.RegisterRequest.Validate` | 400 |
| `email` at most 254 characters | `dto.RegisterRequest.Validate` | 400 |
| `password` between 8 and 72 bytes | `dto.RegisterRequest.Validate` | 400 |
| `email` not already in use | `service.Auth.Register` → unique index | 409 |
| no unexpected field in the body | `httphandler.Decode` | 400 |

The 72-byte upper bound is bcrypt's: it silently ignores anything past byte 72, so a
longer password would make two different passwords equivalent. Refusing it is honest.

**Invariants**:

- A self-registered account is always `PrivilegeUser` (FR-022). The service hardcodes it;
  the registration DTO has no privilege field at all, so no client value can reach it.
- `Email` is stored normalised and looked up normalised. The same function does both.

---

### PrivilegeLevel — `internal/model/privilege.go`

The spec's **Privilege level**. Exactly two values, plus an invalid zero.

```go
type PrivilegeLevel uint8

const (
    PrivilegeUnknown PrivilegeLevel = iota // zero value — never sufficient
    PrivilegeUser
    PrivilegeSuperuser
)
```

- `String()` renders `user` / `superuser`; `PrivilegeUnknown` renders `unknown`.
- `ParsePrivilege(string) PrivilegeLevel` returns `PrivilegeUnknown` for anything it does
  not recognise — never an error, and never a default of `PrivilegeSuperuser`.
- `Satisfies(required PrivilegeLevel) bool` is false whenever either side is
  `PrivilegeUnknown`.

FR-018 requires an absent or unrecognised level to be treated as insufficient. Making the
zero value mean "unknown" and making `Satisfies` refuse it means the failure mode of
every path — a missing claim, a typo, a forgotten field — is refusal.

Stored as `SMALLINT`, carried in the `priv` claim as its string form.

---

### RefreshToken — `internal/model/refresh_token.go`

The spec's **Renewal credential**, as the system persists it. The token string itself is
never stored (R2); this is the record that makes it revocable.

**A domain model**, even though sessions are not what the business is about. Two reasons,
and the first is decisive: it crosses a repository boundary, and the constitution says
what crosses one is a model and never a row. The second is that it owns invariants worth
testing without a database — `IsLive`, and the state machine below.

| Field | Type | Notes |
|---|---|---|
| `ID` | `uuid` as `string` | The token's `jti` claim. Primary key. |
| `FamilyID` | `uuid` as `string` | The session this token belongs to. Created at sign-in, inherited by every rotation, carried in the `sid` claim of both credentials. |
| `UserID` | `int64` | The account it was issued to. |
| `IssuedAt` | `time.Time` | |
| `ExpiresAt` | `time.Time` | `IssuedAt` + the configured refresh lifetime. |
| `UsedAt` | `*time.Time` | Set when exchanged. `nil` while unused. |
| `RevokedAt` | `*time.Time` | Set by sign-out or by reuse detection. `nil` while live. |

**Derived state** — a method, not a column, so the three flags cannot disagree:

```go
func (t RefreshToken) IsLive(now time.Time) bool {
    return t.UsedAt == nil && t.RevokedAt == nil && now.Before(t.ExpiresAt)
}
```

---

### Actor — `internal/middleware/actor.go`

The account identity attributed to an operation, read from the verified credential and
from nowhere else (FR-010).

| Field | Type | Notes |
|---|---|---|
| `ID` | `int64` | from the `sub` claim |
| `Privilege` | `model.PrivilegeLevel` | from the `priv` claim |
| `SessionID` | `string` | from the `sid` claim — the family sign-out revokes |

**Not a domain model**, despite the spec listing it among its key entities. The spec names
concepts; it does not assign layers. Outside an authenticated HTTP request an Actor means
nothing: it is what the authentication middleware extracts from a token it has just
verified. It is also a trimmed `User` — id and privilege, no email, no hash — so putting
it in `model` would add a projection that only the border has a use for, and would make
the domain aware of tokens.

It lives beside the middleware that produces it, together with the typed context key and
`middleware.ActorFromContext(ctx) (Actor, bool)` that reads it back. Controllers read it
there — both are HTTP-layer packages, so no inversion — and pass the fields a service
needs as explicit arguments, which is what keeps `service` from importing `middleware`.

It carries no email by design: FR-028 forbids personal data in logs, and the actor is what
the logs name.

---

### AccessLevel — `internal/server/access.go`

The spec's **Endpoint access level**. Not persisted — it is a property of a route, held
in the route table (R7).

```go
type AccessLevel uint8

const (
    AccessUndeclared AccessLevel = iota // zero value — a registration error
    AccessAnonymous
    AccessAuthenticated
    AccessRenewal
    AccessSuperuser
)
```

The middleware chain each level implies, applied by `buildMux` — the only function that
registers anything on the `ServeMux`:

| Level | Chain applied by `buildMux` |
|---|---|
| `AccessAnonymous` | none |
| `AccessAuthenticated` | `Authenticate(KindAccess)` |
| `AccessRenewal` | `Authenticate(KindRefresh)` |
| `AccessSuperuser` | `Authenticate(KindAccess)` → `Authorize(model.PrivilegeSuperuser)` |
| `AccessUndeclared` | panics at startup; fails the route table test |

`AccessSuperuser` has no route in this delivery. It is defined because FR-013 names four
levels and because the enforcement point it describes is what the next feature plugs into.

---

### The credential itself

Not a persisted entity — it is the JWT described in R1. Its claim set is the contract
between the issuer (`adapters.JWT`) and the verifier (the authentication middleware):

| Claim | Access token | Refresh token |
|---|---|---|
| `sub` | account id, decimal string | account id, decimal string |
| `iat` | issue instant | issue instant |
| `exp` | `iat` + access lifetime (15 min default) | `iat` + refresh lifetime (7 days default) |
| `jti` | UUIDv4 | UUIDv4 — **this is the persisted `RefreshToken.ID`** |
| `typ` | `access` | `refresh` |
| `priv` | `user` or `superuser` | absent |
| `sid` | the session family | the session family — **this is `RefreshToken.FamilyID`** |

The refresh token carries no `priv` because it grants no access to a resource; the
privilege travels on the access token that renewal issues, and is read from the account
at that moment.

Both credentials carry `sid`, and that is what lets sign-out take no parameter: the access
token the caller already presents names the session to end. A family id discloses nothing
— it is a random UUID meaningful only to this system — so it satisfies FR-007.

---

## SQL schema

Appended to `backend/db/init.sql` (R14). The same file seeds the testcontainers instance
used by repository tests.

```sql
CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    email         VARCHAR(254) NOT NULL UNIQUE,
    password_hash VARCHAR(60)  NOT NULL,
    privilege     SMALLINT     NOT NULL,
    active        BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         UUID        PRIMARY KEY,
    family_id  UUID        NOT NULL,
    user_id    BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    issued_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id
    ON refresh_tokens (user_id);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_live
    ON refresh_tokens (user_id)
    WHERE used_at IS NULL AND revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_family
    ON refresh_tokens (family_id)
    WHERE revoked_at IS NULL;
```

Notes on the choices:

- The unique index on `email` is what makes FR-021 a database guarantee rather than a
  check-then-insert race. The service still checks first so it can return 409 with a
  useful message, but the index is what actually decides.
- `password_hash` is `VARCHAR(60)` because a bcrypt digest is exactly 60 characters.
- `ON DELETE CASCADE` matters the day account deletion exists; today nothing deletes an
  account.
- `family_id` is not a foreign key: a family has no row of its own. It is the name of a
  chain of rotations, created at sign-in and inherited by each replacement.
- The two partial indexes match the two revocation queries exactly — by account for reuse
  detection, by family for sign-out — so both read only live rows.
- Timestamps are `TIMESTAMPTZ`. Expiry comparisons on a `TIMESTAMP` without a zone are a
  bug waiting for a deployment in another region.
- The schema contains no superuser row. It is provisioned at startup (R12), because
  seeding one here would mean committing a bcrypt hash to the repository.

---

## Refresh token state transitions

```text
                    ┌──────────────────────────┐
                    │          LIVE            │
      sign-in  ───► │  used_at IS NULL         │
      renewal  ───► │  revoked_at IS NULL      │
                    │  now < expires_at        │
                    └────┬────────┬────────┬───┘
                         │        │        │
              presented  │        │        │  clock passes
              to renewal │        │        │  expires_at
                         ▼        │        ▼
                    ┌─────────┐   │   ┌─────────┐
                    │  USED   │   │   │ EXPIRED │
                    └────┬────┘   │   └─────────┘
                         │        │
           presented     │        │  sign-out, or
           a second      │        │  reuse detected on a
           time          │        │  sibling token
                         ▼        ▼
                    ┌──────────────────────────┐
                    │         REVOKED          │
                    │  every live token of the │
                    │  account, in one update  │
                    └──────────────────────────┘
```

| Transition | Trigger | Effect | Requirement |
|---|---|---|---|
| — → LIVE | successful sign-in | row inserted with a **new** `family_id` | FR-033 |
| LIVE → USED | presented to `/auth/refresh` | `used_at = NOW()`, replacement row inserted **inheriting the same `family_id`**, one transaction | FR-036 |
| LIVE → EXPIRED | time | no write; verification refuses it | FR-005 |
| LIVE → REVOKED | sign-out | `revoked_at = NOW()` on every live row of **that family**, named by the access token's `sid` | FR-039 |
| USED → (refused) + all LIVE → REVOKED | a used token presented again | every non-revoked row of **the account** gets `revoked_at = NOW()` | FR-037 |

Two things this diagram fixes:

- **USED is terminal and visible.** Deleting the row on use would make a replay
  indistinguishable from a token that never existed, and FR-037's theft response could
  never fire. The used row is the evidence.
- **The two revocations have different reach, on purpose.** Sign-out cuts one family,
  because the spec's edge case requires two sign-ins to be independent — ending the
  session on a phone must not end it on a desktop. Reuse detection cuts the whole
  account, because FR-037 says so and is right to: a thief holding one family's token may
  well hold another's, and the cost of being wrong is one extra sign-in.

---

## Requirement traceability

| Requirement | Where it lands |
|---|---|
| FR-002, FR-017 | credential claim table |
| FR-004 | `User.PasswordHash`, `password_hash VARCHAR(60)` |
| FR-016, FR-018 | `PrivilegeLevel` with invalid zero value |
| FR-019 | no superuser row in the schema; `service.Auth.EnsureSuperuser`, called once at startup |
| FR-021 | `users.email UNIQUE` |
| FR-022 | `RegisterRequest` has no privilege field; service hardcodes `PrivilegeUser` |
| FR-023, FR-024, FR-025 | the validation-rules table, `httphandler.Decode` |
| FR-013 | `AccessLevel` and its middleware chain table |
| FR-035, FR-036, FR-037, FR-039 | `refresh_tokens` and the state machine |
| FR-010, FR-026 | `Actor`, and the middleware that publishes it |
