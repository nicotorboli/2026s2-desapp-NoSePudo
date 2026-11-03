# Plan: merge `dev` sobre `ft/scrapper` — frontend

Resolución de los conflictos de frontend del mismo merge. Se conserva la
funcionalidad del mercado y la autenticación "real" de `dev`, y se integran las
pantallas de ranking/scoring y el backoffice de `ft/scrapper`.

## Decisiones

- **Auth**: se descarta la autenticación de prueba del backoffice
  (`adminSession` + `adminInterceptor` + `AdminLogin`) y se usa la de `dev`
  (`AuthContext` + `AuthModal` + `api/auth.ts` + refresh en `api/client.ts`).
- **Catálogo**: ranking (score + orden), tal como en `ft/scrapper`.
- **Ficha**: pública (card + score + estadísticas); sólo el widget de operación
  (`PlayerTrade`) queda detrás de login.
- **Ranking**: se mantiene `RankingPage` como vista propia del navbar.

## Archivos eliminados

- `frontend/src/auth/adminSession.ts` y `adminSession.test.ts`
- `frontend/src/api/adminInterceptor.ts` y `adminInterceptor.test.ts`
- `frontend/src/components/admin/AdminLogin/AdminLogin.tsx` y `AdminLogin.css`

## Conflictos resueltos

- `frontend/src/utils/format.ts`: unión de `formatDateTime`/`formatDuration`
  (sync y scoring) con `parseCredits`/`formatCredits`/`translateOrderStatus`
  (mercado).
- `frontend/src/api/players.ts`: se conserva la versión de `ft/scrapper`
  (tipos y funciones de ranking/score/estadísticas); sin `getPlayers`.
- `frontend/src/App.tsx`: `AuthProvider` + `AuthModal` de `dev` con un estado de
  vista `catalog | detail | ranking | admin | portfolio`; se cablean
  `RankingPage`, `AdminPage` y `PortfolioPage`.
- `frontend/src/components/Navbar/Navbar.tsx`: shell de `dev` (usuario, login,
  logout) con los ítems Catálogo, Ranking, Mi Patrimonio (autenticado) y
  Administración (superusuario).
- `frontend/src/pages/PlayerListPage/PlayerListPage.tsx`: versión de ranking de
  `ft/scrapper`; se elimina la referencia sobrante a `data` que dejó el
  auto-merge en las condiciones de carga/vacío.
- `frontend/src/pages/PlayerDetailPage/PlayerDetailPage.tsx`: ficha pública con
  `PlayerCard` + `PlayerScore` + `PlayerStatistics` + `PlayerTrade`.
- `frontend/src/pages/PlayerDetailPage/PlayerDetailPage.test.tsx`: se combinan
  los mocks de la API de scoring con un mock de `PlayerTrade`; se quita la
  expectativa de la ficha "no autenticada".

## Ediciones posteriores (fuera de los conflictos)

- `frontend/src/main.tsx`: se quita `import './api/adminInterceptor';`.
- `frontend/src/api/admin.ts`: se quitan `SessionDTO`, `login`, `refresh` y
  `logout` (los cubre `api/auth.ts` + `client.ts`); se mantienen los endpoints
  de sync y scoring.
- `frontend/src/pages/AdminPage/AdminPage.tsx`: pasa de `useAdminSession` a
  `useAuth`; se gatea por `isAuthenticated` y `user.privilege === 'superuser'`,
  y recibe `onOpenAuth` para pedir login.
- `frontend/src/pages/AdminPage/AdminPage.test.tsx`: reescrito para mockear
  `useAuth`; el caso "sin sesión" verifica el cartel de acceso restringido y que
  el botón "Iniciar Sesión" dispare `onOpenAuth`.
- `frontend/src/App.test.tsx`: se agrega un mock de `PlayerTrade`, un mock de
  `useAuth` (con `AuthProvider` real) y dos casos nuevos para la navegación
  autenticada ("Mi Patrimonio" y "Administración").

## Tests sin cambios

- `frontend/src/pages/PlayerListPage/PlayerListPage.test.tsx`: ya prueba la
  versión de ranking que se conserva.

## Verificación

| Check | Resultado |
|---|---|
| `npm run typecheck` | pasa |
| `npm run lint` | pasa |
| `npm run knip` | pasa (sin exports huérfanos) |
| `npm run test` | 14 archivos, 49 tests, todos pasan |
| Marcadores de conflicto en `frontend/` | ninguno |
