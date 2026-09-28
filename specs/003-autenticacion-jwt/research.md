# Phase 0 Research: JWT Authentication and Authorization

**Feature**: `003-autenticacion-jwt` | **Date**: 2026-09-28 | **Plan**: [plan.md](./plan.md)

Every decision the specification deliberately left to the plan, plus the three gaps in
the existing code that the feature's success criteria depend on. Nothing here is marked
`NEEDS CLARIFICATION`.

---

## R1 — Token format, algorithm and claims

**Decision**: JWS compact tokens signed with **HS256**, issued and verified by
`github.com/golang-jwt/jwt/v5`. The verifier is constructed with
`jwt.WithValidMethods([]string{"HS256"})`, an explicit allowlist. Claims:

| Claim | Meaning |
|---|---|
| `sub` | The account's identity — its database id, as a decimal string |
| `iat` | Issued at |
| `exp` | Expiry instant |
| `jti` | Unique token identifier (UUIDv4) |
| `typ` | `access` or `refresh` — the kind of credential |
| `priv` | `user` or `superuser` — the privilege level (access tokens only) |
| `sid` | The session family this credential belongs to (UUIDv4) — see R17 |

**Rationale**: HS256 is named by Principle VIII, and the library is named by the
constitution's Technical Constraints. `sub` is the account id, not the email: FR-028
forbids personal data in logs and FR-007 forbids anything damaging in the credential,
and the actor derived from `sub` is what gets logged. The registered claim names are
used where they exist so that the token is readable by any standard tooling; the two
custom claims carry the two things the standard has no name for.

`typ` is what makes FR-038 enforceable — an access token presented to renewal and a
refresh token presented to any other endpoint are both refused by the same check.

**Alternatives rejected**:

- **RS256/asymmetric.** Buys nothing when one process both issues and verifies, and
  adds key distribution and rotation to a first delivery. HS256 is what the constitution
  asks for.
- **`sub` = email.** Turns every log event that names its actor into personal data and
  makes the token disclose the account's email to anyone holding it. Both forbidden.
- **Privilege in a `scope` or `roles` array.** The spec is explicit that there are
  exactly two levels and that it is a single-valued property of the account (FR-016). A
  scalar claim states that truthfully; an array would invite a permission model the
  feature does not have.

---

## R2 — The refresh credential is a signed JWT whose `jti` is persisted

**Decision**: The refresh token is also a JWT, with `typ: refresh` and no `priv` claim.
Its `jti` is persisted as a row in `refresh_tokens` carrying the account, issue and
expiry instants, and the instants at which it was used or revoked. Verification is two
steps: verify the signature and expiry, then look the `jti` up and confirm the row is
neither used nor revoked.

**Rationale**: FR-035 requires a refresh credential the system can invalidate before it
expires, which demands persistence; FR-038 requires the two kinds to be distinguishable,
which the `typ` claim gives for free. Making it a JWT means one verification path, one
signing secret and one allowlist, and it means an expired or tampered refresh token is
refused without touching the database — fail-fast, and one fewer query per refused
attempt. The persisted row stores no secret: the `jti` is a public identifier, and the
signature is what proves the token was not forged.

**Alternatives rejected**:

- **Opaque random token, hashed at rest.** Entirely reasonable and marginally simpler to
  reason about, but it means two credential formats, two verification paths, and a
  database hit for every malformed token. Principle VIII names JWT for authentication;
  keeping both credentials in that format keeps the system to one mechanism.
- **Storing the whole refresh token string.** A stolen database would hand over usable
  credentials. The `jti` alone is enough to revoke, and is worthless to an attacker
  without the signing secret.
- **No persistence, relying on short lifetimes.** Directly contradicts FR-035 — a
  renewal credential the system cannot invalidate is stated as unacceptable — and makes
  sign-out (FR-039) and theft response (FR-037) impossible.

---

## R3 — Rotation and reuse detection

**Decision**: Renewal runs as one database transaction: mark the presented `jti` used,
insert the replacement row. Reuse detection is a single `UPDATE` that revokes every
non-revoked row of the account. Both live in `RefreshTokenSql`, the DAO for that table.

