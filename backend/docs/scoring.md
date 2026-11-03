# Scoring: cómo se calculan los scores (presente e histórico)

Este documento explica el sistema de scoring del backend: cómo se asigna un
puntaje a cada jugador, cómo interviene el histograma para hacerlo relativo, y
cómo se puede reconstruir un score pasado.

## La fórmula

Todo jugador tiene un puntaje en `[0,1]`, comparable entre posiciones y ligas.
Se calcula en dos pasos y un multiplicador on-read:

1. **raw_score** (absoluto, determinista): combina las estadísticas del jugador
   ponderadas por su posición, anclado en el rating.

   ```
   raw_score = α · norm(rating) + (1-α) · Σ w[posición][stat] · norm(stat)
   ```

   - `norm(stat)` lleva cada estadística a `[0,1]` con una escala fija.
   - `w[posición][stat]` es la ponderación por posición (los pesos suman 1).
   - `α` pondera cuánto pesa el `rating`, que es comparable entre posiciones.

2. **percentil** (relativo, por histograma): convierte el raw_score en su
   posición relativa dentro de los jugadores de la misma **posición y liga**.

3. **score final**: se le aplica el coeficiente de liga on-read.

   ```
   score = percentil · coeficiente[liga]
   ```

## El histograma

El percentil no se calcula ordenando a todos los jugadores, sino con un
**histograma**: un arreglo fijo de conteos que agrupa los raw_scores en
intervalos (buckets) del rango `[0,1]`.

- Se usan **1000 buckets**, así cada bucket cubre `0.001`.
- Construirlo es una sola pasada: por cada raw_score se incrementa su bucket.
- El percentil se obtiene con las sumas acumuladas (prefix sums):

  ```
  percentil(x) = (jugadores_en_buckets_menores + 0.5 · jugadores_en_el_bucket_de_x) / total
  ```

  El `0.5` reparte los empates dentro del mismo bucket.

### Ejemplo

Con 5 jugadores y 10 buckets (para simplificar):

| Jugador | raw_score |
|---|---|
| Messi | 0.92 |
| Haaland | 0.88 |
| Saka | 0.71 |
| Bruno | 0.55 |
| Casemiro | 0.33 |

Histograma: `[0,0,0,1,0,1,0,1,1,1]` · Prefix sums: `[0,0,0,1,1,2,2,3,4,5]`.

Para Haaland (`0.88` → bucket 8):

```
percentil = (prefix[7] + 0.5 · hist[8]) / 5 = (3 + 0.5) / 5 = 0.70
```

## Cálculo del score presente

El score se recalcula en dos momentos:

1. **Tras sincronizar estadísticas**: al terminar `SyncStatistics`, se recalcula
   el score de los jugadores de la liga con la estrategia activa y las stats nuevas.
2. **Al activar una estrategia**: se recalcula bajo la estrategia nueva, pero
   solo para los `(jugador, estrategia, snapshot_de_stats)` que todavía no
   tienen score (idempotencia). Volver a una estrategia ya calculada no la toca.

En ambos casos el flujo es:

1. Para cada jugador: obtener posición y su último snapshot de estadísticas.
2. Calcular `raw_score` con la estrategia activa.
3. Agrupar por `(posición, liga)` y construir un histograma por grupo.
4. Calcular el percentil de cada jugador con su histograma.
5. Guardar el score (y el histograma) si no estaba ya calculado.

## Reconstrucción de un score pasado

Cada score se materializa en `player_scores` con:

- `strategy_id` y `weights` (la ponderación usada). La estrategia se identifica
  por `strategy_id`; el nombre es solo para display.
- `statistics_id` (el snapshot de estadísticas usado).
- `raw_score`, `percentile` y `histogram_id`.
- `computed_at`.

Con eso, un score pasado se reconstruye y verifica así:

1. **raw_score**: recomputar con las stats de `statistics_id` (append-only, siguen
   en la base) y las `weights` guardadas.
2. **percentil**: recomputar aplicando el histograma guardado (`histogram_id`)
   al raw_score.
3. **coeficiente de liga**: leer la fila de `league_coefficients` vigente en ese
   instante (append-only).
4. `score = percentil · coeficiente`.

El score materializado da la lectura rápida; la recomputación da la verificación
(trazabilidad).

## Tablas involucradas

- `scoring_strategy_activations`: timeline de la estrategia activa.
- `scoring_histograms`: histograma por `(posición, liga)` de cada recálculo.
- `player_scores`: score materializado por jugador (append-only).
- `league_coefficients`: coeficiente por liga (append-only, aplicado on-read).
