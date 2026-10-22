# Plan: matcheo tolerante entre proveedores + torneo correcto por liga

Segunda corrección del sync de estadísticas. No se creó ni modificó ningún test.

## Diagnóstico
- El scraper ya desbloqueaba (ediciones concurrentes ajustaron el transporte
  TLS), pero las 5 ligas devolvían **454 filas cada una**: con
  `tournamentOptions=2` hardcodeado, el feed devuelve siempre la Premier
  League. El valor correcto es `tournamentOptions=<tournamentId>`: verificado
  empíricamente (BL1=395, PD=477, SA=479, FL1=418, PL=454).
- `matched=0` en las 5 ligas: la clave `nombre|club` exacta nunca coincide
  porque los proveedores nombran distinto al club ("Arsenal FC" vs "Arsenal",
  "Manchester City FC" vs "Man City", "AFC Bournemouth" vs "Bournemouth").
- El "socket hang up" del cliente era por la duración del sync (~45-60s) con
  delay de 4s.

## Modificaciones

### Backend
- `backend/internal/adapters/whoscored/client.go`:
  - `feedURL`: `tournamentOptions` ahora usa el `tournamentID` de la liga (antes
    hardcodeado en "2", traía siempre datos de la Premier League).
  - Eliminado el `tlsClientRoundTripper` propio, reemplazado en la edición
    concurrente por el `http.Transport` con `DialTLSContext` de tls-client
    (quedaba código muerto y rompía el lint `unused`).
- `backend/internal/service/normalize.go`:
  - `normalizeClub`: minúsculas + fold de acentos, quita puntuación, elimina
    sufijos institucionales (`FC`, `CF`, `AFC`, `AC`, `SC`, `AS`, `SS`, `SA`,
    `RCD`, `CA`) y aplica alias conocidos (`Man City`→`manchester city`,
    `Man Utd`→`manchester united`, `Bayern München`→`bayern munich`,
    `Köln`→`cologne`, `M.Gladbach`→`monchengladbach`,
    `Internazionale`→`inter`, `Paris St Germain`→`paris saint-germain`).
- `backend/internal/service/player_statistics_sync.go`:
  - `buildMatchIndexes` arma dos índices por liga: `nombre|club normalizado` y
    `nombre único` (solo nombres que aparecen una vez en el catálogo; los
    homónimos quedan fuera para no matchear mal).
  - El matcheo prueba primero la clave club y, si falla, el nombre único.
- `backend/internal/adapters/container.go`: `whoscoredRequestDelay` 4s →
  1500ms para que el sync (~15-20s) termine antes que el timeout de un cliente
  HTTP típico.

### Verificación
- `gofmt -l`, `go build ./...`: pasan.
- `go vet` y `golangci-lint` scoped a `service`/`adapters`: 0 issues.
- `go test ./internal/service/`: pasa (tests existentes intactos).
- Simulación offline contra datos reales (catálogo en Postgres + feeds de
  WhoScored por liga): **1878/2223 filas matchean (84.5%)** —
  PL 91.6%, BL1 89.6%, PD 76.3%, SA 86.2%, FL1 79.2%. Los residuales son
  transferencias que el catálogo aún no refleja, nombres cortos/nicknames y
  jugadores inactivos: quedan contados en `totalUnmatched`, nunca como
  matcheo falso.

## Fuera de alcance
- Tests: ninguno creado ni modificado (los fallos conocidos de
  `internal/controller`/`internal/server` y el de `internal/configuration`
  por ediciones concurrentes siguen ignorados por decisión del usuario).
