# Testing plan — sistema de scoring (arquitectura)

Plan de tests para validar la arquitectura de scoring antes de implementar
estrategias concretas.

## Principios

1. No se mockea comportamiento salvo servicios externos. La persistencia se
   prueba contra Postgres real (testcontainers). La única pieza "fake" es
   `testScoringStrategy` (implementación de la interfaz de dominio
   `ScoringStrategy`), necesaria porque aún no hay estrategias de producción;
   no es un mock de servicio externo.
2. Se cubren casos felices, borde y negativos.
3. No se modifica código para que los tests pasen; si un test descubre un bug,
   se notifica y el usuario decide.
4. Tests simples: mucha lógica en un test indica un scope equivocado.
5. Se favorecen tests de integración multi-capa.
6. Se asevera una unidad lógica por test.

## A. Unit tests (sin Docker)

### `service/histogram_test.go` (package `service`)
- `bucketIndex`: feliz (`0.5 → 500`), borde (`0.0 → 0`, `1.0 → 999`), negativo
  (clamp fuera de rango).
- `buildHistogram`: conteo por bucket.
- `percentile`: valor esperado; histograma vacío → `0`.

### `dto/score_test.go` y `dto/scoring_test.go`
- `PlayerScoreDesdeModelo`: `score == percentile × coefficient`, copia
  `strategyId`/`strategyName`.
- `LeagueCoefficientDesdeModelo`: mapeo + `effectiveFrom` en RFC 3339.

## B. Persistencia (Docker, testcontainers + `db/init.sql`)

### `repository/player_score_test.go`
- Feliz: `Save` → `HasScoreFor` → `History`.
- Borde: `UNIQUE (player_id, strategy_id, statistics_id)` rechaza duplicado.
- Negativo: `HasScoreFor` false / `History` vacío.

### `repository/score_histogram_test.go`
- Feliz: `Save` round-trip de `buckets BIGINT[]` + `GetByScope`.
- Borde: `Get` devuelve el más reciente por scope.
- Negativo: scope sin histograma.

### `repository/strategy_activation_test.go`
- Feliz: `Activate` → `Active`.
- Borde: dos activaciones → la última.
- Negativo: sin activaciones → `found=false`.

### `repository/league_coefficient_test.go`
- Feliz: `Set` → `Current`.
- Borde: varios `Set` → el más reciente.
- Negativo: sin coeficiente → `found=false`.

## C. Servicio (Docker, con `testScoringStrategy`)

### `service/score_test.go`
- Feliz: `RecalculateAfterSync` persiste scores correctos; `GetScore` aplica coef
  on-read.
- Borde: idempotencia (recalcular 2× no duplica; A→B→A no reescribe A); jugador
  sin stats se saltea; sin estrategia activa → no-op.
- Negativo: `ActivateStrategy` con id desconocido; `SetLeagueCoefficient` ≤ 0;
  `GetScore` sin score → `ErrScoreNotFound`.

## Orden
A → B → C. A no requiere Docker; B y C requieren testcontainers.

## Nota
Es un testing plan: solo escribe tests, no modifica implementación.
