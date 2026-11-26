# Implementation: P2P price units

Branch: `fix/p2p-price-scale` · Date: 2026-11-16 · Source: `plans/p2p-token-session/review.md` (P0)

## Summary

The P2P wire format now carries money as a decimal string in credits, parsed at the controller
boundary into `model.Credits` (scale 10000) and formatted back on the way out. No change to the
service, repository, or DAO layers: they already worked in `model.Credits`.

## Backend

- `backend/internal/controller/dto/p2p.go`
  - `CreateP2PSessionRequest.PricePerToken` is a `string`; new `ParsedPrice()` uses
    `CreditsDesdeDecimal`; `Validate` rejects unparsable and non-positive prices.
  - New sentinel `ErrP2PPriceInvalid` (Spanish message, shown as the 400 body).
  - `P2PSessionResponse` and `P2PTradeResponse` expose `pricePerToken` / `totalAmount` as
    strings produced by `CreditsADecimal`.
- `backend/internal/controller/p2p.go`
  - `Create` parses the price and passes `model.Credits` to the service; parse failures map to
    400 before the service is called.
- `backend/api/openapi.yaml`
  - `CreateP2PSessionRequest.pricePerToken`, `P2PSessionResponse.{pricePerToken,totalAmount}`,
    and `P2PTradeResponse.{pricePerToken,totalAmount}` are strings with a 1–4 decimal pattern,
    example, and unit description.

## Frontend

- `src/api/p2p.ts`: `pricePerToken` and `totalAmount` are `string` in all three DTOs.
- `src/pages/P2PCreatePage/P2PCreatePage.tsx`: price input is a decimal string (default `1`),
  accepts comma or dot, validates 1–4 decimals, previews the total in scaled integer units,
  and submits the normalized string.
- `src/pages/P2PSessionPage/P2PSessionPage.tsx`: price, total, accept button, and success card
  render the API strings.
- `src/utils/format.ts`: new `formatCreditDecimal` trims trailing zeros and uses a comma;
  existing `formatCredits` semantics were not changed.

## Tests added

- `backend/internal/controller/dto/p2p_test.go`: parse table (integer, 1.5, 0.0001, trailing
  zeros), invalid inputs (empty, text, zero, negative, five decimals), response and trade
  formatting, and a JSON check that the amount is a string.
- `backend/internal/controller/p2p_test.go`: `"1.5"` reaches the service as `15000` and the
  response emits `"1.5000"` / `"7.5000"`; `"abc"` is a 400 and never reaches the service.
- `frontend/src/pages/P2PCreatePage/P2PCreatePage.test.tsx`: typing `1,5` submits
  `pricePerToken: "1.5"` and shows `1,5 créditos`; `abc` is rejected without an API call.

## Verification

| Check | Result |
| --- | --- |
| `go build ./...` | pass |
| `go test ./internal/controller/... ./internal/model/... ./internal/dto/... ./internal/middleware/... ./internal/httphandler/...` | pass |
| `go test ./internal/server/ -run Contract` | pass (3 tests) |
| `./scripts/pre-commit.sh` | pass, 0 issues |
| `npm run typecheck` | pass |
| `npm run lint` | pass |
| `npm run knip` | pass |
| `npm test` | pass, 16 files / 58 tests |

## Known unrelated failure

`go test ./internal/server/` fails in `TestDeclaredLevelsMatchTheSpecification`
(`internal/server/routes_test.go:151`): its hardcoded `expected` map lacks the four P2P routes
added by the merged feature. This reproduces on `dev` without these changes and is not caused
by the price fix. The test message treats adding the routes as an explicit exposure decision.

## Not done

- Data remediation: no database was reachable (Docker daemon not running, port 5433 closed),
  so existing `p2p_sessions` / P2P trade rows could not be counted. The boundary fix is correct
  for new data; if rows exist in any deployed environment, remediation is a separate decision.
- Repository/service integration tests for P2P: there are none, and testcontainers could not
  run in this environment.
- The six P1 findings from the review report.
