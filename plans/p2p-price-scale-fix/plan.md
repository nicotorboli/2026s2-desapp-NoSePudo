# Plan: P2P price units

Source: `plans/p2p-token-session/review.md` (P0, verified independently).

## Problem

`model.Credits` is an int64 scaled by `CreditScale` (10000). Market orders parse a decimal
string into that scale, but the P2P controller casts the raw request integer
(`controller/p2p.go:51`) and the response echoes the scaled integer. The UI labels that number
as "créditos", so a listing created at 3 credits moves 0.0003 credits.

## Decision

Use the decimal string at the wire boundary, matching the market contract and reusing
`CreditsDesdeDecimal` / `CreditsADecimal`.

| Layer | Change |
| --- | --- |
| `backend/api/openapi.yaml` | `pricePerToken` and `totalAmount` become strings with up to four decimals |
| `backend/internal/controller/dto/p2p.go` | parse input, format output, new invalid-price error |
| `backend/internal/controller/p2p.go` | pass the parsed `model.Credits` to the service |
| `frontend/src/api/p2p.ts` | amounts are strings |
| `P2PCreatePage.tsx` | decimal string input, preview total from integer units |
| `P2PSessionPage.tsx` | render the server strings |

Rejected: keeping integers as minor units. It contradicts the UI labels and the market
contract, and forces users to type 10000 for one credit.

## Data check

Blocked: Docker daemon is not running and port 5433 is closed, so existing rows could not be
counted. The boundary fix is correct regardless; if any `p2p_sessions` or P2P trade rows exist
in an environment that matters, stop and decide remediation before deploying.

## Tests

Explicitly requested by the user, overriding the no-test-changes rule for this task:

- `dto/p2p_test.go`: parsing and formatting round-trip, invalid inputs.
- `controller/p2p_test.go`: request `"1.5"` reaches the service as `15000`; response emits
  `"1.5000"`; invalid input is a 400.
- `P2PCreatePage.test.tsx`: the page submits the typed decimal string.

## Acceptance

- 1.5 credits × 5 tokens: page shows 1.5 and 7.5; ledger moves 75000 scaled units.
- `go build ./...`, `go test ./internal/controller/...`, contract tests.
- `npm run typecheck`, `npm run lint`, `npm test`.

## Out of scope

The six P1 findings in the review report.
