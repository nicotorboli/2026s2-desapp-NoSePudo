# Plan: desbloqueo del scraper (tls-client) + métricas ampliadas por partido

Feature: destrabar el scraper de WhoScored y ampliar las métricas persistidas.
No se creó ni modificó ningún test.

## Diagnóstico
- El seed visit a `Regions/...` recibía 403 captcha (Imperva + `verify-client.js`
  de oddschecker): era la causa del 502 inmediato.
- El feed `statisticsfeed` está detrás de Cloudflare: con el TLS de `net/http`
  (incluso con UA de Chrome) responde 403 "Just a moment...". Con fingerprint
  TLS de Chrome 131 el primer request devuelve 301 (setea `__cf_bm`) y el
  segundo, con la cookie, responde 200 con el JSON real. Verificado también
  por HTTP/1.1 (`curl_chrome131 --http1.1` → 200), así que no hace falta h2.
- El feed no trae `stats{...}`: campos planos por subcategory. `summary/offensive`
  trae goal, assistTotal, shotsPerGame, keyPassPerGame, dribbleWonPerGame;
  `summary/defensive` trae tacklePerGame, interceptionPerGame, foulsPerGame,
  clearancePerGame, outfielderBlockPerGame, wasDribbledPerGame. Ambos traen
  `rating` y `playerId`, 454 filas sin paginación (`paging.totalPages=0`).
- El parser anterior leía claves inexistentes (hubiera persistido ceros).

## Modificaciones

### Backend
- `backend/go.mod` / `go.sum`: + `github.com/bogdanfinn/tls-client` v1.16.0
  (directa) y sus dependencias (utls fork, fhttp, etc.).
- `backend/internal/adapters/whoscored/client.go` (reescrito):
  - `tlsClientRoundTripper`: RoundTripper propio que marca la conexión con
    `tls_client.GetTLSDialer()` (ClientHello Chrome 131, `WithForceHttp1`) y
    escribe/lee la request en HTTP/1.1; se inyecta con
    `collector.WithTransport(...)`.
  - Se **eliminó el seed visit** (403eaba): el handshake 301 + `__cf_bm` del
    propio feed resuelve el challenge; colly sigue redirects y su jar conserva
    la cookie.
  - Por liga se visitan 2 feeds (`summary/offensive`, `summary/defensive`);
    ~10 requests por corrida con `LimitRule` de 4s.
  - Errores iguales: 429 → `ErrRateLimitExceeded`, 403/challenge →
    `ErrBotChallenge`, `looksLikeChallenge` de respaldo.
- `backend/internal/adapters/whoscored/dto.go` (reescrito): payload plano
  `playerTableStat` con los campos reales y `mergeFeeds` que une ofensivo y
  defensivo por `playerId` (nombre/club/ataque del ofensivo; entradas,
  intercepciones, faltas, despejes, bloqueos y gambetas recibidas del
  defensivo).
- `backend/internal/model/player_statistics.go`: los valores por partido pasan
  a `float64` (Shots, KeyPasses, Dribbles, Tackles) y se agregan
  Interceptions, Fouls, Clears, Blocks, WasDribbled. Goals/Assists quedan int
  (totales de temporada, como los muestra WhoScored).
- `backend/db/init.sql`: columnas por partido en `NUMERIC(6,2)` + 5 columnas
  nuevas (`interceptions`, `fouls`, `clears`, `blocks`, `was_dribbled`).
  Requiere `docker compose down -v && docker compose up -d`.
- `backend/internal/persistence/dao/player_statistics_sql.go`: COPY actualizado
  con las 12 columnas de métricas.
- `backend/internal/service/player_statistics_sync.go`: mapeo con los campos
  nuevos.
- `backend/internal/dto/player_statistics.go`: respuesta con los campos nuevos
  (floats).
- `backend/api/openapi.yaml`: `PlayerStatisticsItem` actualizado (4 campos a
  double + 5 nuevos).

### Frontend
- `frontend/src/api/players.ts`: `PlayerStatisticsDTO` ampliado.
- `frontend/src/components/PlayerStatistics/PlayerStatistics.tsx`: 12 tiles
  (con `toFixed(2)` en los por partido) y tabla de historial con las 13
  columnas (fecha + 11 métricas + rating).

## Verificación
- `gofmt -l`, `go build ./...`: **pasan**.
- `go vet` / `golangci-lint` sobre los paquetes tocados (adapters/whoscored,
  model, dto, dao, repository, service): **pasan** (0 issues).
- Frontend `typecheck`, `lint`, `knip`, `test` (11/11): **pasan**.
- `go test -race ./...`: falla solo en paquetes ya documentados, ignorados por
  decisión del usuario:
  - `internal/controller`, `internal/server`: `[build failed]` por los mismos
    desfases de tests previos (aridad de `NewSyncController`, `stubController`).
  - `internal/configuration`: `TestLoadCfgPrefersTheEnvironmentOverTheDefaults`
    — **colateral de ediciones concurrentes ajenas a esta feature** (otro
    proceso renombró las variables de entorno en `configuration.go` a
    `NSP_HOST`/`NSP_PORT`/`NSP_PSQL_DS`/`NSP_FOOTBALLDATA_API_KEY` mientras los
    tests siguen usando los nombres viejos). También hay ediciones concurrentes
    en `routes_test.go` y `openapi.yaml` (cambio de niveles de acceso de
    `/players`). Ninguna de esas se tocó ni se revirtió.
  - Resto de paquetes (`adapters`, `footballdata`, `service`, `repository`,
    `e2e`, etc.): **pasan**.

## Notas operativas
- Tras el cambio de esquema: `docker compose down -v && docker compose up -d`
  desde `backend/`.
- El primer sync hace el handshake de Cloudflare (2 requests más lentos) y
  después va directo; la corrida completa son ~10 requests con delay de 4s.
- Si Cloudflare escalara a desafío JS completo haría falta `cf_clearance`
  (chromedp); hoy no aplica.
