# Plan: tests de la feature de estadísticas

Tests de integración (prioridad) y de comportamiento para el scraper de
WhoScored, el matcheo, la persistencia, la lectura y el controller unificado.
No se modificó código de producción ni tests ajenos a la feature.

## Tests nuevos

### Adapter (fixtures, API externa → stub permitido)
- `backend/internal/adapters/whoscored/client_test.go`:
  - `TestFetchLeagueStatistics_MergesOffensiveAndDefensive`: con un
    `httptest.Server` que distingue `subcategory`, verifica que se piden
    `offensive`+`defensive` y que el merge por `playerId` produce las filas
    esperadas (goles/asistencias enteros; el resto por partido y rating).
  - `TestFetchLeagueStatistics_UnsupportedLeague`: código de liga inválido → error.
  - `TestFetchLeagueStatistics_BotChallenge`: 403 → `ErrBotChallenge`.
  - `TestFetchLeagueStatistics_RateLimit`: 429 → `ErrRateLimitExceeded`.

### Repository (testcontainers, Postgres real)
- `backend/internal/persistence/repository/player_statistics_test.go`:
  - `SaveAndList`: `SaveStatistics` inserta vía COPY y devuelve
    `(processed, inserted)`; dos pasadas → snapshot anexado, el más reciente
    primero en `GetPlayerStatistics`.
  - `ListsOnlyRequestedPlayer`: solo devuelve las filas del jugador pedido.
  - Helper local `startStatisticsPostgres` que **falla en voz alta** si Docker
    no está (AGENTS), en lugar de saltar en silencio como el helper viejo.

### Service (testcontainers + adaptador externo falso)
- `backend/internal/service/player_statistics_sync_test.go`:
  - `MatchesAndPersists`: seed de clubes divergentes (`Arsenal FC`,
    `Manchester City FC`, `AFC Bournemouth`) vs `Arsenal`/`Man City`/
    `Bournemouth` → verifica `matched=3/unmatched=1/inserted=3` y las filas en
    `player_statistics`.
  - `UniqueNameFallback`: `Brighton & Hove Albion FC` matchea con `Brighton`.
  - `DuplicateNameIsNotMatched`: homónimos quedan `unmatched` (sin falso positivo).
  - `RateLimit` / `BotChallenge`: adaptador con esos errores → centinelas
    `ErrStatisticsRateLimitExceeded` / `ErrStatisticsProviderBlocked`.
  - `PartialSuccessWhenALeagueFails`: `partial_success` conservando lo ya
    procesado.
- `backend/internal/service/player_statistics_test.go`:
  - `GetPlayerStatistics`: nuevas primero, `ErrNotFound` si no existe el
    jugador, lista vacía si no tiene filas.

### Controller + server (tests existentes del controller unificado)
- `backend/internal/controller/sync_test.go`: aridad de `NewSyncController`
  (3 servicios) + fakes de `PlayerStatisticsSyncService`/`PlayerStatisticsService`;
  casos nuevos `SyncStatistics` (200/429/502) y `GetPlayerStatistics` (200/404/400).
- `backend/internal/server/server_test.go`: `NewSyncController` con 3 servicios.
- `backend/internal/server/routes_test.go`: `stubController` con
  `SyncStatistics`/`GetPlayerStatistics`; whitelist con
  `GET /players/{id}/statistics` (anonymous) y `POST /players/statistics/sync`
  (superuser).

## Verificación
- `gofmt -l`, `go build ./...`, `go vet ./...`, `golangci-lint run ./...`: **pasan** (0 issues).
- `go test ./...`:

| Paquete | Resultado |
|---|---|
| adapters/whoscored, controller, configuration, model, dao, middleware, httphandler, e2e, footballdata, adapters | **ok** |
| repository, service | **fallan solo por Docker** (abajo) |
| server | 2 fallos preexistentes ajenos a la feature (abajo) |

### Fallos documentados (dejados así por consigna: >3 intentos o imposibles de arreglar sin tocar lo ajeno)

1. **`internal/persistence/repository` y `internal/service`** (11 tests nuevos de
   testcontainers): `failed to create Docker provider` — el daemon de Docker no
   está disponible en el entorno (sin `/var/run/docker.sock`, sin podman, sin
   sudo; `systemctl start docker` da "Connection timed out"). Los tests están
   bien escritos y **fallan en voz alta** como exige AGENTS; correrán cuando
   Docker esté disponible (`docker compose up -d` + daemon activo). No es un
   fallo de lógica.

2. **`internal/server`**: `TestServer_ServeHTTP_GetPlayers_SinCredencial` y
   `TestServer_ServeHTTP_GetPlayers_CredencialInvalida` esperan 401 en
   `GET /players`, pero el cambio concurrente de niveles de acceso dejó esa ruta
   en `AccessAnonymous` → responde 200. Son tests del catálogo (no de la feature
   de estadísticas); se dejan como están por instrucción de no tocar tests
   ajenos a la feature.

## Fuera de alcance
- Código de producción (solo se arreglaron los tests del controller unificado
  que la propia feature rompió).
- `configuration` (resuelto aparte).
- Tests que "parsean el proyecto" (prohibido por instrucción).
