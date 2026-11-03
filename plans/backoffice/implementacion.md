> Nota: la sección Catálogo de este plan se eliminó en la entrega de refactors y tests (`plans/refactors-y-tests/implementacion.md`).

# Implementación — Backoffice (Fase 2)

## Alcance entregado

Backoffice de administración dentro del SPA existente, con login temporal,
sin router nuevo y con una única página de scroll con secciones. Incluye:

- Sincronizaciones: estado y edición de schedules, disparo manual, polling del
  run y historial.
- Estrategias: listado, estrategia activa, alpha y pesos por posición en solo
  lectura, y activación con confirmación.
- Coeficientes por liga: las cinco ligas fijas, valor actual o "sin definir",
  edición y guardado.
- Catálogo: búsqueda de jugadores, score, historial y últimas estadísticas.

## Modificaciones

### Backend (cambio chico, sin ruta nueva)
- `backend/internal/dto/scoring.go`: `ScoringStrategyResponse` ahora expone
  `alpha` y `weights`.
- `backend/internal/service/score.go`: `ListStrategies` puebla `Alpha` y
  `Weights`.
- `backend/api/openapi.yaml`: schema `ScoringStrategy` actualizado con `alpha` y
  `weights`.

No se agregaron rutas ni se modificó el whitelist de tests.

### API y sesión
- `frontend/src/api/admin.ts` (nuevo): DTOs y una función tipada por endpoint
  para auth, sync runs, schedules, estrategias y coeficientes. Incluye
  `getLeagueCoefficientOptional` y `getActiveStrategyOptional`, que traducen el
  `404` a `null`.
- `frontend/src/api/adminInterceptor.ts` (nuevo): interceptores de axios.
  Adjunta `Authorization` con el access token y, ante `401`, refresca con
  *single-flight* y reintenta una vez. No interviene en `/auth/*`.
- `frontend/src/auth/adminSession.ts` (nuevo): almacenamiento de la sesión en
  `sessionStorage`, decodificación del claim `priv` para saber si es
  superusuario, login, logout, refresh y hook `useAdminSession`.
- `frontend/src/main.tsx`: importa el interceptor antes de montar la app.

### UI
- `frontend/src/pages/AdminPage/AdminPage.tsx` + `.css` (nuevos): página de
  administración con login, control de acceso por privilegio, header con email
  y logout, sub-nav de anclas y las cuatro secciones.
- `frontend/src/components/admin/AdminLogin/` (nuevo): formulario de login.
- `frontend/src/components/admin/AdminFeedback/` (nuevo): `StatusMessage` y
  `ConfirmDialog` compartidos.
- `frontend/src/components/admin/SyncSection/` (nuevo): schedules, disparo
  manual, progreso efímero por polling e historial de runs.
- `frontend/src/components/admin/StrategySection/` (nuevo): estrategias,
  activa, alpha y pesos traducidos, activación con confirmación.
- `frontend/src/components/admin/CoefficientSection/` (nuevo): tabla de las
  cinco ligas con guardado y nota de aplicación on-read.
- `frontend/src/components/admin/CatalogSection/` (nuevo): buscador y detalle
  reutilizando `PlayerTable`, `PlayerCard`, `PlayerScore` y `PlayerStatistics`.
- `frontend/src/components/Navbar/Navbar.tsx`: entrada "Ingresar" cuando no hay
  sesión de superusuario y "Administración" cuando la hay.
- `frontend/src/App.tsx`: nueva vista `admin`; no modifica los flujos públicos.
- `frontend/src/utils/leagues.ts` (nuevo): las cinco ligas fijas.
- `frontend/src/utils/format.ts` (nuevo): formato de fecha y duración.
- `frontend/src/utils/translations.ts`: `translateStat` para las claves de
  pesos (`key_passes`, `was_dribbled`, etc.).

No se crearon tests nuevos.

## Verificación

| Check | Resultado |
|---|---|
| `npm run typecheck` | pasa |
| `npm run lint` | pasa |
| `npm run knip` | pasa |
| `npm run test` | 6 archivos, 20 tests, pasan |
| `npm run build` | pasa |
| `./scripts/pre-commit.sh` (backend) | `0 issues`, pasa |
| `TestPublishedContractMatchesTheRouteTable` | pasa |
| Prueba manual con backend + Postgres | no ejecutada: Docker no disponible en el entorno |

## Notas y riesgos

- **Login temporal.** Vive aislado en `adminSession.ts` para reemplazarlo por
  la branch de login real. Los tokens van en `sessionStorage` (por pestaña).
- **Privilegio en el cliente.** Se decodifica el claim `priv` solo para mostrar
  u ocultar la UI; la autorización real sigue en el backend.
- **Refresh single-flight.** Varios 401 concurrentes comparten un único refresh
  para no rotar dos veces el refresh token. Duplicar una pestaña puede provocar
  un refresh con el mismo token en dos pestañas; el backend lo trata como reuso
  y una de las sesiones se cierra.
- **Entrada de login.** El Navbar muestra "Ingresar" cuando no hay sesión de
  superusuario para que la pantalla de login sea alcanzable; una vez logueado
  como superusuario pasa a "Administración".
- **Pesos de solo lectura.** El backend expone `alpha` y `weights`, pero no hay
  endpoint de edición.
- **Progreso efímero.** El avance de sync se ve solo mientras el run está
  `running`; después quedan los contadores persistidos.