**Rationale**: Principle XI makes transactions the default, and rotation is precisely
the case that breaks without one: a crash between "mark used" and "insert replacement"
would sign the account out silently, and a concurrent double-refresh would otherwise
hand out two live tokens from one. Row-level locking on the presented `jti` is what
makes the second request see the first one's `used_at`, which is the signal FR-037 calls
theft.

The transaction lives in the DAO rather than the repository because it touches exactly
one table. The constitution assigns cross-table coordination to the repository; this is
not that, and pushing `*sql.Tx` up through the repository interface would leak the
storage engine into a layer that is supposed to speak models.

**Alternatives rejected**:

- **Delete the old row instead of marking it used.** Then a replayed token is
  indistinguishable from an expired or never-issued one, and FR-037's "treat reuse as
  evidence of theft" cannot fire. The used row is the evidence.
- **Repository-level transaction with the DAO accepting a `Querier`.** Correct and more
  flexible, and the right move the day an operation spans `users` and `refresh_tokens`.
  Today it is machinery with one caller.

---

## R4 — Password hashing

**Decision**: `golang.org/x/crypto/bcrypt`, with the cost factor injected from
configuration (FR-031) and defaulting to 12. Verification always runs
`bcrypt.CompareHashAndPassword`, including when no account matched the identifier —
against a fixed dummy hash — so that the response time does not reveal whether the
account exists.

**Rationale**: The library is named by the constitution. bcrypt is deliberately slow, and
that slowness is the whole point, which is why the cost belongs in configuration: the
demonstration budget in SC-007 is one second at the 95th percentile, and cost 12 sits
around 250 ms on ordinary hardware, leaving room. FR-003 demands that a wrong password
and a missing account be indistinguishable, and skipping the hash comparison for a
missing account would make them distinguishable by timing even though the response body
is identical.

**Alternatives rejected**:

- **argon2id.** Better on the merits and the modern default, but the constitution names
  bcrypt and names "una sola librería por responsabilidad".
- **Returning early when no account matched.** Faster, and it leaks the account's
  existence through a side channel the spec explicitly closes.

---

## R5 — JWT and bcrypt live behind interfaces in `internal/adapters`

**Decision**: Two adapters, `adapters.JWT` and `adapters.Password`, each constructed in
`cmd` with the configuration it needs and consumed by the auth service through an
interface declared next to the service.

**Rationale**: Principle IV requires the service to depend on an interface, and Principle
XIII requires the auth service to be testable with its collaborators mocked — which
means signing and hashing must be mockable, otherwise every service test pays a bcrypt
round and needs a real signing key. Each adapter is also the single place where the
injected configuration is applied, which is what FR-031 asks for.

Principle III used to describe adapters as "integración con servicios externos", which
would have made this a deviation. It was amended instead: in ports and adapters, an
adapter is any concrete implementation of a port against a technology, and the network is
not what defines one. What defines it is that the concrete implementation lives in one
place, with its configuration applied there, and that the consumer depends on the
interface. Both wrappers are that.

The amendment also reached Principle XIII, whose adapter rule assumed a network on the
other side: an adapter over a library is tested against the real library with fixed
inputs, and whatever makes the result unpredictable — here the clock, see R15 — is
injected so the boundary case can be written at all.

**Alternatives rejected**: calling the libraries directly from the service (untestable,
configuration scattered); putting them in `model` (makes the domain depend on
third-party libraries and on configuration).

---

## R6 — Two shapes of middleware, chosen by what each one needs

**Decision**:

- **Authentication and authorization** are `func(httphandler.Endpoint) httphandler.Endpoint`
  — decorators over the endpoint signature the codebase already has.
- **Correlation ID and request logging**, when the observability feature builds them, are
  `func(http.Handler) http.Handler`, applied once around the whole router. They are not
  part of this feature; the shape is recorded because it is the reason the split exists.

