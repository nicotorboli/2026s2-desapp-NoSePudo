# 2026-11-25 — P2P price units

- Status: proposed
- Area: p2p
- Request: P2P offers store and return the raw request integer as credits, so an offer entered as 3 credits moves 0.0003 credits.

## Invariants

- Money is an `int64` scaled by 10,000; division truncates and guards zero — `backend/internal/model/credits.go:3-5`, `:23-28`.
- Credits cross the wire as decimal strings, four decimals maximum — `backend/internal/dto/credits.go:9-25`, `backend/internal/controller/dto/market.go:42-76`.
- A P2P total equals price times quantity; transitions and escrow are unchanged — `backend/internal/model/p2p_session.go:49-94`, `backend/internal/persistence/repository/p2p.go:54-121`.

## Data and constraints

None. `p2p_sessions.price` and `market_trades.price` / `total_amount` are `BIGINT` in scaled units (`backend/db/init.sql:315`, `backend/db/init.sql:258-260`). No column, check, index, or FK changes.

## Transactions and lock order

None. The change stops at the controller boundary. `P2PRepository.Create` keeps the player lock, seller position lock, reservation, insert, audit, and commit order (`backend/internal/persistence/repository/p2p.go:54-121`).

## HTTP contract delta

| Method | Path | Access level | Request DTO | Response DTO | Units |
| --- | --- | --- | --- | --- | --- |
| POST | `/p2p/sessions` | authenticated | `CreateP2PSessionRequest` | `P2PSessionResponse` | decimal string credits |
| GET | `/p2p/sessions/{token}` | authenticated | — | `P2PSessionResponse` | decimal string credits |
| POST | `/p2p/sessions/{token}/accept` | authenticated | — | `P2PTradeResponse` | decimal string credits |

All three DTOs stay in `internal/controller/dto`. In `backend/api/openapi.yaml`, operations `createP2PSession`, `getP2PSession`, and `acceptP2PSession` declare `pricePerToken` and `totalAmount` as strings with pattern `^[0-9]+(\.[0-9]{1,4})?$` (`:1859`, `:1901`, `:1906`, `:1947`, `:1952`); `quantity` stays an integer.

## Authorization matrix

| Actor | Allowed? | Enforced at |
| --- | --- | --- |
| anonymous | no | route `backend/internal/server/routes.go:161-181` |
| authenticated seller creating | yes | route; the actor becomes the seller (`backend/internal/controller/p2p.go:51-58`) |
| authenticated seller cancelling / accepting their offer | cancel yes; accept no | `backend/internal/service/p2p.go:114-125`; `backend/internal/model/p2p_session.go:63-65` |
| authenticated third party viewing by token | yes | route only; `backend/internal/service/p2p.go:92-104` has no ownership check |
| superuser | same as authenticated | no P2P superuser branch |

## Error mapping

| Condition | Sentinel | HTTP status | Mapping site |
| --- | --- | --- | --- |
| unparsable or over-precise price | `controller/dto.ErrP2PPriceInvalid` | 400 | `backend/internal/controller/dto/p2p.go:37-46`, `backend/internal/controller/p2p.go:46-49` |
| zero or negative price | `controller/dto.ErrP2PPriceRequired` | 400 | same |
| service re-check | `service.ErrInvalidPrice` | 400 | `backend/internal/controller/p2p.go:146-147` |

The default for an unmatched error is 500 (`backend/internal/httphandler/wrapper.go:47-50`). No new layer owns a sentinel; `ErrP2PPriceInvalid` already lives in `controller/dto`.

## Concurrency and idempotency

None. Parsing is pure and runs before the transaction. Acceptance remains guarded by the status checks and the session row lock (`backend/internal/persistence/repository/p2p.go:138-161`).

## Migration and rollback

No schema migration. Pre-fix P2P rows stored the request integer as scaled units, so intent cannot be inferred. Before deploying with data, run `SELECT count(*) FROM p2p_sessions;` and `SELECT count(*) FROM market_trades WHERE p2p_session_id IS NOT NULL;`. Empty: no action. Non-empty: stop and decide remediation; never backfill automatically.

## Observability

None. No new log lines; prices and session tokens must not be logged.

## Rejected options

- Keep integers as minor units on the wire: contradicts the UI label and the market contract; users would type 10000 for one credit.
- Parse the decimal in the service: leaks transport into the service and duplicates the market parser.
- Scale only in the frontend: the API and ledger stay wrong for every other client.

## File checklist

- [ ] `backend/internal/controller/dto/p2p.go` — string request field, `ParsedPrice`, string response fields.
- [ ] `backend/internal/controller/p2p.go` — parse before the service and map failures to 400.
- [ ] `backend/api/openapi.yaml` — three schemas, patterns, and unit descriptions.
- [ ] `frontend/src/api/p2p.ts` — `pricePerToken` and `totalAmount` as strings.
- [ ] `frontend/src/pages/P2PCreatePage/P2PCreatePage.tsx` — decimal input, validation, preview.
- [ ] `frontend/src/pages/P2PSessionPage/P2PSessionPage.tsx` — render the server strings.
- [ ] `frontend/src/utils/format.ts` — `formatCreditDecimal`.
- [ ] Tests, explicitly commanded for this task only: `backend/internal/controller/dto/p2p_test.go`, `backend/internal/controller/p2p_test.go`, `frontend/src/pages/P2PCreatePage/P2PCreatePage.test.tsx`.

## Out of scope

- Remediating pre-fix rows; that is a separate data decision.
- The six P1 findings in `plans/p2p-token-session/review.md`.
- Consolidating the two DTO packages.
