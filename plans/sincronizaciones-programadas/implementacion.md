# Implementación — Fase 1: sincronizaciones programadas

## Alcance entregado

Motor de sincronizaciones programadas con un único concepto de dominio,
`SyncRun`, disparado por cron o manualmente. Configuración de frecuencia por
tipo en DB, con seed inicial desde variables de entorno. Progreso efímero en
memoria por liga. Sin frontend.

## Modificaciones

### Esquema y arranque
- `backend/db/init.sql`: nuevas tablas `sync_schedules` y `sync_runs`, índices
  `idx_sync_runs_type_started` e `idx_sync_runs_one_running` (único parcial por
  tipo mientras el run está `running`).
- `backend/cmd/server/main.go`: `sync_schedules` y `sync_runs` en
  `expectedTables`; `runCtx` creado antes del contenedor; uso de
  `service.NewContainerWithSync`; arranque del scheduler y apagado ordenado
  (`Stop` + `SyncRuns.Shutdown`).
- `backend/internal/configuration/configuration.go`: nuevas variables
  `NSP_SYNC_RUN_TIMEOUT`, `NSP_SYNC_PLAYERS_INTERVAL`,
  `NSP_SYNC_STATISTICS_INTERVAL`, `NSP_SYNC_PLAYERS_ENABLED` y
  `NSP_SYNC_STATISTICS_ENABLED`; helpers `syncIntervalFromEnv` y `boolFromEnv`.

### Modelo
- `backend/internal/model/sync_schedule.go` (nuevo): `SyncType`,
  `ParseSyncType`, `SyncSchedule` y límites de intervalo.
- `backend/internal/model/sync_run.go` (nuevo): `SyncTrigger`, `SyncStatus`,
  `SyncRun` y `SyncRunResult`.

### Persistencia
- `backend/internal/persistence/dao/sync_schedule_sql.go` (nuevo): `List`,
  `Get`, `Ensure`, `Set`, `ReserveDue`.
- `backend/internal/persistence/dao/sync_run_sql.go` (nuevo): `InsertRunning`
  (`ON CONFLICT` sobre el índice parcial), `InsertSkipped`, `Finish`, `Get`,
  `List` y `MarkInterrupted`.
- `backend/internal/persistence/dao/container.go`: DAOs nuevos.
- `backend/internal/persistence/repository/sync_schedule.go` (nuevo):
  interfaz consumidora + implementación.
- `backend/internal/persistence/repository/sync_run.go` (nuevo): interfaz
  consumidora + `Start` transaccional (running o skipped).
- `backend/internal/persistence/repository/container.go`: repositorios nuevos.

### Servicios
- `backend/internal/service/progress.go` (nuevo): `Progress`, `Reporter`,
  `WithProgress`, `FromContext` y `reportProgress`.
- `backend/internal/service/sync_run.go` (nuevo): `SyncRunService` con
  `Trigger`, `Recover`, `GetRun`, `ListRuns`, `ListSchedules`,
  `UpdateSchedule`, `Shutdown`; ejecución en goroutine con timeout; mapeo de
  estados; progreso efímero en un mapa protegido por mutex; cierre del run con
  contexto independiente del timeout.
- `backend/internal/service/sync_scheduler.go` (nuevo): `SyncSettings`,
  `DefaultSyncSettings`, `SyncScheduler` con ticker de 1 minuto, catch-up al
  arrancar, `ReserveDue` y `Ensure`.
- `backend/internal/service/container.go`: `Container` con `SyncRuns` y
  `SyncScheduler`; `NewContainerWithSync`; `NewContainer` conserva su firma
  para los tests y e2e.
- `backend/internal/service/player.go`: reporte de progreso por liga en
  `SyncPlayers`.
- `backend/internal/service/player_statistics.go`: reporte de progreso por liga
  en `SyncStatistics`.

### Borde HTTP
- `backend/internal/dto/sync_run.go` (nuevo): `SyncRunProgress`,
  `SyncRunResponse`, `SyncRunPageResponse`, `SyncScheduleResponse`,
  `UpdateSyncScheduleRequest` y conversiones desde modelo.
- `backend/internal/controller/sync.go`: `SyncService` local; handlers
  `CreateRun`, `GetRun`, `ListRuns`, `ListSchedules`, `UpdateSchedule`;
  `GetPlayerStatistics` conservado; eliminados `SyncPlayers` y
  `SyncStatistics`.
- `backend/internal/controller/container.go`: wiring con `PlayerStats` y
  `SyncRuns`.
- `backend/internal/server/routes.go`: eliminadas `POST /players/sync` y
  `POST /players/statistics/sync`; agregadas las cinco rutas `/sync/*` con
  `AccessSuperuser`.
- `backend/api/openapi.yaml`: paths viejos y schemas `SyncResult`/
  `SyncStatisticsResult` eliminados; paths y schemas nuevos de runs y
  schedules; tag `sync`; referencia a `/sync` en la descripción de `Forbidden`.

### Tests
- `backend/internal/server/routes_test.go`: stub de `controller.SyncService`,
  constructor actualizado y whitelist con las rutas nuevas.
- `backend/internal/server/server_test.go`: mock de `SyncService` y
  constructor actualizado.
- `backend/internal/controller/sync_test.go`: casos viejos de sync reemplazados
  por `CreateRun`, `GetRun`, `ListRuns`, `ListSchedules` y `UpdateSchedule`;
  conservados los de `GetPlayerStatistics`.

No se crearon tests nuevos del scheduler, del run service ni de los DAOs.

## Verificación

| Check | Resultado |
|---|---|
| `gofmt -l .` | sin salida |
| `go build ./...` | pasa |
| `go vet ./...` | pasa |
| `./scripts/pre-commit.sh` (golangci-lint v2.14.0) | `0 issues`, pasa |
| `go test -race` de `model`, `dto`, `controller`, `server`, `configuration`, `middleware`, `httphandler` | pasa |
| `TestPublishedContractMatchesTheRouteTable` | pasa |
| `TestDeclaredLevelsMatchTheSpecification` | pasa |
| `TestEveryRouteDeclaresAnAccessLevel` | pasa |
| Tests de `repository` y `service` con Docker | fallan de forma visible: Docker no disponible (`permission denied` en `/var/run/docker.sock`) |
| Tests e2e con Docker | el paquete reporta `ok` porque `stack_test.go` hace `t.Skip` cuando no hay Docker (comportamiento preexistente, fuera del alcance de esta feature) |

### Limitación de verificación

Docker no está disponible en el entorno, así que los tests de integración con
Postgres (testcontainers) fallaron de forma visible, como exige AGENTS.md, y no
se reportaron como `ok`:

```
no se pudo levantar Postgres con testcontainers: ... docker info:
permission denied while trying to connect to the docker API at
unix:///var/run/docker.sock
```

Queda pendiente correr `go test -race ./...` en un entorno con Docker.

## Operación

- Recrear el volumen para aplicar el esquema:
  `docker compose down -v && docker compose up -d` desde `backend/`.
- Schedules arrancan deshabilitados por defecto; se habilitan con
  `PUT /sync/schedules/{type}` o con los `NSP_SYNC_*_ENABLED`.
- `POST /sync/{type}/runs` responde `202`; el estado se consulta con
  `GET /sync/runs/{id}`.
- El progreso es efímero: desaparece al terminar el run o al reiniciar el
  proceso.
