# Quickstart: Player Token Market (Mercado de tokens)

**Feature**: `005-mercado-de-tokens`

## Prerequisites

1. PostgreSQL running:
   ```bash
   cd backend
   docker compose down -v && docker compose up -d
   ```
2. Environment:
   ```bash
   export NSPPSQLDS="postgres://devuser:devpassword@localhost:5433/nsp_db?sslmode=disable"
   export NSP_JWT_SECRET="super-secret-key-that-is-at-least-32-bytes-long"
   export NSP_SUPERUSER_EMAIL="admin@nosepudo.ar"
   export NSP_SUPERUSER_PASSWORD="admin-secure-password"
   export NSP_INITIAL_USER_BALANCE="1000"
   ```

The superuser credentials are what make the market work: that account holds the 100 tokens of every player and rests them at the active quote. Without them the book is empty and every order is rejected for lack of liquidity.

## Run

```bash
cd backend
go run ./cmd/server
```

At startup, and after every `POST /players/sync`, the market bootstrap runs: each player gets an initial quote of 1 credit, the superuser gets 100 tokens, and one sell order for the unsold tokens rests at the quote. It is idempotent, so restarting the server does not issue more tokens or duplicate the offer.

## Walkthrough

### 1. Register and sign in

```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "trader@nosepudo.ar", "password": "password123"}'

curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "trader@nosepudo.ar", "password": "password123"}'
```

Keep the `access_token` and the user `id` from the responses. The account starts with `NSP_INITIAL_USER_BALANCE` credits available and nothing reserved.

### 2. Portfolio before trading

```bash
curl http://localhost:8080/users/<USER_ID>/portfolio \
  -H "Authorization: Bearer <TOKEN>"
```

```json
{
  "userId": 2,
  "availableCash": 1000.0,
  "reservedCash": 0.0,
  "tokensValuation": 0.0,
  "totalValue": 1000.0,
  "unrealizedResult": 0.0,
  "positions": []
}
```

### 3. Buy at the active quote

No price, so the order is priced at the player's active quote and defaults to `FOK`: it executes in full against the market maker or it is rejected.

```bash
curl -X POST http://localhost:8080/orders/buy \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"playerId": 1, "quantity": 10}'
```

```json
{
  "id": 2,
  "side": "BUY",
  "playerId": 1,
  "price": 1.0,
  "quantity": 10,
  "executedQuantity": 10,
  "remainingQuantity": 0,
  "timeInForce": "FOK",
  "status": "FILLED",
  "createdAt": "2026-10-17T12:00:00Z",
  "trades": [
    {"quantity": 10, "price": 1.0, "totalAmount": 10.0, "executedAt": "2026-10-17T12:00:00Z"}
  ]
}
```

The portfolio now reports 10 tokens, `availableCash` 990, `tokensValuation` 10 and `totalValue` 1000.

### 4. A limit order that rests

With a price below what anybody is offering, the order cannot cross and defaults to `GTC`, so it waits. The credits it needs are moved from available to reserved.

```bash
curl -X POST http://localhost:8080/orders/buy \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"playerId": 1, "quantity": 5, "price": 0.5}'
```

Status `PENDING`, `executedQuantity` 0. The portfolio now shows `availableCash` 987.5 and `reservedCash` 2.5: the same total, with part of it committed.

### 5. See it and cancel it

```bash
curl http://localhost:8080/orders -H "Authorization: Bearer <TOKEN>"

curl -X DELETE http://localhost:8080/orders/<ORDER_ID> \
  -H "Authorization: Bearer <TOKEN>"
```

```json
{"id": 3, "status": "CANCELLED", "releasedCredits": 2.5}
```

`reservedCash` returns to 0 and `availableCash` back to 990. Cancelling an order that is already `FILLED` or `CANCELLED` returns 409; cancelling somebody else's returns 403.

### 6. Not waiting: IOC

With only part of the quantity available at that price, `IOC` takes what it can and gives the rest back instead of leaving it resting.

```bash
curl -X POST http://localhost:8080/orders/buy \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"playerId": 1, "quantity": 500, "price": 1.0, "timeInForce": "IOC"}'
```

The order comes back `CANCELLED` with `executedQuantity` equal to whatever the market maker still had, and no reservation held.

### 7. Sell

```bash
curl -X POST http://localhost:8080/orders/sell \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"playerId": 1, "quantity": 4}'
```

Four tokens go back to the market maker at the active quote and 4 credits return to the balance. The average purchase price of the remaining 6 tokens is unchanged by the sale: selling releases cost basis in proportion to what left.

### 8. History

```bash
curl "http://localhost:8080/users/<USER_ID>/transactions?page=1&limit=20" \
  -H "Authorization: Bearer <TOKEN>"
```

One entry per execution, newest first, with the side from that user's perspective. Requesting another user's history without being the superuser returns 403.

## What to check by hand

| Claim | How to see it |
|---|---|
| Nothing stays reserved by a dead order | After cancelling everything, `reservedCash` is 0 and `SELECT SUM(reserved_balance) FROM user_accounts` is 0 |
| Credits are conserved | `SELECT settlement_group FROM ledger_entries GROUP BY settlement_group HAVING SUM(amount) <> 0` returns no rows |
| Exactly 100 tokens per player | `SELECT player_id, SUM(available_quantity + reserved_quantity) FROM user_positions GROUP BY player_id` is 100 for every player |
| The bootstrap is idempotent | Restart the server and re-run the two queries above: nothing changes |
| Every change is audited | `SELECT entity_type, action, actor, diff_old, diff_new FROM player_audit_logs ORDER BY created_at DESC LIMIT 10` |

## Automated verification

```bash
cd backend
go test ./...
golangci-lint run
```
