# Plan: estrategias de scoring como fuente de verdad en DB

Implementación de las estrategias de scoring persistidas en la base de datos
(`scoring_strategies`) y cargadas al arranque. El algoritmo de cálculo sigue en
código; lo que se mueve a la DB es la definición de cada estrategia
(`id`, `name`, `alpha`, `weights`). Incluye el descuento on-read para posición
desconocida y el refactor de la interfaz previa hacia un tipo de datos.

## Modificaciones

### Schema
- `backend/db/init.sql`: nueva tabla `scoring_strategies`
  (`id`, `name`, `alpha NUMERIC(4,3)`, `weights JSONB`, `updated_at`), creada
  antes de `scoring_strategy_activations`. Seed idempotente
  (`ON CONFLICT (id) DO NOTHING`) de:
  - `1 / rateada / α = 0.5`
  - `2 / estadistica / α = 0.2`

  con los pesos por posición. `fouls` entra como disciplina en todas las
  posiciones; el DF de `rateada` incluye además `key_passes` y `dribbles`.

### Model
- `backend/internal/model/scoring_strategy.go` (nuevo):
  - `ScoringStrategy{Name, Weights, ID, Alpha}`.
  - `Score(position, stats)`: `α·rating/10 + (1−α)·Σ w·norm`.
  - `scoreFallback`: promedio balanceado (`1/11`) de las 11 stats para posición
    desconocida.
  - `normalizeStat`: escala fija por estadística (`min(v/max, 1)`); invierte
    `fouls` y `was_dribbled`.
  - `PositionDiscount` + `UnknownPositionDiscount = 0.5`: posiciones conocidas
    (`Goalkeeper`, `Defender`, `Midfielder`, `Attacker`) → `1.0`; el resto →
    `0.5`.

### DAO
- `backend/internal/persistence/dao/scoring_strategy_sql.go` (nuevo): `List`
  (`SELECT ... ORDER BY id`, con `json.Unmarshal` de `weights`).
- `backend/internal/persistence/dao/container.go`: +`ScoringStrategy`.

### Repository
- `backend/internal/persistence/repository/scoring_strategy.go` (nuevo):
  interfaz `ScoringStrategyDAO` + `ScoringStrategyRepositoryImpl` (`List`).
- `backend/internal/persistence/repository/container.go`: +`ScoringStrategy`.

### Service
- `backend/internal/service/scoring_strategy.go`: **eliminado** (la estrategia ya
  no es una interfaz polimórfica: es data + un método `Score()`).
- `backend/internal/service/score.go`: `strategies` pasa a
  `map[int]model.ScoringStrategy`; accesos por campo (`strategy.ID`, `.Name`,
  `.Weights`); `GetScore` y `GetScoreHistory` aplican
  `model.PositionDiscount` on-read.
- `backend/internal/service/container.go`: `NewContainer(ctx, ...) (*Container, error)`;
  carga las estrategias de DB vía `repos.ScoringStrategy.List` y arma el registry.

### Arranque
- `backend/cmd/server/main.go`: `scoring_strategies` en `expectedTables`; nueva
  firma de `NewContainer` con manejo de error.

### Tests
- `backend/internal/testutil/testutil.go` (nuevo): helpers `InsertStrategy`
  (inserta en `scoring_strategies`) y `RatingStrategy` (estrategia determinista
  `Score = rating/10`).
- `backend/internal/model/scoring_strategy_test.go` (nuevo): `Score` (fórmula,
  inversión de negativas, fallback), `normalizeStat` y `PositionDiscount`.
- `backend/internal/persistence/repository/scoring_strategy_test.go` (nuevo):
  `List` con round-trip del JSONB (ids de test ≥ 100, sin asumir el seed).
- `backend/internal/service/score_test.go`: reescrito; el fixture inserta una
  estrategia con `testutil` y carga el registry desde DB; cubre coeficiente
  on-read y descuento de posición desconocida.
- `backend/internal/service/player_statistics_sync_test.go`: ajuste del tipo del
  mapa de estrategias.
- `backend/test/e2e/stack_test.go`: ajuste de la firma de `NewContainer`.

## Verificación

| Check | Resultado |
|---|---|
| `gofmt -l .` | sin salida |
| `go build ./...` | pasa |
| `go vet ./...` | pasa |
| Suite no-Docker (`dto`, `controller`, `server`, `model`, `testutil`) | pasa |
| Tests con Docker (`service`, `repository`, `e2e`) | verdes |

## Notas
- La fuente de verdad de los pesos es la DB; el algoritmo (`Score()`,
  normalización, fallback) sigue en código.
- **Hot-reload de pesos (deuda futura):** hoy se cargan al arranque; ajustar
  pesos requiere reiniciar el servicio (sin redeploy de código). Para hot-reload
  habría que pasar a carga bajo demanda (`ListStrategies(ctx) (..., error)` y
  ajustes en controller y tests de rutas).
- `strategy_id` no tiene FK hacia `scoring_strategies`; la existencia se valida
  en el service contra el registry cargado.
- Las tablas nuevas requieren `docker compose down -v && docker compose up -d`
  para aplicarse en un volumen existente.
- Conducta de idempotencia: con syncs parciales, un jugador no re-sincronizado
  conserva su percentil anterior (comparabilidad no global), por diseño.