**Rationale**: An auth refusal is an error, and the codebase already has one way to turn
an error into a response: return it and let `httphandler.Wrap` encode it. Making auth an
`Endpoint` decorator means refusals get the same JSON error shape and the same logging as
every other failure, with no second error path to keep in sync.

The two transversal ones are the opposite case: they must run for every request, including
those that never reach an endpoint — a 404, a method mismatch. Wrapping the router is the
only place that sees those. That difference is also why they separate cleanly into another
feature: they attach at a different point and share no code with authentication.

**Alternatives rejected**: making everything an `http.Handler` decorator, which would
force the auth middleware to write its own error responses and duplicate the wrapper;
making everything an `Endpoint` decorator, which would leave unrouted requests with no
correlation ID and no log line.

---

## R7 — Declared access levels: a route table whose zero value is invalid

**Decision**: `internal/server/access.go` defines

```go
type AccessLevel uint8

const (
    AccessUndeclared AccessLevel = iota // zero value — always an error
    AccessAnonymous
    AccessAuthenticated
    AccessRenewal
    AccessSuperuser
)
```

`routes()` **describes** and does not register: it returns a `[]route` whose entries are
`{pattern, access, endpoint}`. `buildMux` is the only code that calls `mux.Handle`; it
panics on `AccessUndeclared` and otherwise applies the middleware chain the level implies.
`NewServer` connects the two:

```go
func (s *Server) routes() []route { return []route{ ... } }   // describes only

s.router = buildMux(s.routes(), s.logger, s.middleware)       // the only registrar
```

The shape matters as much as the check. With `routes()` registering into `s.router`, the
easiest way to add an endpoint is one more `s.router.Handle(...)` next to the others — the
path of least resistance leads straight past the table. With the mux built from the
returned slice, there is no `s.router` in scope to hang a stray route on: the field does
not hold anything until `buildMux` has run. Bypassing the table stops being the easy thing
and becomes a deliberate one.

The automated check FR-014 demands is `internal/server/routes_test.go`, in two parts:

1. A table test over the returned slice asserting every entry has a declared level, and
   that the levels of this delivery's five routes are the ones FR-015 fixes.
2. A source-level test that parses `internal/server/` with `go/ast` and fails, naming the
   file and line, if `Handle` or `HandleFunc` is called on a `*http.ServeMux` anywhere
   except inside `buildMux`.

**Rationale**: This is the story with the least obvious solution, because Go gives no help
here: an endpoint registered without a wrapper is simply public, and nothing — compiler,
linter, or ordinary test — says a word.

Three things carry the weight, in descending order of how much they buy per line spent.
The **invalid zero value** makes the level impossible to omit by accident: a `route`
literal that forgets `access` gets `AccessUndeclared`, which is an error rather than a
default of "public". The **shape of `routes()`** removes the easy bypass by leaving no mux
in scope to bypass it with. The **AST test** closes what remains — the deliberate or
uninformed `mux.Handle` — and is the only mechanism available for it, since Go exposes no
way to enumerate a `ServeMux`'s registered patterns and the source is therefore the only
other source of truth.

Strictly, FR-014 and SC-002 ask only for the first: the requirement is worded around an
endpoint "without a declared access level", which the table test catches. The AST test
covers the case the requirement does not name — an endpoint that never entered the table
at all — which is the same failure with a different cause, and the one that ordinary
review misses.

SC-002 asks for this to be demonstrated by deliberately adding an undeclared endpoint.
Both parts fail: the table test names the entry, the AST test names the line.

**Alternatives rejected**:

- **Enumerating the registered patterns from `http.ServeMux`.** There is no public API for
  it, in Go 1.26 or any earlier version.
- **A comment or struct tag convention.** Nothing enforces a comment.
- **Trusting a code-review checklist.** The story exists precisely because review is what
  fails to catch a forgotten wrapper.
- **A `grep` in `scripts/pre-commit.sh` instead of the AST test.** Three lines instead of
  seventy, and it runs in CI because CI runs that script. But it lives outside `go test`,
  so it never runs while someone is working on the package, and it breaks on a comment or
  a string that happens to contain `router.Handle`.
