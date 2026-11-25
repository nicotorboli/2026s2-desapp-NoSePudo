# Error handling

Where sentinel errors live, how they travel, and which HTTP status each controller produces.
Line numbers are from the current tree.

Inventory (77 `errors.New` lines outside tests):

```bash
grep -rn 'errors.New' backend/internal --include='*.go' | grep -v '_test.go'
```

Per package: `model` 24, `service` 16, `controller/dto` 10, `adapters` 7, `controller` 6,
`middleware` 5, `persistence/repository` 4, `persistence/dao` 3, `dto` 1, `configuration` 1.
The sum is 77.

## 1. Sentinel ownership per layer

| Layer | File | Sentinels |
| --- | --- | --- |
| `model` | `model/account.go:6-10` | `ErrInvalidAmount`, `ErrInsufficientFunds`, `ErrReservationTooSmall`, `ErrAccountNotFound`, `ErrAccountAlreadyExists` |
| `model` | `model/position.go:6-7` | `ErrInsufficientTokens`, `ErrPositionNotFound` |
| `model` | `model/order.go:9-11` | `ErrOrderNotLive`, `ErrInvalidQuantity`, `ErrOrderNotFound` |
| `model` | `model/p2p_session.go:9-13` | `ErrP2PSessionNotFound`, `ErrP2PSessionNotLive`, `ErrP2PSessionExpired`, `ErrP2PSessionAlreadyTaken`, `ErrP2PSessionSelfTrade` |
| `model` | `model/quote.go:9-10` | `ErrQuoteNotFound`, `ErrQuoteAlreadyExists` |
| `model` | `model/refresh_token.go:18,22,27,39` | `ErrRefreshTokenReused`, `ErrRefreshTokenExpired`, `ErrRefreshTokenNotFound`, `ErrRefreshTokenRevoked` |
| `model` | `model/user.go:14,19,25` | `ErrEmailTaken`, `ErrInvalidCredentials`, `ErrUserNotFound` |
| `service` | `service/player.go:17-18` | `ErrNotFound`, `ErrRateLimitExceeded` |
| `service` | `service/market.go:14-19` | `ErrInvalidQuantity`, `ErrInvalidPrice`, `ErrPlayerInactive`, `ErrNoLiquidity`, `ErrUnknownSide`, `ErrNotOwner` |
| `service` | `service/p2p.go:11-18` | `ErrP2PSelfTrade`, `ErrP2PExpired`, `ErrP2PAlreadyTaken`, `ErrP2PNotLive`, `ErrP2PNotFound` (aliases of model values), `ErrP2PForbidden` (new) |
| `service` | `service/score.go:16` | `ErrScoreNotFound` |
| `service` | `service/portfolio.go:12` | `ErrForbidden` |
| `service` | `service/sync_run.go:16-18` | `ErrSyncRunNotFound`, `ErrSyncScheduleNotFound`, `ErrInvalidSyncType` |
| `service` | `service/player_statistics.go:16-17` | `ErrStatisticsRateLimitExceeded`, `ErrStatisticsProviderBlocked` |
| `persistence/repository` | `persistence/repository/player.go:14` | `ErrNotFound` |
| `persistence/repository` | `persistence/repository/market.go:13` | `ErrPlayerInactive` |
| `persistence/repository` | `persistence/repository/p2p.go:14-16` | `ErrP2PDatabaseNil`; plus an unnamed inline error at `persistence/repository/p2p.go:335` |
| `persistence/dao` | `persistence/dao/player_sql.go:14` | `ErrNotFound` |
| `persistence/dao` | `persistence/dao/container.go:13,15` | `errNilDatabase`, `errLockNeedsTransaction` (package-private) |
| `controller/dto` | `controller/dto/market.go:16-21` | `ErrQuantityNotPositive`, `ErrPriceNotPositive`, `ErrPriceNotANumber`, `ErrPriceTooPrecise`, `ErrUnknownTimeInForce`, `ErrPlayerNotIdentified` |
| `controller/dto` | `controller/dto/p2p.go:12-15` | `ErrP2PPlayerRequired`, `ErrP2PQuantityRequired`, `ErrP2PPriceRequired`, `ErrP2PPriceInvalid` |
| `dto` | `dto/sync_run.go:61` | inline error for a missing `enabled` field (no named sentinel) |
| `controller` | `controller/auth.go:121`, `controller/market.go:119,210` | `errMissingActor`, `errInvalidPlayerFilter`, `errInvalidIdentifier` (package-private) |
| `controller` | `controller/p2p.go:72,94,115` | inline errors for a missing token (no named sentinel) |
| `middleware` | `middleware/authentication.go:86-88`, `middleware/authorization.go:15-16` | `errMissingCredential`, `errMalformedHeader`, `errWrongTokenKind`, `errNoActor`, `errInsufficientPrivilege` (package-private) |
| `adapters` | `adapters/jwt.go:51-52`, `adapters/password.go:11`, `adapters/footballdata/client.go:18-19`, `adapters/whoscored/client.go:20-21` | token, password, and provider errors |
| `configuration` | `configuration/configuration.go:30` | `ErrMissingJWTSecret` |

