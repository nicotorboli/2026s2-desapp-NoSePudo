# Quickstart: Interactive API Documentation (Swagger/OpenAPI)

Validates the four user stories end-to-end against a real running backend. No
fixtures beyond what the backend already needs (see repo [README](../../README.md)
for `NSP_JWT_SECRET` / `NSPPSQLDS` setup).

## Prerequisites

```bash
cd backend
docker compose up -d          # Postgres
go run ./cmd/server
```

## US1 — Browse the complete documentation (P1)

1. Open `http://localhost:8080/docs/` in a browser.
2. Confirm it loads without any network request leaving the machine (open dev
   tools → Network, reload, check every request's host is `localhost:8080`).
3. Confirm endpoints are grouped under the `auth` and `players` tags, matching
   [`backend/api/openapi.yaml`](../../backend/api/openapi.yaml)'s `tags:` and
   `paths:` sections exactly — count them: today that's 7 operations
   (`POST /auth/register`, `POST /auth/login`, `POST /auth/refresh`,
   `POST /auth/logout`, `GET /players`, `GET /players/{id}`,
   `POST /players/sync`). See [data-model.md](./data-model.md) for why this
   count always matches the contract, never a second description.
4. Expand `POST /auth/register`: confirm its request body schema and response
   codes are shown, matching the contract's `requestBody`/`responses`.

**Expected**: all 7 operations visible, zero manually duplicated text, zero
CDN requests.

## US2 — Try it out for real (P1)

1. On `POST /auth/register`, click "Try it out", fill a body with a fresh
   email/password, Execute.
2. Confirm the page shows a real `201` (or whatever the backend actually
   returns) with the actual response body — not a client-side mock.
3. Click the padlock ("Authorize"), log in via `POST /auth/login` first to get
   a token, then paste it into the `accessToken` field and Authorize.
4. Call `GET /players` via try-it-out with no extra header set: confirm it
   succeeds — the token from step 3 was reused automatically (FR-006; see
   [research.md §3](./research.md#3-making-try-it-out-target-the-right-backend-regardless-of-hostport)
   for why Swagger UI does this natively).
5. Call `GET /players` again after clearing the Authorize field: confirm the
   real `401` the backend returns is shown, not a generic tool-invented error.

**Expected**: every call in this section hit the real backend; status codes
match what `curl`/Postman would get for the same input (SC-005).

## US3 — Documentation tracks the live contract (P2)

1. With the backend still running, open `backend/api/openapi.yaml` and change
   one operation's `summary`.
2. Reload `http://localhost:8080/docs/` — no restart, no rebuild.
3. Confirm the new summary is visible immediately (FR-003/SC-003).
4. Revert the edit.
5. From `backend/`, run `go test ./internal/server/...`: confirm
   `TestPublishedContractMatchesTheRouteTable` and
   `TestNoRouteIsRegisteredOutsideBuildMux` still pass unmodified — this
   feature added no entries to either test (FR-008; see
   [research.md §4](./research.md#4-where-the-new-routes-live-relative-to-the-existing-route-table)).

## US4 — Easy to find (P3)

1. From a fresh clone (or just re-reading [README.md](../../README.md)),
   confirm the "La API" section names `/docs/` as the path.
2. Request `http://localhost:8080/docs/` with the backend running: confirm it
   resolves (`200`).

## Edge cases

- **Missing contract file**: temporarily rename
  `backend/api/openapi.yaml`, request `http://localhost:8080/openapi.yaml`
  directly: confirm a clear `500` with a short message, and that the backend
  process is still up and serving `/players` etc. normally (FR-012). Restore
  the file afterward.
- **Protected endpoint, no credential**: call `POST /players/sync` via
  try-it-out with the Authorize field empty: confirm the real `401`/`403` is
  shown.
- **Two methods, same path**: confirm `GET /players/{id}` and any other method
  on the same path (if one exists) both appear as distinct entries.