- **`panic` alone, without the tests.** It catches the forgotten level, but only for a
  route that went through `buildMux` — which is precisely the case the bypass test covers.

---

## R8 — Error taxonomy, and fixing the wrapper that swallows it

**Decision**: Three layers, each with one job.

- **Domain sentinels live beside the type that raises them**, not in a shared
  `errors.go`: `ErrEmailTaken` and `ErrInvalidCredentials` in `model/user.go`,
  `ErrTokenReused` and `ErrTokenExpired` in `model/refresh_token.go`. They know nothing
  about HTTP. A single `errors.go` collecting the errors of four unrelated concepts is a
  `utils.go` under another name, and it makes the one place every new error gets appended
  to without thought.

  Two refusals that look like they belong in that list do not: presenting an access token
  where a refresh token was required, and holding insufficient privilege. Both are decided
  by middleware, which is already at the border and raises `httphandler.Error` directly —
  a domain sentinel there would be a domain concept invented for an HTTP problem.
- `internal/httphandler/error.go` adds a concrete `httphandler.Error` implementing the
  existing `APIError` interface, carrying a status, a client-facing message and a wrapped
  cause.
- The **controller** maps sentinels to status codes with `errors.Is`, because translating
  to HTTP is the controller's job and the service's ignorance of HTTP is a constitutional
  rule. The **middleware** raises `httphandler.Error` directly, since it is already at the
  border.

`httphandler.Wrap` changes from a bare type assertion to
`errors.AsType[APIError](err)`, which walks the `%w` chain.

**Rationale**: The current wrapper does `err.(APIError)`, which matches only an
undecorated error. Any layer that wraps with `fmt.Errorf("...: %w", err)` — which every
layer does — turns an intended 401 into a 500. FR-012 and SC-010 forbid exactly that, and
the spec lists it as a prerequisite. `errors.AsType[T]` is the Go 1.26 form of
`errors.As`, and it accepts an interface type parameter, so the fix is a one-line change
in the wrapper.

The 401/403 split FR-011 requires falls out of this: the authentication middleware
returns 401, the authorization middleware returns 403, and they are different call sites,
not two branches of one check.

**Alternatives rejected**:

- **Domain errors implementing `APIError` directly.** Fewer moving parts, and it puts
  `StatusCode() int` on a domain type — the constitution's "el service/dominio no sabe de
  HTTP" ruled out.
- **Leaving the wrapper alone and never wrapping errors in inner layers.** Trades a
  fixable one-line bug for a convention nobody can enforce, and throws away the context
  that makes errors diagnosable.

---

## R9 — How the actor reaches the logs, given that the logging infrastructure is deferred

**Decision**: this feature publishes the actor and nothing else about logging. The
authentication middleware, having verified the credential, takes the logger from the
context, decorates it with `actor` — the account id from `sub` — and puts it back:

```go
log := logger.FromContext(ctx).With("actor", actor.ID)
ctx = logger.Into(ctx, log)
```

Every layer logs through `logger.FromContext(ctx)`. The two things that make those
events useful — JSON output and a correlation identifier already on the logger — come
from the observability feature this one depends on, and the plan's Dependencies section
records what stays blocked until it exists. Until then `logger.FromContext` returns the
undecorated logger and this code is correct but under-informative.

Field names this feature contributes, shared with whatever the observability feature
defines: `actor`, `operation`, `reason`.

**Rationale**: FR-026 says the actor appears "alongside the correlation identifier every
request already carries", and no correlation identifier exists in the codebase at all.
That is a third gap beyond the two the spec names, and the decision was to send it and
the JSON output to a feature of their own rather than grow this one — which is right,
since both belong to Principle IX in full, including the health check and metrics that
nobody has built either.

What stays here is the part that is genuinely this feature's: only the authentication
middleware knows who the actor is, because only it has verified the credential. Putting
that on the context logger rather than threading it through signatures is what makes
FR-026 nearly free once the infrastructure lands — a layer that logs at all logs the
actor, with nothing added to its parameters. It also gives every layer a single import,
`internal/logger`, instead of the service importing `middleware`, which would be a
layering inversion.

