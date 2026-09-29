# Quickstart: JWT Authentication and Authorization

**Feature**: `003-autenticacion-jwt` | **Plan**: [plan.md](./plan.md) | **Contract**: [contracts/openapi.yaml](./contracts/openapi.yaml)

How to run the feature and prove each user story against a live service. This is a
validation guide, not an implementation guide — the code belongs in `tasks.md`.

---

## Prerequisites

- Go 1.26.5 (`backend/go.mod` is the source of truth)
- Docker, for Postgres and for the testcontainers-based tests
- `golangci-lint` at the version in `.golangci-version`
- `curl` and `jq` for the flows below

---

## 1. Database

The schema ships in `backend/db/init.sql`. Postgres runs the files in
`/docker-entrypoint-initdb.d/` **only when its data volume is empty**, so that file ran
once — the day the volume was created — and editing it afterwards changes nothing until
the volume is recreated. An existing local volume predates the `users` and
`refresh_tokens` tables, so:

```bash
cd backend
docker compose down -v      # -v is the point: it drops the volume, so init.sql runs again
docker compose up -d
docker compose logs -f db   # wait for "database system is ready to accept connections"
```

You do not have to remember this. The server checks at startup that every table it needs
exists and refuses to start otherwise, naming the missing tables and this command (R14).
To see it, start the server against a volume that predates the feature:

```bash
go run ./cmd/server; echo "exit: $?"
```

**Expected**: a message naming `users` and `refresh_tokens` as missing, the
`docker compose down -v && docker compose up -d` command, and a non-zero exit. Not a
`relation "users" does not exist` surfacing from three layers down at the first request.

---

## 2. Environment

The service now refuses to start without a usable signing secret (FR-032, SC-009).

```bash
export NSPPSQLDS="postgres://devuser:devpassword@localhost:5433/nsp_db?sslmode=disable"
export NSPHOST=127.0.0.1
export NSPPORT=8080

export NSP_JWT_SECRET="$(openssl rand -base64 48)"   # at least 32 bytes
export NSP_ACCESS_TTL=15m
export NSP_REFRESH_TTL=168h
export NSP_BCRYPT_COST=12

export NSP_SUPERUSER_EMAIL=admin@nosepudo.ar
export NSP_SUPERUSER_PASSWORD='cambiar-esto-en-serio'
```

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `NSPPSQLDS` | yes | — | Postgres data source |
| `NSPHOST` | no | `127.0.0.1` | Listen address |
| `NSPPORT` | no | `8080` | Listen port |
| `NSP_JWT_SECRET` | **yes** | — | HS256 signing key, 32 bytes minimum |
| `NSP_ACCESS_TTL` | no | `15m` | Access token lifetime |
| `NSP_REFRESH_TTL` | no | `168h` | Refresh token lifetime (7 days) |
| `NSP_BCRYPT_COST` | no | `12` | bcrypt cost factor |
| `NSP_SUPERUSER_EMAIL` | no | — | Superuser provisioned at startup, idempotently |
| `NSP_SUPERUSER_PASSWORD` | no | — | Its password |

### Validating SC-009 first

Before anything else, confirm the service refuses to start without a secret:

```bash
cd backend
env -u NSP_JWT_SECRET go run ./cmd/server; echo "exit: $?"
```

**Expected**: a message naming the missing signing secret on stderr, and a non-zero
exit. No port is opened.

---

## 3. Run

```bash
cd backend
go run ./cmd/server
```

**Expected**, on stdout:

```text
2026-09-28T14:00:00Z INF Database connection successful
2026-09-28T14:00:00Z INF Starting server at 127.0.0.1:8080
```

Console output, not JSON — the switch belongs to the observability feature this one
depends on. It is why the `jq` checks under US6 are marked blocked.

---

## 4. Validation flows

Each flow maps to a user story. Run them in order; later ones reuse earlier output.

### US1 — Create an account

```bash
curl -s -o /dev/null -w '%{http_code}\n' -X POST localhost:8080/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"Participante@NoSePudo.ar  ","password":"unaClaveLarga1"}'
```