Names that exist in more than one package:

- `ErrNotFound`: `service/player.go:17`, `persistence/repository/player.go:14`,
  `persistence/dao/player_sql.go:14`. Three distinct values with the same name.
- `ErrPlayerInactive`: `service/market.go:16` and `persistence/repository/market.go:13`.
- `ErrInvalidQuantity`: `service/market.go:14` and `model/order.go:10`.
- `ErrRateLimitExceeded`: `service/player.go:18`, `adapters/footballdata/client.go:19`,
  `adapters/whoscored/client.go:21`.
- `service/p2p.go:12-16` does not create new values for the five P2P sentinels; it aliases the
  model ones. `ErrP2PForbidden` at `service/p2p.go:17` is genuinely new.

Named validation sentinels live in `controller/dto`. `internal/dto` and a few controller sites use
inline errors instead; `model` and `service` decide business outcomes.

## 2. Translation rule: repository/dao to service to controller

1. **DAO** returns its own sentinel or a wrapped driver error.
   `sql.ErrNoRows` becomes `model.ErrP2PSessionNotFound` for a session
   (`persistence/dao/p2p_session_sql.go:183-185`) and `dao.ErrNotFound` for a player
   (`persistence/dao/player_sql.go:14`). Other driver failures are wrapped with `%w`
   (`persistence/dao/p2p_session_sql.go:186`).
2. **Repository** either passes the DAO sentinel through or re-declares it as a repository
   sentinel. `PlayerRepositoryImpl.GetPlayerByID` maps `dao.ErrNotFound` to
   `repository.ErrNotFound` (`persistence/repository/player.go:71-79`);
   `MarketRepository.WithPlayerLocked` maps the same DAO error to `repository.ErrNotFound` and a
   false active flag to `repository.ErrPlayerInactive`
   (`persistence/repository/market.go:76-85`).
3. **Service** has three behaviours:
   - maps repository sentinels to service sentinels:
     `service/player.go:99-101` (`repository.ErrNotFound` → `service.ErrNotFound`),
     `service/market.go:543-552` (repository `ErrNotFound`/`ErrPlayerInactive` → service values),
     `service/score.go:209-211` and `service/score.go:242-243`;
   - passes model sentinels through unchanged: `service/p2p.go:106-112` returns whatever the
     repository produced, and `service/market.go:504-506` returns `model.ErrOrderNotLive`
     directly;
   - re-declares aliases so callers can match either name: `service/p2p.go:11-16`.
4. **Controller** converts a sentinel into `httphandler.NewError(status, message, cause)` through
   a `*StatusFor` function (`controller/market.go:240-262`, `controller/p2p.go:127-151`) or an
   inline `errors.Is` branch (`controller/auth.go:38-42`, `controller/player.go:57-60`).
   Unmatched errors are returned unchanged.
5. **httphandler** keeps the cause for `errors.Is`/`errors.As` and exposes the status
   (`httphandler/error.go:21-27`, `httphandler/error.go:44-49`). The wrapper finds any `APIError`
   in the `%w` chain and writes its status; anything else becomes
   `500 {"error":"Internal Server Error"}` (`httphandler/wrapper.go:39-50`).

Rule of thumb: a sentinel is owned by the layer that can decide it. The DAO knows a row is
missing; the repository knows whether that means "player not found" for this workflow; the
service knows whether it is an orchestration error; the controller knows the HTTP status.

## 3. Error to status mapping per controller

Default for every controller: an error that matches no branch is returned unchanged and the
wrapper answers `500` (`httphandler/wrapper.go:47-50`).

### `marketStatusFor` (`controller/market.go:240-262`)

| Match | Status | Line |
| --- | --- | --- |
| `service.ErrNotFound`, `service.ErrPlayerInactive` | `404` | `controller/market.go:242-243` |
| `service.ErrForbidden`, `service.ErrNotOwner` | `403` | `controller/market.go:244-245` |
| `service.ErrNoLiquidity` | `409` | `controller/market.go:246-247` |
| `model.ErrOrderNotLive` | `409` | `controller/market.go:248-249` |
| `model.ErrInsufficientFunds` | `422` | `controller/market.go:250-251` |
| `model.ErrInsufficientTokens`, `model.ErrPositionNotFound` | `422` | `controller/market.go:252-253` |
| `service.ErrInvalidQuantity`, `service.ErrInvalidPrice`, `service.ErrUnknownSide` | `400` | `controller/market.go:254-256` |
| `model.ErrAccountNotFound` | `404` | `controller/market.go:257-258` |
| anything else | `500` | `controller/market.go:259-260` |

### `p2pStatusFor` (`controller/p2p.go:127-151`)