**Alternatives rejected**:

- **Passing the actor as an explicit parameter to everything that might log.** Honest,
  and it changes the signature of every function in the call graph.
- **Reading the raw actor from the context at each call site.** Every site repeats the
  same `With` call, and the one that forgets breaks SC-005 silently.
- **Deferring the actor to the observability feature too.** FR-026 is a requirement of
  *this* spec, and the actor is the half of it that only this feature can produce.

---

## R10 — Input validation: unknown fields refused at decode, shape validated in the DTO

**Decision**: `httphandler.Decode` gains `DisallowUnknownFields()` on its decoder, refuses
a body with trailing content after the first JSON value, and **runs the DTO's own
validation when it has one**:

```go
type Validator interface{ Validate() error }

func Decode[T any](r *http.Request) (T, error) {
    // ... json decode with DisallowUnknownFields ...
    if v, ok := any(&value).(Validator); ok {
        if err := v.Validate(); err != nil {
            return value, err
        }
    }
    return value, nil
}
```

Each request DTO implements `Validate() error` for the shape of its own fields. Validation
failures return 400 with a message naming the offending field and never echoing a
submitted value.

**Rationale**: FR-024 requires unexpected fields to be refused rather than ignored, and
`DisallowUnknownFields` is exactly that check, applied once in the one place every body
is decoded. It is also what makes US1's fourth scenario true for free: a registration
request carrying a privilege level is refused because the DTO has no such field, so there
is no path by which a client-supplied privilege could ever reach an account.

Shape validation — present, non-empty, plausible email, password length bounds — lives in
the DTO because it is a property of the border format, not of the domain: answering those
questions needs the request and nothing else, and the message they produce names a JSON
field, which is border vocabulary. Domain invariants — email normalisation, the closed set
of privilege levels, uniqueness — live in `model` and the service. That split is what
keeps the DTO free of business rules, as the constitution requires, while still failing
fast.

Having `Decode` run the validation rather than the controller is the same move as
`DisallowUnknownFields`, for the same reason: a check every endpoint must perform belongs
in the one place every endpoint goes through, not in a line each controller has to
remember. A DTO that defines `Validate` is always validated; one that does not pays
nothing. It is the shape Mat Ryer's `Validator` takes in the article this codebase already
borrowed `Encode` and `Decode` from.

The line between border and domain moves when a rule stops being about shape. A password
policy — "must contain a digit" — is a business decision about account security, and the
right response then is not to move validation into the service but to put the predicate in
`model` and have the DTO call it: the rule lives where it belongs and the refusal still
happens before persistence is touched.

The decode change is safe: no endpoint currently accepts a body.

**Alternatives rejected**:

- **A validation library (`go-playground/validator`).** A dependency and a tag-based DSL
  for six fields, in a project whose constitution asks for one library per
  responsibility.
- **Validating in the service.** The service would then own error messages that name JSON
  field names, which is border vocabulary.
- **`DisallowUnknownFields` per endpoint, or `Validate()` called by each controller.** The
  one endpoint that forgets is the one that ships the hole.

---

## R11 — Email normalisation

**Decision**: `model.NormalizeEmail` trims surrounding whitespace and lowercases the
address. It is applied at registration and at sign-in, and the `users.email` column
carries a unique index over the already-normalised value.

**Rationale**: The spec's assumption is explicit — one address must not become two
accounts, and the same normalisation has to run on both paths or the unique index
protects nothing. Keeping it a domain function rather than a database `CITEXT` column or
a functional index means it is testable as a pure function and it behaves identically in
the application and in the database, because the database only ever sees normalised
values.

**Alternatives rejected**: `CITEXT` (a Postgres extension, and it handles case but not
whitespace); a functional unique index on `lower(email)` (splits the rule between Go and
SQL, where the two can drift); normalising only at registration (sign-in then fails for
an address that differs by case, which is the exact edge case the spec lists).

---

## R12 — Superuser provisioning