**Expected**: `201`.

```bash
# same address, different case and whitespace — must collide
curl -s -X POST localhost:8080/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"participante@nosepudo.ar","password":"otraClaveLarga1"}'

# a privilege level the client tried to grant itself
curl -s -X POST localhost:8080/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"nuevo@nosepudo.ar","password":"unaClaveLarga1","privilege":"superuser"}'

# password below the lower bound
curl -s -X POST localhost:8080/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"corto@nosepudo.ar","password":"corta"}'
```

**Expected**: `409 email already registered`; `400 unknown field "privilege"`; `400`
naming the password bound. The second is the one that matters most — the field does not
exist on the DTO, so there is no path by which a client-supplied privilege could reach
an account.

Confirm nothing recoverable was stored:

```bash
docker compose exec db psql -U devuser -d nsp_db \
  -c "SELECT email, left(password_hash, 7) AS algo, privilege FROM users;"
```

**Expected**: the email stored lowercased and trimmed, `algo` showing `$2a$12$`, and
`privilege = 1` (common user).

### US2 — Sign in

```bash
SESSION=$(curl -s -X POST localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"participante@nosepudo.ar","password":"unaClaveLarga1"}')

ACCESS=$(echo "$SESSION"  | jq -r .access_token)
REFRESH=$(echo "$SESSION" | jq -r .refresh_token)
echo "$SESSION" | jq '{token_type, access_expires_at, refresh_expires_at}'
```

**Expected**: both tokens present, `token_type: Bearer`, and two RFC 3339 expiry
instants.

Read the access token's payload — anyone holding it can, which is why FR-007 exists:

```bash
echo "$ACCESS" | cut -d. -f2 | base64 -d 2>/dev/null | jq .
```

**Expected**: `sub`, `iat`, `exp`, `jti`, `typ: "access"`, `priv: "user"`, `sid`. No email, no
password, nothing damaging.

Both failure paths must be indistinguishable:

```bash
curl -s -X POST localhost:8080/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"participante@nosepudo.ar","password":"claveIncorrecta"}'
curl -s -X POST localhost:8080/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"nadie@nosepudo.ar","password":"claveIncorrecta"}'
```

**Expected**: two identical `401 {"error":"invalid credentials"}` responses, taking
comparable time — the second path still runs a bcrypt comparison so the timing does not
reveal that the account is missing.

### US3 — The catalog requires a credential

```bash
curl -s -o /dev/null -w 'none:      %{http_code}\n'  localhost:8080/players
curl -s -o /dev/null -w 'no scheme: %{http_code}\n'  localhost:8080/players -H "Authorization: $ACCESS"
curl -s -o /dev/null -w 'unknown:   %{http_code}\n'  localhost:8080/players -H "Authorization: Basic $ACCESS"
curl -s -o /dev/null -w 'altered:   %{http_code}\n'  localhost:8080/players -H "Authorization: Bearer ${ACCESS}x"
curl -s -o /dev/null -w 'refresh:   %{http_code}\n'  localhost:8080/players -H "Authorization: Bearer $REFRESH"
curl -s -o /dev/null -w 'valid:     %{http_code}\n'  localhost:8080/players -H "Authorization: Bearer $ACCESS"
```

**Expected**: `401` for the first five, `200` for the last. The `refresh` line is
FR-038: a renewal credential never grants access to a resource.

For the expired case, restart with `NSP_ACCESS_TTL=1s`, sign in, wait two seconds and
repeat the last call: `401`.

### US4 — Every endpoint declares its access level

This one is a test, not a request:

```bash
cd backend
go test ./internal/server/ -run 'TestRoutes' -v
```

**Expected**: the table test passes and reports the five routes with their levels;
`anonymous` for register and login, `authenticated` for `/players` and logout,
`renewal` for refresh.

Now demonstrate SC-002 by breaking it deliberately, in the two ways it can break. First
add an entry to the route table in `routes.go` with no `access` field. **Expected**: the
table test fails naming the entry, and the server panics at startup. Then take that entry
out of the table and register it directly with `mux.Handle(...)`. **Expected**: the bypass
test fails, naming the file and line. Revert both.

