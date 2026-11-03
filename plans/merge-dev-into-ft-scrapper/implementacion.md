# Plan: merge `dev` (mercado de tokens) sobre `ft/scrapper`

Resolución de conflictos del merge de `dev` (`a36afe5`, feature de mercado) sobre
la rama `ft/scrapper` (scraper, sincronizaciones programadas, scoring y
backoffice). Alcance acordado: **backend completo**; frontend queda para una
pasada aparte.

## Decisiones

- Variables de entorno: se conservan las de `dev` (`NSPPSQLDS`,
  `NSPFOOTBALLDATAAPIKEY`, `NSPHOST`, `NSPPORT`).
- Catálogo público: `GET /players` y `GET /players/{id}` siguen `anonymous`.
- El bootstrap del mercado se dispara también al final de cada sincronización de
  jugadores (se preserva el gancho `WithMarketBootstrap` de `dev`).
- En los `_test.go` de e2e/sesión se conserva la sonda autenticada de HEAD
  (`POST /auth/logout`) en lugar de `GET /players/1`, que ahora es público.

## Modificaciones

### Configuración
- `backend/internal/configuration/configuration.go`: unión de `Cfg`; se agregan
  `InitialUserBalance` y los campos de sincronización; `LoadCfg` lee ambos
  grupos; los nombres de variables quedan como en `dev`; se conservan los
  helpers `initialUserBalanceFromEnv`, `syncIntervalFromEnv` y
  `syncEnabledFromEnv`.
- `backend/internal/configuration/configuration_test.go`: `NSP_HOST`/`NSP_PORT`
  pasan a `NSPHOST`/`NSPPORT`.

### Persistencia
- `backend/internal/persistence/dao/container.go`: unión de DAOs (se suman
  `Account`, `Position`, `Order`, `Trade`, `Ledger`, `Quote`; `Audit` sigue
  concreto).
- `backend/internal/persistence/dao/audit_sql.go`: se agrega la interfaz
  `AuditDAO` de `dev` (la usa `MarketRepository`).
- `backend/internal/persistence/dao/player_sql.go`: se conservan
  `GetPlayersByExternalIDs` (HEAD) y `ListByIDs` (dev); se elimina
  `GetPlayerByExternalID`, que quedó sin uso tras el merge.
- `backend/internal/persistence/dao/user_sql.go`: se conserva la convención de
  HEAD (`db`/`d`); se agregan `InsertTx` y `GetSuperuser` adaptados.
- `backend/internal/persistence/dao/postgres_test.go`: se adopta la
  infraestructura compartida de `dev` (`testsupport`, `TestMain`, `Fresh`),
  `seedUser(t, db)` de 2 argumentos, `seedPlayer` y `beginTx`.
- `backend/internal/persistence/dao/sync_run_sql_test.go` y
  `sync_schedule_sql_test.go`: llamadas a `seedUser(t, db)` sin email.
- `backend/internal/persistence/repository/container.go`: unión de repositorios;
  se agregan `Market`, `Portfolio`, `Registration`, `containerSettings`,
  `ContainerOption` y `WithInitialUserBalance`.
- `backend/internal/persistence/repository/postgres_test.go`: `startPostgres`
  devuelve `sharedPostgres.Fresh(t)`.

### Service
- `backend/internal/service/container.go`: unión de `Container`; `buildContainer`
  construye `marketMaker`, cablea `WithMarketBootstrap` al jugador, arma
  `Market`/`Portfolio` y usa `WithMarketAccounts` en `Auth`.
- `backend/internal/service/player.go`: se agregan `MarketBootstrapper`,
  `PlayerOption` y `WithMarketBootstrap`; `bootstrapMarket` se invoca al cerrar
  la sincronización (éxito y parcial).
- `backend/internal/service/player_sync.go`: eliminado (HEAD ya lo había
  reemplazado).
- `backend/internal/service/player_sync_market_test.go`: se adapta a
  `PlayerServiceImpl`/`NewPlayerService`.

### Controller
- `backend/internal/controller/container.go`: unión de controllers concretos
  (`Player`, `Sync`, `Scoring`, `Auth`, `Market`).

### Server y contrato
- `backend/cmd/server/main.go`: `expectedTables` con las tablas de ambos
  features; el bloque de superusuario ejecuta `EnsureDefaultActivation` y
  `MarketMaker.Bootstrap`.
- `backend/db/init.sql`: unión de las tablas de scoring/sync (HEAD) y de mercado
  (dev).
- `backend/api/openapi.yaml`: se resuelve el hunk de `/players` quitando el
  bloque `security` (ruta anónima).
- `backend/internal/server/routes_test.go`: mapa `expected` con la unión de rutas
  y `Market: controller.NewMarketController(nil, nil)`.
- `backend/internal/server/server_test.go`: contenedor fusionado; se eliminan los
  tests de `dev` que esperaban 401 en `/players/1`; `RutasAnonimas` se fusiona con
  `method`/`pattern`/`body`/`want`.

### Tests e2e
- `backend/test/e2e/catalog_test.go`: se conserva la versión pública de HEAD; se
  restaura el helper `accessTokenFor` (lo usa `market_test.go`); se descartan los
  tests de `dev` sobre `/players/1`.
- `backend/test/e2e/logging_test.go` y `session_test.go`: se conserva la sonda
  `POST /auth/logout`.

## Verificación

| Check | Resultado |
|---|---|
| `gofmt -l` sobre `backend/` | sin salida |
| `go build ./...` | pasa |
| `go vet ./...` | pasa |
| `./scripts/pre-commit.sh` (gofmt + vet + golangci-lint v2.14.0) | 0 issues |
| Tests no-Docker con `-race` (`configuration`, `controller`, `dto`, `httphandler`, `middleware`, `model`, `adapters`, `server`) | pasan |
| Tests con Docker (dao, repository, service con testcontainers, e2e) | no ejecutados: Docker no disponible en el entorno |

## Frontend

Resuelto en una pasada aparte; ver `plans/merge-dev-into-ft-scrapper/frontend.md`.
