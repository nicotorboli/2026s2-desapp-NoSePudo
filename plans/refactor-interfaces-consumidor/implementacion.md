# Plan: interfaces definidas por el consumidor en todas las capas

Refactor de arquitectura pedido explícitamente: cada capa tipa sus dependencias
como interfaces declaradas del lado del consumidor (en el paquete que las usa) y
los proveedores exponen structs concretos, no interfaces innecesarias. Se revierte
así la decisión anterior de exponer interfaces en los containers (ver
`whoscored-statistics-sync/refactor-controller-unificado.md`).

## Modificaciones

### Adapters
- `backend/internal/adapters/footballdata/client.go`: eliminada la interfaz `Client`
  (queda el struct concreto `FootballDataClient`).
- `backend/internal/adapters/whoscored/client.go`: eliminada la interfaz `Client`
  (queda `WhoscoredClient`).
- `backend/internal/adapters/container.go`: `FootballData` y `Whoscored` pasan a
  punteros concretos `*footballdata.FootballDataClient` / `*whoscored.WhoscoredClient`.

### DAO
- `backend/internal/persistence/dao/player_sql.go`: eliminada `PlayerDAO`.
- `backend/internal/persistence/dao/player_statistics_sql.go`: eliminada
  `PlayerStatisticsDAO`.
- `backend/internal/persistence/dao/audit_sql.go`: eliminada `AuditDAO`.
- `backend/internal/persistence/dao/container.go`: `Player`, `PlayerStatistics` y
  `Audit` pasan a `*PlayerSql`, `*PlayerStatisticsSql`, `*AuditSql`.

### Repository
- `backend/internal/persistence/repository/player.go`: eliminada `PlayerRepository`;
  declaradas localmente `PlayerDAO` (ListPlayers, GetPlayerByID,
  GetPlayerByExternalID, UpsertPlayer, DeactivateMissingPlayers) y `AuditDAO`
  (InsertAuditLog). `NewPlayerRepository` recibe las interfaces locales.
- `backend/internal/persistence/repository/player_statistics.go`: eliminada
  `PlayerStatisticsRepository`; declarada localmente `PlayerStatisticsDAO`
  (InsertStatistics, ListByPlayerID); quitado el import `dao`.
- `backend/internal/persistence/repository/container.go`: `Player` y
  `PlayerStatistics` pasan a `*PlayerRepositoryImpl` / `*PlayerStatisticsRepositoryImpl`.

### Service
- `backend/internal/service/dependencies.go` (nuevo): interfaces compartidas del
  paquete `service` — `PlayerRepository`, `PlayerStatisticsRepository`,
  `FootballDataClient`, `WhoscoredClient`.
- `backend/internal/service/player.go`: eliminada `PlayerService`; campo `repo PlayerRepository`.
- `backend/internal/service/player_sync.go`: eliminada `PlayerSyncService`; campos
  `adapter FootballDataClient` y `repo PlayerRepository`; quitado el import `repository`.
- `backend/internal/service/player_statistics.go`: eliminada `PlayerStatisticsService`;
  campos `players PlayerRepository` y `statistics PlayerStatisticsRepository`.
- `backend/internal/service/player_statistics_sync.go`: eliminada
  `PlayerStatisticsSyncService`; campos `adapter WhoscoredClient`,
  `players PlayerRepository`, `statistics PlayerStatisticsRepository`; quitado el
  import `repository`.
- `backend/internal/service/container.go`: campos `Player`, `Sync`,
  `StatisticsSync` y `PlayerStats` pasan a los punteros concretos `*PlayerServiceImpl`,
  `*PlayerSyncServiceImpl`, `*PlayerStatisticsSyncServiceImpl`,
  `*PlayerStatisticsServiceImpl`.

### Controller
- `backend/internal/controller/container.go`: eliminadas `PlayerController`,
  `SyncController` y `AuthController`; campos a `*RestPlayerController`,
  `*RestSyncController`, `*RestAuthController`; quitado el import `httphandler`.
- `backend/internal/controller/player.go`: declarada localmente `PlayerService`
  (ListPlayers, GetPlayerByID).
- `backend/internal/controller/sync.go`: declaradas localmente `PlayerSyncService`,
  `PlayerStatisticsSyncService`, `PlayerStatisticsService` (agregado import `dto`).

### Tests (mínimos, para mantener `go test` verde)
- `backend/internal/service/player_statistics_test.go`: helper devuelve
  `*service.PlayerStatisticsServiceImpl`.
- `backend/internal/service/player_statistics_sync_test.go`: helper recibe
  `*fakeWhoscoredClient` y devuelve `*service.PlayerStatisticsSyncServiceImpl`.
- `backend/internal/server/server_test.go`: `mockAuthController` reemplazado por
  `mockAuthService` (Register/Login/Refresh/Logout a nivel servicio) cableado con
  `controller.NewAuthController(...)`; el test de rutas anónimas manda cuerpos JSON
  válidos (el controller real ahora decodifica el cuerpo). Quitado el import
  `httphandler`, agregado `strings`.
- `backend/internal/server/routes_test.go`: `stubController` reemplazado por stubs
  de servicio y controllers reales vía `NewPlayerController`/`NewSyncController`/
  `NewAuthController`.

## Verificación

| Check | Resultado |
|---|---|
| `gofmt -l .` | pasa (sin salida) |
| `go build ./...` | pasa |
| `go vet ./...` | pasa |
| `golangci-lint run ./...` | **pasa** (0 issues, versión v2.14.0) |
| `go test` no-Docker (model, httphandler, middleware, adapters, configuration, controller, controller/dto) | **pasan** |
| `go test` mocks de service y repository | **pasan** |

### Fallos conocidos (ajenos a este refactor)
- `internal/server`: `TestServer_ServeHTTP_GetPlayers_SinCredencial` y
  `TestServer_ServeHTTP_GetPlayers_CredencialInvalida` esperan 401 en
  `GET /players`, pero la ruta está declarada `AccessAnonymous` → responde 200.
  Son los mismos 2 fallos que ya existían antes de este refactor (verificado con
  `git stash`): no se tocan por estar fuera de alcance.
- Docker no disponible en el entorno: los tests de testcontainers
  (`internal/persistence/repository`, `internal/service` de integración y
  `test/e2e`) no se ejecutaron; el código compila y `go vet` los valida.
