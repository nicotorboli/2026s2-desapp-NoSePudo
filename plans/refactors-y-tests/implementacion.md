# Implementación — Refactors y tests pendientes

## Alcance entregado

Refactors de comportamiento por defecto, eliminación del inspector de catálogo
del backoffice y cobertura de tests pendientes del motor de sincronizaciones y
del backoffice.

## Refactors

### Estrategia activa por defecto
- `backend/internal/service/auth.go`: nuevo `Auth.Superuser(ctx, email)` para
  obtener la cuenta aprovisionada sin cambiar la firma de `EnsureSuperuser`.
- `backend/internal/service/score.go`: `defaultStrategyID = 1` y
  `ScoreService.EnsureDefaultActivation(ctx, actorID)`, idempotente: solo
  activa la #1 si no hay ninguna activación previa.
- `backend/cmd/server/main.go`: después de aprovisionar el superusuario, llama
  a `EnsureDefaultActivation` con el id de esa cuenta. Si no hay credenciales
  de superusuario, se omite.

### Coeficientes 1.0 por defecto
- `backend/internal/dto/scoring.go`: `LeagueCoefficientResponse.EffectiveFrom`
  pasa a `*string`.
- `backend/internal/controller/scoring.go`: `GET
  /scoring/league-coefficients/{league}` devuelve `200` con
  `{leagueCode, coefficient: 1.0, effectiveFrom: null}` cuando no hay
  coeficiente guardado, en vez de `404`.
- `backend/api/openapi.yaml`: se quitó el `404` y `effectiveFrom` quedó
  nullable.
- `frontend/src/api/admin.ts`: `effectiveFrom` es `string | null`; se eliminó
  `getLeagueCoefficientOptional` y `getLeagueCoefficient` quedó público.
- `frontend/src/components/admin/CoefficientSection/CoefficientSection.tsx`:
  muestra “1.0000 (por defecto)” cuando `effectiveFrom` es `null`; se agregaron
  `aria-label` a los inputs y botones.

### Schedules habilitados por defecto
- `backend/internal/service/sync_scheduler.go`: `DefaultSyncSettings` habilita
  catálogo y estadísticas.
- `backend/internal/configuration/configuration.go`: nuevo
  `syncEnabledFromEnv` con default `true`; solo `"false"` explícito apaga cada
  schedule. Se eliminó el `boolFromEnv` que quedó sin uso.
- `SyncScheduleSql.EnableLegacyDefaults`: al arrancar enciende las filas con el
  default viejo (deshabilitadas, intervalo 1440 y `updated_by IS NULL`). No
  toca configuraciones hechas por un operador (siempre dejan `updated_by`).

### Catálogo fuera del backoffice
- Se eliminó `frontend/src/components/admin/CatalogSection/`.
- `frontend/src/pages/AdminPage/AdminPage.tsx`: se quitó el import, el ancla y
  el link. El catálogo/ranking/detalle públicos siguen intactos.

## Tests agregados

### Backend
- `internal/persistence/dao/postgres_test.go`: helper de testcontainers y
  usuario semilla.
- `internal/persistence/dao/sync_schedule_sql_test.go`: `Ensure` idempotente,
  `Get`, `Set`, `ReserveDue` y `EnableLegacyDefaults`.
- `internal/persistence/dao/sync_run_sql_test.go`: `InsertRunning` con
  conflicto, `Finish`, `InsertSkipped`, `Get`, `List` con filtros, paginación y
  total, `MarkInterrupted`.
- `internal/persistence/repository/sync_run_test.go`: `Start` running vs
  skipped, `Finish`, `MarkInterrupted`.
- `internal/persistence/repository/sync_schedule_test.go`: `Ensure`, `Set`,
  `ReserveDue`.
- `internal/service/sync_run_test.go`: `Trigger` manual y cron, skipped,
  progreso efímero, mapeo partial/failed, `UpdateSchedule`, `ListSchedules`,
  `ListRuns`, `Recover`, `GetRun` inexistente.
- `internal/service/sync_scheduler_test.go`: `Start` siembra los defaults y
  dispara el catch-up.
- `internal/service/sync_scheduler_internal_test.go`: defaults habilitados,
  `defaultSchedules` y clamp de intervalo.
- `internal/service/score_test.go`: `EnsureDefaultActivation` idempotente y
  respeto por la elección del operador.
- `internal/service/auth_test.go`: `Auth.Superuser` devuelve la cuenta
  aprovisionada y reporta una cuenta inexistente.
- `internal/controller/scoring_test.go`: coeficiente por defecto y coeficiente
  guardado.
- `internal/dto/scoring_test.go`: adaptado a `effectiveFrom` puntero.
- `internal/configuration/configuration_test.go`: defaults de sync y `false`
  explícito.

### Frontend
- `src/auth/adminSession.test.ts`: decode del JWT, storage, login/logout,
  refresh single-flight y hook.
- `src/api/adminInterceptor.test.ts`: adjunta token, refresh + retry, no
  refresca en `/auth/*`, limpia sesión ante segundo 401, respeta
  `Authorization` explícito.
- `src/components/admin/CoefficientSection/CoefficientSection.test.tsx`:
  default 1.0, guardado y validación.
- `src/components/admin/StrategySection/StrategySection.test.tsx`: estrategia
  activa, pesos traducidos y activación con confirmación.
- `src/pages/AdminPage/AdminPage.test.tsx`: login, acceso restringido, panel y
  logout; verifica que “Catálogo” ya no está.

## Verificación

| Check | Resultado |
|---|---|
| `gofmt -l .` | sin salida |
| `go build ./...` | pasa |
| `go vet ./...` | pasa |
| `./scripts/pre-commit.sh` | `0 issues`, pasa |
| Tests backend sin Docker (`model`, `dto`, `controller`, `server`, `configuration`, scheduler puro) | pasan |
| Tests backend con testcontainers (DAO/repository/service) | no ejecutados: Docker no disponible (`permission denied` en `/var/run/docker.sock`) |
| `npm run typecheck` | pasa |
| `npm run lint` | pasa |
| `npm run knip` | pasa |
| `npm run test` | 11 archivos, 42 tests, pasan |
| `npm run test:coverage` | pasa (65.28% statements) |
| `npm run build` | pasa |

## Notas

- Los tests de DAO/repository/service nuevos compilan y están listos para
  correr en CI o en un entorno con Docker.
- En un volumen existente, las filas que quedaron con el default viejo
  (deshabilitadas, intervalo 1440 y sin actor) se encienden solas al arrancar.
  Las configuraciones hechas desde el backoffice se preservan.
- El default de estrategia requiere superusuario aprovisionado. Sin
  `NSP_SUPERUSER_EMAIL/PASSWORD` no se activa nada por defecto.
