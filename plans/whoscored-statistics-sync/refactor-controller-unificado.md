# Plan: refactor — controller unificado y segregación por interfaces (opción B)

Refactor de arquitectura pedido explícitamente:

1. Un único controller de sincronización/estadísticas en lugar de dos
   controllers especiales.
2. Los containers exponen interfaces, no implementaciones concretas.
3. Los tests no se modifican (opción B): las verificaciones que fallan por el
   desfase con los tests se documentan acá.

## Modificaciones (solo archivos no-test)

### Backend
- `backend/internal/controller/sync.go`: `RestSyncController` unificado con
  tres endpoints y tres dependencias de servicio:
  - `SyncPlayers()` → `service.PlayerSyncService` (football-data)
  - `SyncStatistics()` → `service.PlayerStatisticsSyncService` (WhoScored,
    con mapeo 429/502)
  - `GetPlayerStatistics()` → `service.PlayerStatisticsService` (lectura,
    con parseo de `{id}` y 400/404)
  - Constructor único: `NewSyncController(svc, statsSyncSvc, statsReadSvc)`.
- `backend/internal/controller/statistics.go`: **eliminado** (sus dos structs
  `RestStatisticsSyncController` y `RestPlayerStatisticsController` se
  integraron en `sync.go`).
- `backend/internal/controller/container.go`:
  - `SyncController` ahora exige `SyncPlayers`, `SyncStatistics` y
    `GetPlayerStatistics`.
  - `Container` quedó con solo interfaces: `Player`, `Sync`, `Auth`. Se
    eliminaron los campos concretos `StatisticsSync *RestStatisticsSyncController`
    y `PlayerStats *RestPlayerStatisticsController`.
- `backend/internal/server/routes.go`: ambas rutas de estadísticas apuntan al
  controller unificado (`s.controllers.Sync.SyncStatistics()` y
  `s.controllers.Sync.GetPlayerStatistics()`).
- `backend/internal/adapters/container.go`: los campos pasan a interfaces
  `footballdata.Client` y `whoscored.Client`; `Password`/`JWT` quedan concretos
  (patrón preexistente: los consumen `middleware` y `Auth` directamente).
- `backend/internal/service/container.go`: sin cambios (ya exponía interfaces).
- `backend/api/openapi.yaml`: sin cambios (ya documentaba ambas rutas).

## Verificación

| Check | Resultado |
|---|---|
| `gofmt -l` | pasa |
| `go build ./...` | pasa |
| `go vet ./...` | **falla** (tests no compilan, abajo) |
| `golangci-lint run ./...` | **falla** (typecheck de tests) |
| `go test -race -count=1 ./...` | **falla** solo en `internal/controller` e `internal/server` (`[build failed]`); el resto de los paquetes pasa, incluido `test/e2e` (que arma el wiring real con `controller.NewContainer`, confirmando que la producción está bien conectada) |

### Errores de verificación (determinísticos, por opción B sin tocar tests)
Cada comando falló 3 veces con el mismo resultado y se detuvo según lo indicado.

```
internal/server/routes_test.go:42:12: cannot use stubController{} as
controller.SyncController value in struct literal: stubController does not
implement controller.SyncController (missing method GetPlayerStatistics)
internal/controller/sync_test.go:44/74/95: not enough arguments in call to
controller.NewSyncController (have *mockPlayerSyncService; want
service.PlayerSyncService, service.PlayerStatisticsSyncService,
service.PlayerStatisticsService)
internal/server/server_test.go: not enough arguments in call to
controller.NewSyncController
```
Causa: los tests existentes codifican la forma vieja del controller (métodos e
instanciación). Al no compilar el paquete `internal/server`, el test de
contrato `TestPublishedContractMatchesTheRouteTable` tampoco puede ejecutarse
(hasta que se permita actualizar esos tests).

Para dejar verde cuando se habilite editar tests:
- `stubController` (routes_test.go): agregar `SyncStatistics()` y
  `GetPlayerStatistics()`.
- `sync_test.go` y `server_test.go`: pasar los tres servicios a
  `NewSyncController` (mocks nuevos para `PlayerStatisticsSyncService` y
  `PlayerStatisticsService`).
- `TestDeclaredLevelsMatchTheSpecification`: agregar al mapa whitelist
  `POST /players/statistics/sync` (superuser) y
  `GET /players/{id}/statistics` (authenticated).

## Fuera de alcance
- Tests: ninguno creado ni modificado (opción B, decisión del usuario).
- Frontend: sin cambios.