**Decision**: the auth service exposes a second creation path, separate from registration:

```go
func (a *Auth) EnsureSuperuser(ctx context.Context, email, password string) error
```

It normalises the email, and if no account holds it, creates one with the superuser
privilege level. If one already exists it does nothing — including not resetting its
password. `cmd` calls it once at startup with `NSP_SUPERUSER_EMAIL` and
`NSP_SUPERUSER_PASSWORD`, and skips it when either is unset.

Registration cannot produce a superuser: its DTO has no privilege field and `Register`
hardcodes the common level.

**Rationale**: FR-019 requires exactly one superuser that account creation cannot produce,
and the spec assumes it is provisioned at bootstrap from injected configuration.

Provisioning belongs in the service and not in `cmd`, even though `cmd` is what triggers
it. Creating an account means normalising an email, hashing a password at the configured
cost and enforcing uniqueness — business rules, all three, and `cmd`'s job is to assemble
the dependency graph and start the server, not to know them. It needs its own method
rather than reusing `Register` precisely because `Register` must always produce a common
user (FR-022); a privilege parameter on it would be the hole FR-019 exists to close.

Idempotence is what lets the container restart without a second account or a failed start,
and what makes the same code path work on a fresh volume and on an existing one. Not
resetting the password on an existing account matters too: a restart would otherwise
silently undo a password the owner had changed.

**Alternatives rejected**:

- **`cmd` creating the account directly**, which an earlier draft of this plan said. It
  puts domain rules in the composition root and makes the provisioning untestable without
  starting a server.
- **An `INSERT` in `init.sql`.** It would need a bcrypt hash committed to the repository,
  and `init.sql` only runs on an empty volume.
- **A CLI subcommand.** One more thing to remember, and the demonstration fails if anyone
  forgets it.
- **A flag on registration.** FR-019 forbids it, and a flag like that is exactly the path
  an attacker looks for first.

---

## R13 — Configuration becomes fallible

**Decision**: `LoadCfg() *Cfg` becomes `LoadCfg() (*Cfg, error)`. It returns an error when
the signing secret is absent or shorter than 32 bytes, and when a lifetime or the bcrypt
cost is present but unparseable. Defaults are applied with `cmp.Or` for the values that
have sensible ones — host, port, lifetimes (15 minutes and 7 days), bcrypt cost (12).
`main` already prints the error and exits non-zero.

New variables: `NSP_JWT_SECRET` (required), `NSP_ACCESS_TTL`, `NSP_REFRESH_TTL`,
`NSP_BCRYPT_COST`, `NSP_SUPERUSER_EMAIL`, `NSP_SUPERUSER_PASSWORD`.

**Rationale**: FR-032 and SC-009 require the service to refuse to start without a usable
signing secret rather than serve credentials nobody can verify, and the current
`LoadCfg` cannot express failure at all. The 32-byte minimum is what "usable" means for
an HMAC-SHA256 key — a shorter secret is a weaker key, silently.

The existing variables keep their names (`NSPPSQLDS`, `NSPHOST`, `NSPPORT`) even though
the new ones read better with separators; renaming them is a change to deployment that
this feature has no reason to make.

**Alternatives rejected**: panicking inside `LoadCfg` (loses the clean error path `main`
already has, and makes the function untestable); accepting any non-empty secret (SC-009
says "usable", and a four-character secret is not); a config file (environment variables
are what the Docker setup already uses).

---

## R14 — Schema delivery, and a startup check that makes it safe

**Decision**: the two new tables are appended to `backend/db/init.sql`. Repository tests
mount that same file into the testcontainers Postgres. **And** `cmd` verifies, right
after the connection ping, that every table the service needs exists; when one does not,
it refuses to start and names both the missing tables and the command that fixes them.

The check is a small DAO, `dao.SchemaSql`, with one method:

```go
func (dao *SchemaSql) MissingTables(ctx context.Context, want []string) ([]string, error)
```

It queries `information_schema.tables`. `cmd` owns the list of expected tables and the
decision to abort; the DAO only runs the SQL, which keeps it out of `cmd` where it would
break the layering.

