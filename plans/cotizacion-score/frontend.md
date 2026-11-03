# Plan: cotización en el frontend

Implementación de la lectura de cotizaciones en el frontend: valor en créditos
en el catálogo y, en el detalle, cotización actual, historial y búsqueda por
fecha. No se tocó el backend: los endpoints y campos ya existían.

## Decisiones

- La columna de créditos va **solo en el catálogo** (no en el Ranking).
- La cotización actual muestra **solo el precio**.
- Se incluye el **historial** de cotizaciones como tabla.
- La sección "Cotización" va **debajo del Score**, en la columna izquierda.

## Modificaciones

### API
- `frontend/src/api/players.ts`:
  - `PlayerDetailDTO` y `PlayerRankingItemDTO` suman `price: string | null`.
  - Nuevos tipos `PlayerQuoteDTO` y `PlayerPriceDTO`.
  - Nuevas funciones `getPlayerQuotes(id)` (`GET /players/{id}/quotes`) y
    `getPlayerPrice(id, date?)` (`GET /players/{id}/price`).

### Catálogo
- `frontend/src/components/PlayerTable/PlayerTable.tsx`: columna "Créditos"
  junto a "Score", con `formatCredits(player.price)` y `—` cuando es nulo.
- `frontend/src/components/PlayerTable/PlayerTable.css`: estilo de la celda de
  créditos.

### Detalle
- `frontend/src/components/PlayerQuote/PlayerQuote.tsx` (nuevo):
  - Cotización actual (última cotización, solo precio).
  - Tabla de historial de cotizaciones (fecha + precio).
  - Búsqueda por fecha con `<input type="date">` + botón, que consulta
    `getPlayerPrice` y muestra el precio; mensaje si no hay cotización.
- `frontend/src/components/PlayerQuote/PlayerQuote.css` (nuevo): estilos BEM
  `player-quote__*`.
- `frontend/src/pages/PlayerDetailPage/PlayerDetailPage.tsx`: monta
  `<PlayerQuote playerId={player.id} />` debajo de `<PlayerScore />`.

### Tests
- `frontend/src/api/players.test.ts`: `price` en los fixtures y casos para
  `getPlayerQuotes` y `getPlayerPrice` (con y sin `date`).
- `frontend/src/components/PlayerQuote/PlayerQuote.test.tsx` (nuevo):
  cotización actual + historial, estado vacío, búsqueda por fecha y error.
- `frontend/src/pages/PlayerListPage/PlayerListPage.test.tsx`: `price` en los
  fixtures y aserción de la columna "Créditos".
- `frontend/src/pages/RankingPage/RankingPage.test.tsx`, `App.test.tsx` y
  `PlayerDetailPage.test.tsx`: fixtures con `price` y mock de los nuevos
  endpoints.

### Ajustes de visibilidad y formato
- `frontend/src/utils/format.ts`: `formatCredits` usa coma como separador
  decimal y limita la salida a un máximo de 2 decimales (presentación).
- `frontend/src/components/PlayerQuote/PlayerQuote.css`: sección con fondo
  claro, borde de acento, cotización actual en caja oscura con precio grande,
  historial con encabezado resaltado y resultado de búsqueda destacado.
- `frontend/src/components/PlayerTable/PlayerTable.tsx` + `.css`: los créditos
  del catálogo se muestran como un badge destacado.

## Verificación

- `npm run typecheck`: OK.
- `npm run lint`: OK.
- `npm run knip`: OK.
- `npm run test`: 56 tests OK.