| Match | Status | Line |
| --- | --- | --- |
| `service.ErrNotFound`, `service.ErrPlayerInactive`, `model.ErrP2PSessionNotFound` | `404` | `controller/p2p.go:129-131` |
| `service.ErrP2PForbidden` | `403` | `controller/p2p.go:132-133` |
| `model.ErrP2PSessionSelfTrade`, `service.ErrP2PSelfTrade` | `409` | `controller/p2p.go:134-135` |
| `model.ErrP2PSessionAlreadyTaken`, `service.ErrP2PAlreadyTaken` | `409` | `controller/p2p.go:136-137` |
| `model.ErrP2PSessionNotLive`, `service.ErrP2PNotLive` | `409` | `controller/p2p.go:138-139` |
| `model.ErrP2PSessionExpired`, `service.ErrP2PExpired` | `410` | `controller/p2p.go:140-141` |
| `model.ErrInsufficientFunds` | `422` | `controller/p2p.go:142-143` |
| `model.ErrInsufficientTokens`, `model.ErrPositionNotFound` | `422` | `controller/p2p.go:144-145` |
| `service.ErrInvalidQuantity`, `service.ErrInvalidPrice` | `400` | `controller/p2p.go:146-147` |
| anything else | `500` | `controller/p2p.go:148-149` |

### Other controllers

- **auth**: `model.ErrEmailTaken` → `409` (`controller/auth.go:38-40`);
  `model.ErrInvalidCredentials` → `401` (`controller/auth.go:57-59`); refresh reuse, expiry, or
  revocation → `401` (`controller/auth.go:76-80`); a missing actor → `401`
  (`controller/auth.go:108-119`); everything else → `500`.
- **player**: a malformed path id → `400` (`controller/player.go:45-53`); `service.ErrNotFound`
  on `GET /players/{id}` → `404` (`controller/player.go:57-60`); list errors are returned
  unchanged → `500`.
- **sync**: `service.ErrSyncRunNotFound` → `404` (`controller/sync.go:75-77`);
  `service.ErrSyncScheduleNotFound` → `404` (`controller/sync.go:147-149`);
  `service.ErrNotFound` for statistics → `404` (`controller/sync.go:171-173`); malformed type,
  id, offset, or limit → `400` (`controller/sync.go:181-226`); everything else → `500`.
- **scoring**: `service.ErrNotFound` and `service.ErrScoreNotFound` → `404`
  (`controller/scoring.go:164-166`); `service.ErrNotFound` on score history, quotes, and price →
  `404` (`controller/scoring.go:183-185`, `controller/scoring.go:240-242`,
  `controller/scoring.go:267-269`); no active strategy → `404`
  (`controller/scoring.go:65-67`); an unknown strategy id or a non-positive coefficient is a
  plain error from the service (`service/score.go:140-142`, `service/score.go:172-174`) and
  therefore `500`.

Request-shape validation is separate: `httphandler.Decode` returns `400` for unknown fields,
multiple JSON values, type mismatches, and DTO `Validate` failures
(`httphandler/utils.go:40-67`), and `413` for a body over 1 MB
(`httphandler/utils.go:46-49`). Middleware answers `401` for a missing, malformed, invalid, or
wrong-kind credential (`middleware/authentication.go:115-128`) and `403` when the privilege is
insufficient (`middleware/authorization.go:41-52`).

## 4. Known gaps (facts, not fixed here)

- **Unknown player on P2P create maps to `500`, not `404`.** The service returns the repository
  sentinel directly (`service/p2p.go:65-68`), which is `repository.ErrNotFound`
  (`persistence/repository/player.go:71-79`, value at `persistence/repository/player.go:75`).
  `p2pStatusFor` matches
  `service.ErrNotFound` (`controller/p2p.go:129`), a different value declared at
  `service/player.go:17`. The error falls to `controller/p2p.go:148-149` and the wrapper answers
  `500` (`httphandler/wrapper.go:47-50`).
- **Inactive player on P2P create maps to `500` for the same reason.**
  `service/p2p.go:69-71` returns `repository.ErrPlayerInactive`
  (`persistence/repository/market.go:13`), while `p2pStatusFor` matches
  `service.ErrPlayerInactive` (`service/market.go:16`).
- **A non-UUID token maps to `500`.** The controller only rejects an empty token
  (`controller/p2p.go:70-73`, `controller/p2p.go:92-95`, `controller/p2p.go:113-116`). The DAO
  compares that string against a `UUID` column (`persistence/dao/p2p_session_sql.go:64-79`);
  Postgres rejects the comparison, and the DAO wraps the driver error as a generic read error
  (`persistence/dao/p2p_session_sql.go:186`) instead of `model.ErrP2PSessionNotFound` (mapped at
  `persistence/dao/p2p_session_sql.go:183-185` only for `sql.ErrNoRows`). The wrapper then
  answers `500` (`httphandler/wrapper.go:47-50`).
