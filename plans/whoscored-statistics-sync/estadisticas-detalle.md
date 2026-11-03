# Plan: estadísticas del jugador en la página de detalle

Feature: endpoint `GET /players/{id}/statistics` (authenticated) que devuelve
las fotos de estadísticas del jugador (más reciente primero) y sección de
estadísticas en la página de detalle del frontend con solapas "Actual" e
"Historial". No se modificó ni se creó ningún archivo de test.

## Decisiones
- Mostrar el snapshot más reciente en forma destacada y un historial con los
  anteriores (solapas "Actual" / "Historial").
- Estado vacío: "Sin estadísticas registradas" cuando el jugador no tiene filas.
- Tests: sin tocar (la whitelist de rutas vuelve a fallar; ver abajo).

## Modificaciones

### Backend
- `backend/internal/persistence/dao/player_statistics_sql.go`:
  `PlayerStatisticsDAO.ListByPlayerID(ctx, playerID)` — consulta las filas de
  `player_statistics` ordenadas por `created_at DESC`.
- `backend/internal/persistence/repository/player_statistics.go`:
  `PlayerStatisticsRepository.GetPlayerStatistics(ctx, playerID)` (delega al DAO).
- `backend/internal/dto/player_statistics.go` (nuevo):
  `PlayerStatisticsResponse` (createdAt, goals, assists, shots, keyPasses,
  dribbles, tackles, rating) y `PlayerStatisticsDesdeModelo`.
- `backend/internal/service/player_statistics.go` (nuevo):
  `PlayerStatisticsService.GetPlayerStatistics` — valida que el jugador exista
  (reusa `service.ErrNotFound` → 404) y mapea las filas. Interfaz nueva, sin
  tocar `PlayerService` ni `PlayerRepository` para no romper mocks existentes.
- `backend/internal/controller/statistics.go`: nuevo
  `RestPlayerStatisticsController.GetPlayerStatistics()` — parsea `{id}`
  (400 inválido), 404 con `ErrNotFound`, 200 con el array.
- `backend/internal/controller/container.go`: campo nuevo `PlayerStats
  *RestPlayerStatisticsController` (puntero concreto, nil-safe al armar la
  tabla de rutas, misma decisión que la entrega anterior).
- `backend/internal/service/container.go`: campo `PlayerStats
  PlayerStatisticsService`.
- `backend/internal/server/routes.go`: ruta `GET /players/{id}/statistics`
  con `AccessAuthenticated` (mismo nivel que `GET /players/{id}`).
- `backend/api/openapi.yaml`: nuevo path `/players/{id}/statistics`
  (`operationId: getPlayerStatistics`, `x-access-level: authenticated`,
  respuestas 200/400/401/404/500) y schema `PlayerStatisticsItem`.

### Frontend
- `frontend/src/api/players.ts`: `PlayerStatisticsDTO` y
  `getPlayerStatistics(id)` (una función tipada por endpoint; axios sigue solo
  en `client.ts`).
- `frontend/src/components/PlayerStatistics/PlayerStatistics.tsx` (nuevo):
  estado vacío, solapas `role="tablist"` "Actual"/"Historial" (solo si hay más
  de un snapshot), grilla de 7 métricas + rating (2 decimales) + "Actualizado:
  <fecha es-ES>" y tabla de historial. Sin `role="status"` (los tests
  existentes esperan un único elemento con ese rol).
- `frontend/src/components/PlayerStatistics/PlayerStatistics.css` (nuevo):
  clases BEM `player-stats__*`.
- `frontend/src/pages/PlayerDetailPage/PlayerDetailPage.tsx`: segundo
  `useEffect` que, al cargar el jugador, pide las estadísticas con
  `.catch(() => setStats([]))`. El catch es deliberado: los tests existentes
  (`PlayerDetailPage.test.tsx`, `App.test.tsx`) espían solo `getPlayerById`,
  así que la llamada real de estadísticas falla (sin servidor/401) y cae al
  estado vacío sin romper ninguna aserción — los tests existentes pasan sin
  modificarlos.

## Verificación
- Backend: `./scripts/pre-commit.sh` (gofmt + go vet + golangci-lint v2.14.0):
  **pasa**. `go build ./...` y `go vet ./...`: **pasan**.
- Frontend: `npm run typecheck`, `npm run lint`, `npm run knip`: **pasan**.
  `npm run test`: **4 archivos, 11 tests, todos pasan**.
  `npm run test:coverage`: pasa (statements 79.41%); la cobertura bajó porque
  el código nuevo no tiene tests, consecuencia acordada de la regla de no
  mezclar features y tests.
- `go test -race -count=1 ./...`: falla un único test preexistente (abajo); el
  resto de los paquetes pasa, incluido el test de contrato OpenAPI.

### Error de verificación (determinístico, sin modificar tests por consigna)
```
--- FAIL: TestDeclaredLevelsMatchTheSpecification
    routes_test.go:107: la ruta "POST /players/statistics/sync" está declarada
    como superuser y no figura en la especificación: agregarla al mapa de este
    caso es la decisión de dejarla expuesta
    routes_test.go:107: la ruta "GET /players/{id}/statistics" está declarada
    como authenticated y no figura en la especificación: agregarla al mapa de
    este caso es la decisión de dejarla expuesta
```
`internal/server/routes_test.go` mantiene un mapa exacto de rutas esperadas
(líneas 76–84). Las dos rutas nuevas de esta feature y de la anterior
(`POST /players/statistics/sync`, `GET /players/{id}/statistics`) no están en
ese mapa; agregarlas implica editar un test, lo cual está prohibido. El
comando falló 3 veces con el mismo resultado y se detuvo según lo indicado.
Para dejarlo verde cuando se habilite editar tests:
```
"POST /players/statistics/sync":     AccessSuperuser,
"GET /players/{id}/statistics":      AccessAuthenticated,
```

## Fuera de alcance
- Autenticación en el frontend (brecha preexistente: no hay login ni cabecera
  Authorization en el cliente).
- Estadísticas en el listado o en cualquier otra página.
- Tests: ninguno creado ni modificado.
