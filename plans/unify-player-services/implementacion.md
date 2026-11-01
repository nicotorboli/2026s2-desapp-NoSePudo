# Plan: unificación de servicios de jugadores y estadísticas

Refactor pedido explícitamente: colapsar cuatro servicios en dos siguiendo el
criterio de dominio (jugador vs. estadísticas), no el de operación (sync vs.
lectura). Cada servicio de dominio reúne sus lecturas y su sincronización:

- `PlayerServiceImpl` = `ListPlayers` + `GetPlayerByID` + `SyncPlayers`
- `PlayerStatisticsServiceImpl` = `GetPlayerStatistics` + `SyncStatistics`

## Modificaciones

### Service
- `backend/internal/service/player.go`: absorbido `player_sync.go`. El struct
  queda con `adapter FootballDataClient`, `repo PlayerRepository` y
  `logger *slog.Logger`; `NewPlayerService` pasa a recibir `(adapter, repo,
  logger)`; se mueven `defaultLeagues`, `ErrRateLimitExceeded` y el método
  `SyncPlayers`.
- `backend/internal/service/player_statistics.go`: absorbido
  `player_statistics_sync.go`. El struct queda con `adapter WhoscoredClient`,
  `players PlayerRepository`, `statistics PlayerStatisticsRepository` y
  `logger *slog.Logger`; `NewPlayerStatisticsService` pasa a recibir
  `(adapter, players, statistics, logger)`; se mueven
  `ErrStatisticsRateLimitExceeded`, `ErrStatisticsProviderBlocked`,
  `maxPlayersPerLeague`, `SyncStatistics`, `syncLeague`, `buildMatchIndexes` y
  `mapAdapterError`.
- `backend/internal/service/player_sync.go`: eliminado.
- `backend/internal/service/player_statistics_sync.go`: eliminado.
- `backend/internal/service/container.go`: `Container` pasa a `Player`
  (`*PlayerServiceImpl`), `PlayerStats` (`*PlayerStatisticsServiceImpl`) y
  `Auth`; se quitan `Sync` y `StatisticsSync`.

### Controller
- `backend/internal/controller/sync.go`: `PlayerStatisticsService` incorpora
  `SyncStatistics`; se elimina `PlayerStatisticsSyncService`;
  `RestSyncController` pasa de tres dependencias a dos (`playerSvc`, `statsSvc`)
  y se actualizan los receptores de los tres endpoints.
- `backend/internal/controller/container.go`: `NewSyncController` se cablea con
  `services.Player` y `services.PlayerStats`.

### Tests
- `backend/internal/service/player_test.go`: `NewPlayerService(nil, mockRepo,
  nil)` en los casos de lectura; `NewPlayerService(mockAdapter, mockRepo,
  logger)` en los de sincronización.
- `backend/internal/service/player_statistics_test.go`: el helper llama
  `NewPlayerStatisticsService(nil, repos.Player, repos.PlayerStatistics, nil)`.
- `backend/internal/service/player_statistics_sync_test.go`: el helper devuelve
  `*service.PlayerStatisticsServiceImpl` y llama `NewPlayerStatisticsService(
  adapter, repos.Player, repos.PlayerStatistics, logger)`.
- `backend/internal/controller/sync_test.go`: `mockStatisticsSyncService` se
  fusiona dentro de `mockPlayerStatisticsService` (métodos `GetPlayerStatistics`
  y `SyncStatistics`); `newSyncController` pasa a dos argumentos.
- `backend/internal/server/server_test.go`: `mockStatisticsSyncService` se
  fusiona en `mockPlayerStatisticsService`; `NewSyncController` con dos
  argumentos.
- `backend/internal/server/routes_test.go`: `stubStatisticsSyncService` se
  fusiona en `stubPlayerStatisticsService`; `NewSyncController` con dos
  argumentos.

## Sin cambios
- `backend/internal/service/dependencies.go`, `normalize.go`,
  `backend/internal/server/routes.go`, `backend/api/openapi.yaml`,
  `cmd/server/main.go` y `test/e2e/stack_test.go` (solo usan los constructores de
  containers, cuya firma no cambia).
- Rutas y niveles de acceso: sin cambios.

## Verificación

| Check | Resultado |
|---|---|
| `gofmt` sobre los paquetes tocados | sin salida |
| `go build ./...` | pasa |
| `go vet ./...` | pasa |
| Tests no-Docker (`service` player, `controller`, `server` rutas/contrato) | pasan |
| Tests con Docker (testcontainers de estadísticas y e2e) | a cargo del usuario / no ejecutados en el sandbox |

### Fallos conocidos (ajenos a este refactor)
- `internal/server`: `TestServer_ServeHTTP_GetPlayers_SinCredencial` y
  `TestServer_ServeHTTP_GetPlayers_CredencialInvalida` esperan 401 en
  `GET /players`, pero la ruta está declarada `AccessAnonymous` → responde 200.
  Son fallos preexistentes, no de este refactor.
