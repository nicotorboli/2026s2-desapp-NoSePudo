# Corrección: Filtrado por Estrategia Activa en Ranking y Detalle

## Problema
`LatestPercentiles` y `GetScore` consultaban los scores más recientes ordenando únicamente por `computed_at DESC, id DESC`. Cuando una estrategia ya había sido calculada previamente, alternar entre estrategias no modificaba el ranking porque la estrategia calculada más recientemente siempre ganaba la selección, ignorando cuál estaba activa en `scoring_strategy_activations`.

## Modificaciones

### Backend
1. `backend/internal/persistence/dao/player_score_sql.go`:
   - `LatestPercentiles(ctx context.Context, playerIDs []int64, strategyID ...int)`: si se pasa un `strategyID > 0`, agrega la cláusula `AND strategy_id = $2` a la consulta de percentiles.
2. `backend/internal/persistence/repository/player_score.go`:
   - Se actualiza la interfaz `PlayerScoreDAO` y el método `LatestPercentiles` para propagar el parámetro opcional `strategyID ...int`.
3. `backend/internal/service/score.go`:
   - `ScoreRepository`: actualiza la firma de `LatestPercentiles`.
   - `rankingItems`: obtiene la estrategia activa de `s.activations.Active(ctx)` y se la pasa a `LatestPercentiles`.
   - `GetScore`: busca en el historial el score que coincide con `activation.StrategyID` (o cae en `history[0]` si aún no existe).

### Verificación
- Verificado cambio de estrategia en vivo:
  - Estrategia 1 (`rateada`): Saka 50.00%, Isak 16.67%.
  - Estrategia 2 (`estadistica`): Isak 50.00%, Saka 16.67%.
- Inversión y recalculo reflejado inmediatamente en `/players/ranking`.