**Rationale**: the mechanics of `init.sql` are that Postgres runs
`/docker-entrypoint-initdb.d/` only when its data volume is empty. The file therefore ran
once, the day the volume was created, and editing it afterwards changes nothing until the
volume is recreated with `docker compose down -v`.

Losing the data is not the problem — `players` holds two seeded rows. The problem is that
nothing announces the mismatch. A teammate pulls, starts the server, and gets a 500
carrying `pq: relation "users" does not exist` from three layers down, with no hint that
what changed was a `.sql` file their Postgres never read. That happens once per person
per schema change, and this is a group project.

The startup check is the same pattern this feature already implements for the signing
secret: FR-032 refuses to start rather than serve credentials nobody can verify. A schema
the service cannot work against is the other infrastructure precondition, and it deserves
the same treatment — fail early and clearly instead of late and confusingly. Roughly
thirty lines, and it turns the worst symptom of the simple mechanism into an instruction.

**Alternatives rejected**:

- **goose / golang-migrate / tern.** They solve this properly: numbered files, a version
  table in the database, and `git pull` followed by a start applies what is missing
  without destroying anything. The cost is a dependency, an operational concept the
  constitution does not name, and a decision about whether migrations run at startup or
  by hand — for a database with three tables and a disposable development volume. It
  becomes worth it when there is data worth preserving; the startup check buys the time
  to get there without the confusing failure in between.
- **`init.sql` alone, with the recreation documented in quickstart.** What the plan said
  before. Documentation is not a mechanism: the person who needs the instruction is
  precisely the one who does not know to look for it.

---

## R15 — Testing strategy per layer

**Decision**:

| Layer | What is real | What is mocked |
|---|---|---|
| `model` | everything — pure functions | nothing |
| `adapters` | the real `jwt` and `bcrypt` libraries, fixed inputs, injected clock for expiry | nothing; no network |
| `service` | the service | repositories and both adapters |
| `repository` | a real Postgres from testcontainers, seeded with `init.sql` | nothing |
| `controller` | the controller, checked against `contracts/openapi.yaml` | the service |
| `middleware` | the middleware | the token verifier |
| `server` | the route table and the registration bypass check | the controllers |
| e2e | the whole stack through `server.NewServer`, Postgres from testcontainers | nothing |

Case selection follows Principle XIII: equivalence classes for input validation, boundary
values at every threshold — the expiry instant is exercised one tick before, exactly at,
and one tick after, which is the spec's first edge case — and a decision table for the
combinations, notably token kind × access level, which is the rule FR-038 states.

Expiry is testable at the boundary only if the adapter takes a clock, so `adapters.JWT`
receives a `func() time.Time` and tests inject a fixed one.

**Rationale**: this is the constitution's own table, applied. The one thing it adds is the
clock: "exactly at the expiry instant" is not a case you can write against `time.Now`.

**Alternatives rejected**: mocking the DAO in repository tests, which is what the existing
`player_test.go` does — it tests the delegation and not the SQL, which is the only thing a
repository has. testcontainers is the constitution's answer and CI already runs on a
runner with Docker.

---

## R16 — Where the OpenAPI document lives

**Decision**: designed here as [contracts/openapi.yaml](./contracts/openapi.yaml) and
published to `backend/api/openapi.yaml`, which is the maintained artifact Principle VII
requires. Controller contract tests read the published copy, so a drift between the
implementation and the document fails the build.

**Rationale**: the spec directory is a record of what was designed for this feature; the
backend needs one document that describes the current API and keeps describing it after
feature 004. Pointing the contract tests at the published copy is what keeps it honest —
an undocumented endpoint or a changed status code breaks a test rather than aging
quietly.

**Alternatives rejected**: keeping only the copy under `specs/` (feature 004 would add a
second partial document and there would be no whole); generating the document from code
annotations (a code-generation dependency, and it makes the document a report rather than
a contract).

---

## R17 — How each credential is transported

**Decision**:

| Endpoint | Access level | Credential, and where |
|---|---|---|
| `POST /auth/register` | anonymous | none |
| `POST /auth/login` | anonymous | email and password in the body |
| `POST /auth/refresh` | renewal | the refresh token in `Authorization: Bearer` |
| `POST /auth/logout` | authenticated | access token in `Authorization: Bearer`, **no body** |
| `GET /players` | authenticated | access token in `Authorization: Bearer` |

**Session families.** A sign-in opens a family, identified by a UUID that every rotation
of that session inherits: `R1 → R2 → R3` are three tokens and one session. The family id
is a column on `refresh_tokens` and a `sid` claim on both credentials.

That is what lets sign-out take no parameter. Without it, the access token says only "you
are account 42", and an account can hold several live refresh tokens from different
devices, so the server has no way to know which session the caller is ending — it has to
ask, which means the client sends a refresh token, which means the server must then verify
that the token belongs to the caller, which means a 403 exists and has to be tested. With
`sid`, the session is named by a credential the middleware already verified:

```sql
UPDATE refresh_tokens SET revoked_at = NOW()
WHERE family_id = $sid AND user_id = $actor AND revoked_at IS NULL;
```

The `user_id` predicate is redundant — the `sid` came from a verified token — and it stays
as a cheap guard against a signing-key mistake ever making it load-bearing.

The authentication middleware refuses anything that is not exactly `Bearer <token>` — no
scheme, an empty value, an unknown scheme — rather than trying to interpret it, which is
US3's fourth scenario.

**Rationale**: renewal carries its credential in the header because on that endpoint the
refresh token *is* the credential, which is what lets the middleware enforce the renewal
access level by checking `typ` — the level would be a label with nothing behind it
otherwise.

Sign-out is the endpoint the family id exists for. FR-015 fixes it as authenticated, so
its credential is the access token; but an access token cannot be revoked, so what
sign-out invalidates is a refresh token it has to identify somehow. Taking that identity
from a verified credential rather than from the request body is the same principle FR-010
states for the actor: a value the client supplies is a value that has to be validated,
tested and distrusted, and the cheapest way to handle one is not to accept it.

**Alternatives rejected**:

- **Refresh token in the body of the sign-out request.** What an earlier draft of this
  plan said. It needs a request DTO with its validation, an ownership comparison of the
  body token's `sub` against the context actor, and a 403 path — all of which exist only
  because the endpoint accepts a parameter it does not need. It also obliges the frontend
  to hold the refresh token and resend it on an endpoint where it sends nothing else.
- **Refresh token in a cookie.** The right answer for a browser client, and it brings
  `SameSite`, CSRF and domain configuration into a delivery whose scope is explicitly the
  API boundary, with the browser side deferred to a frontend feature.
- **Sign-out revoking every refresh token of the account.** No family column and no claim,
  and it breaks the spec's own edge case that two sign-ins are independent and neither
  invalidates the other: closing the session on a phone would close it on the desktop.
- **Refresh token in the body of `/auth/refresh` too.** Then the renewal access level
  could not be enforced by middleware and would have to be re-checked inside the
  controller — a declaration the enforcement does not read.

**What the family does *not* change**: reuse detection still revokes every live refresh
token of the **account**, not of the family. FR-037 says "every renewal credential of that
account", and it is right to: a thief holding one family's token may well hold another's,
and the cost of being wrong is one extra sign-in.

---

## Go 1.26 idioms this feature adopts

Checked against the Modern Go Guidelines for the version in `backend/go.mod`:

- `errors.AsType[APIError](err)` in `httphandler.Wrap`, instead of `errors.As` with a
  target variable — R8 depends on this.
- `cmp.Or` for configuration defaults, instead of an `if` per value — R13.
- `omitzero` on DTO fields whose zero value should vanish (numeric, time, struct);
  `omitempty` stays for strings, slices and maps.
- `t.Context()` in tests that need a context bound to the test's lifetime.
- Method-aware `ServeMux` patterns, which `routes.go` already uses.
- `for range n` where the index is unused.