### US5 — Privilege travels in the credential

```bash
SU=$(curl -s -X POST localhost:8080/auth/login -H 'Content-Type: application/json' \
  -d "{\"email\":\"$NSP_SUPERUSER_EMAIL\",\"password\":\"$NSP_SUPERUSER_PASSWORD\"}" \
  | jq -r .access_token)

echo "$SU" | cut -d. -f2 | base64 -d 2>/dev/null | jq -r .priv
```

**Expected**: `superuser`. The common user's token says `user`. And:

```bash
docker compose exec db psql -U devuser -d nsp_db \
  -c "SELECT count(*) FROM users WHERE privilege = 2;"
```

**Expected**: exactly `1`, and it was not created through `/auth/register`. Restart the
service and re-run: still `1`, because provisioning is idempotent.

### US7 — Staying signed in

```bash
NEW=$(curl -s -X POST localhost:8080/auth/refresh -H "Authorization: Bearer $REFRESH")
NEW_ACCESS=$(echo "$NEW"  | jq -r .access_token)
NEW_REFRESH=$(echo "$NEW" | jq -r .refresh_token)
```

**Expected**: `200`, a new pair, and `NEW_REFRESH != REFRESH` — the presented token was
replaced (FR-036).

Now replay the old one, which is the theft scenario:

```bash
curl -s -X POST localhost:8080/auth/refresh -H "Authorization: Bearer $REFRESH"
curl -s -X POST localhost:8080/auth/refresh -H "Authorization: Bearer $NEW_REFRESH"
```

**Expected**: the first is `401` — it was already used. The second is **also** `401`,
even though it had never been used: detecting the replay revoked every live refresh
token of the account (FR-037), because there is no way to tell the legitimate holder
from the thief.

```bash
docker compose exec db psql -U devuser -d nsp_db \
  -c "SELECT count(*) FROM refresh_tokens WHERE used_at IS NULL AND revoked_at IS NULL;"
```

**Expected**: `0` for that account — SC-012.

Sign out, after a fresh sign-in:

```bash
SESSION=$(curl -s -X POST localhost:8080/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"participante@nosepudo.ar","password":"unaClaveLarga1"}')
ACCESS=$(echo "$SESSION" | jq -r .access_token)
REFRESH=$(echo "$SESSION" | jq -r .refresh_token)

curl -s -o /dev/null -w 'logout:  %{http_code}\n' -X POST localhost:8080/auth/logout \
  -H "Authorization: Bearer $ACCESS"          # sin body: el sid del access token alcanza

curl -s -o /dev/null -w 'refresh: %{http_code}\n' -X POST localhost:8080/auth/refresh \
  -H "Authorization: Bearer $REFRESH"

curl -s -o /dev/null -w 'players: %{http_code}\n' localhost:8080/players \
  -H "Authorization: Bearer $ACCESS"
```

**Expected**: `204`, then `401` for the renewal, then `200` for the catalog — the access
token is deliberately not invalidated and expires on its own (FR-039).

And the other session must survive it, which is the edge case the family exists for:

```bash
# two independent sign-ins, as if from two devices
A=$(curl -s -X POST localhost:8080/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"participante@nosepudo.ar","password":"unaClaveLarga1"}')
B=$(curl -s -X POST localhost:8080/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"participante@nosepudo.ar","password":"unaClaveLarga1"}')

# their sid claims differ: two families
for S in "$A" "$B"; do echo "$S" | jq -r .access_token | cut -d. -f2 | base64 -d 2>/dev/null | jq -r .sid; done

# sign out of A only
curl -s -o /dev/null -X POST localhost:8080/auth/logout \
  -H "Authorization: Bearer $(echo "$A" | jq -r .access_token)"

curl -s -o /dev/null -w 'A renueva: %{http_code}\n' -X POST localhost:8080/auth/refresh \
  -H "Authorization: Bearer $(echo "$A" | jq -r .refresh_token)"
curl -s -o /dev/null -w 'B renueva: %{http_code}\n' -X POST localhost:8080/auth/refresh \
  -H "Authorization: Bearer $(echo "$B" | jq -r .refresh_token)"
```

