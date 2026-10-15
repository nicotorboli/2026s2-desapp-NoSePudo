# Plan: sincronización de estadísticas desde WhoScored (go-colly)

Feature: endpoint `POST /players/statistics/sync` (superuser) que scrapea las
estadísticas de los jugadores de las cinco ligas desde WhoScored con go-colly
y anexa una fotografía por jugador en una tabla de solo anexado. No se
modificó ni se creó ningún archivo de test.

## Modificaciones

### Dependencias
- `backend/go.mod` / `backend/go.sum`: agregada `github.com/gocolly/colly/v2`
  (v2.3.0) y sus dependencias indirectas (`go get` + `go mod tidy`).

### Esquema
- `backend/db/init.sql`: nueva tabla `player_statistics` (solo anexado: `id`
  serial, `player_id` FK a `players(id) ON DELETE CASCADE`, `goals`, `assists`,
  `shots`, `key_passes`, `dribbles`, `tackles`, `rating NUMERIC(4,2)`,
  `created_at` default NOW; sin constraint único, las filas nunca se
  actualizan) e índice `idx_player_statistics_trace (player_id, created_at DESC)`.
  Requiere recrear el volumen: `docker compose down -v && docker compose up -d`.
- `backend/cmd/server/main.go`: `player_statistics` agregada a `expectedTables`.

### Modelo
- `backend/internal/model/player_statistics.go` (nuevo): `PlayerStatistics`
  (fila persistida) y `PlayerStatisticsRow` (fila scrapeada con nombre y club).

### Adaptador (scraper)
- `backend/internal/adapters/whoscored/client.go` (nuevo): `WhoscoredClient`
  sobre `colly.Collector` con UserAgent de navegador, `StdlibContext(ctx)`,
  `ParseHTTPErrorResponse`, límite de pedidos (`LimitRule`, ~4s, parallelism 1),
  dominio `www.whoscored.com`. Visita primero la página regional de la liga
  para adquirir las cookies de sesión del proveedor (jar propio de colly) y
  después recorre las páginas del feed
  `/statisticsfeed/1/GetPlayerStatistics?...&page=N` hasta `paging.numberPages`.
  Configuración por liga (region/tournament/slug) como constantes internas.
  Errores centinela `ErrBotChallenge` y `ErrRateLimitExceeded`; detección de
  desafío por status 403/429 o marcadores `incap_ses`/`captcha` en el cuerpo.
- `backend/internal/adapters/whoscored/dto.go` (nuevo): payloads del feed y
  `parseFeedPage` con claves tolerantes (`goals`, `assists`, `totalShots`,
  `keyPasses`, `dribbles`, `tackles`, `rating`).

### Persistencia
- `backend/internal/persistence/dao/player_statistics_sql.go` (nuevo):
  `PlayerStatisticsDAO.InsertStatistics` con `COPY ... FROM STDIN` dentro de la
  transacción (sin `pq.CopyIn`, deprecado).
- `backend/internal/persistence/dao/container.go`: campo `PlayerStatistics`.
- `backend/internal/persistence/repository/player_statistics.go` (nuevo):
  `PlayerStatisticsRepository.SaveStatistics` (transacción, insert batch, sin
  logs de auditoría).
- `backend/internal/persistence/repository/container.go`: campo
  `PlayerStatistics`.

### Servicio
- `backend/internal/service/player_statistics_sync.go` (nuevo):
  `PlayerStatisticsSyncService.SyncStatistics` — por cada liga scrapea el
  adaptador, lista los jugadores activos de la liga
  (`PlayerRepository.ListPlayers` con `PlayerFilter{League, Limit: 10000}`, sin
  tocar interfaces existentes), matchea por `normalize(nombre)|normalize(club)`
  y persiste. Cuenta processed/matched/unmatched/inserted; `partial_success`
  si el proveedor falla a mitad de corrida; centinelas
  `ErrStatisticsRateLimitExceeded` y `ErrStatisticsProviderBlocked`.
- `backend/internal/service/normalize.go` (nuevo): `normalizeName` (minúsculas,
  trim y plegado de diacríticos con mapa interno, sin dependencias nuevas).
- `backend/internal/service/container.go`: campo `StatisticsSync`.

### DTO
- `backend/internal/dto/sync_statistics.go` (nuevo): `SyncStatisticsResponse`
  (status, message, totalProcessed, totalMatched, totalUnmatched,
  totalInserted).

### Controller
- `backend/internal/controller/statistics.go` (nuevo):
  `RestStatisticsSyncController.SyncStatistics` — 429 ante rate limit, 502 ante
  bloqueo del proveedor.
- `backend/internal/controller/container.go`: campo `StatisticsSync` declarado
  como puntero concreto `*RestStatisticsSyncController` (no interfaz) para que
  la tabla de rutas no entre en pánico en los tests existentes de
  `internal/server`, que construyen `controller.Container` por literal sin el
  campo nuevo; un método con receiver puntero sobre receptor nil no se
  desreferencia hasta invocar el endpoint.

### Rutas
- `backend/internal/server/routes.go`: ruta `POST /players/statistics/sync`
  con `AccessSuperuser`.

### Contrato
- `backend/api/openapi.yaml`: nuevo path `/players/statistics/sync`
  (`x-access-level: superuser`, `operationId: syncPlayerStatistics`,
  respuestas 200/401/403/429/502/500) y schema `SyncStatisticsResult`.

## Fuera de alcance
- Frontend (sin botón ni página; solo el endpoint).
- Tests: ninguno creado ni modificado.
- Endpoints GET de lectura de estadísticas.

## Verificación
- `./scripts/pre-commit.sh` (gofmt + go vet + golangci-lint v2.14.0): **pasa**.
- `go build ./...` y `go vet ./...`: **pasan**.
- `go test -race ./...`: falla un único test preexistente (ver abajo); el
  resto de los paquetes pasa, incluidos `internal/persistence/repository`,
  `internal/service` y `test/e2e` (testcontainers).

### Error de verificación (determinístico, sin modificar tests por consigna)
```
--- FAIL: TestDeclaredLevelsMatchTheSpecification
    routes_test.go:107: la ruta "POST /players/statistics/sync" está declarada
    como superuser y no figura en la especificación: agregarla al mapa de este
    caso es la decisión de dejarla expuesta
```
`internal/server/routes_test.go` mantiene un mapa exacto de rutas esperadas
(líneas 76–84) y exige editar ese mapa al agregar una ruta. La consigna de la
feature prohíbe modificar tests, por lo que se deja documentado: la ruta nueva
queda declarada y documentada en OpenAPI (el test de contrato
`TestPublishedContractMatchesTheRouteTable` pasa), pero el whitelist de
`TestDeclaredLevelsMatchTheSpecification` requiere actualizar su mapa en
`routes_test.go` agregando:
`"POST /players/statistics/sync": AccessSuperuser,`
El comando falló 3 veces con el mismo resultado y se detuvo según lo indicado.

## Notas operativas
- Tras el cambio de esquema: `docker compose down -v && docker compose up -d`
  desde `backend/` antes de levantar el servidor.
- Cada corrida del endpoint anexa un snapshot completo (~2.5k filas); es el
  comportamiento acordado para trazabilidad por timestamp.
- El scraper depende del sitio de WhoScored (anti-bot); ante bloqueo responde
  502 y deja los datos intactos.