**Expected**: two different `sid` values, then `401` for A and `200` for B. Signing out of
one session leaves the other working.

### US6 — The logs

> **Blocked on the observability feature.** The checks below assume JSON output and a
> correlation identifier on every event, and this feature delivers neither — see the
> plan's Dependencies section. What *is* deliverable here is the `grep` for leaked
> secrets, which works against console output too. The rest is written now so that it
> runs unchanged the day the logger emits JSON.

Capture a complete run and check it mechanically, which is what FR-030 exists for:

```bash
go run ./cmd/server > /tmp/run.jsonl 2>&1 &
# ... run the flows above ...
kill %1
```

```bash
# every line parses
jq -e . /tmp/run.jsonl > /dev/null && echo "all lines parse"

# SC-004: no secret and no personal datum anywhere
grep -icE 'unaClaveLarga1|cambiar-esto-en-serio|participante@nosepudo\.ar|eyJhbGciOiJIUzI1NiI' /tmp/run.jsonl

# SC-005: authenticated operations name their actor
jq -r 'select(.operation=="listPlayers") | .actor' /tmp/run.jsonl

# every request carries a correlation id
jq -r 'select(.correlation_id == null) | .' /tmp/run.jsonl

# refusals record a reason, with no subject when none could be identified
jq -r 'select(.status==401) | {operation, reason, actor, correlation_id}' /tmp/run.jsonl
```

**Expected**: all lines parse; the grep count is `0`; every `listPlayers` line has a
numeric actor; the correlation-id filter prints nothing; each refusal has a `reason`,
and the one for an unknown account has `correlation_id` and `reason` but no `actor`.

Trace one request end to end:

```bash
curl -s -o /dev/null -D- localhost:8080/players -H "Authorization: Bearer $ACCESS" \
  -H 'X-Correlation-ID: quickstart-001'
jq -r 'select(.correlation_id=="quickstart-001")' /tmp/run.jsonl
```

**Expected**: the header comes back on the response, and every event of that request is
in the output.

### SC-010 — An inner refusal stays a refusal

```bash
curl -s -o /dev/null -w '%{http_code}\n' -X POST localhost:8080/auth/refresh \
  -H "Authorization: Bearer $(echo "$REFRESH" | sed 's/.$/X/')"
```

**Expected**: `401`, not `500`. This is the wrapper fix: the refusal is raised below the
controller and wrapped with `%w` on the way up, and `errors.AsType` finds it anyway.

---

## 5. The full verification, as CI runs it

```bash
./scripts/pre-commit.sh        # gofmt, go vet, golangci-lint — from the repository root
cd backend
go test -race ./...            # needs Docker: repository and e2e tests use testcontainers
go build ./...
```

The first repository test run pulls `postgres:15-alpine` and is slow; later runs reuse
the image.

---

## Coverage of the success criteria

| Criterion | Where it is validated above |
|---|---|
| SC-001 | US3 — six calls, five refused, state unchanged |
| SC-002 | US4 — the deliberate undeclared endpoint, both tests |
| SC-003 | US1 — the three refusals, then the `psql` count |
| SC-004 | US6 — the `grep -c` returning `0`. Runs today; the `jq` half needs the observability feature |
| SC-005 | US6 — the two `jq` filters. **Blocked** until log events are JSON |
| SC-006 | The US1 → US2 → US3 sequence, run start to finish |
| SC-007 | Measured with `curl -w '%{time_total}'` over repeated sign-ins |
| SC-008 | US3 gives `401`; `403` appears in the logout ownership check |
| SC-009 | Section 2, before anything is started |
| SC-010 | The tampered refresh token returning `401` |
| SC-011 | US7 — renewal after the access token expired |
| SC-012 | US7 — the replay, and the `0` live tokens that follows |
| SC-013 | US7 — sign-out, then the refused renewal |
