# Mini Plan: Actualización de Estilos del Panel de Administración

## Objetivo
Actualizar los estilos CSS del panel de administración (`AdminPage` y sus componentes internos: `SyncSection`, `StrategySection`, `CoefficientSection`, `AdminFeedback`) para alinearlos con la estética retro de Catálogo (`PlayerListPage`) y Ranking (`RankingPage`), incorporando el contenedor con fondo color beige/marroncito (`#dfc29d`), encabezados oscuros (`var(--ink-blue)`) con texto en verde azulado (`var(--teal)`), tipografía serif `Merriweather` en títulos, y componentes estilizados con alto contraste y armonía visual.

## Archivos modificados

1. `frontend/src/pages/AdminPage/AdminPage.css`
   - Centrado y ancho máximo unificado (`max-width: 1200px`, `margin: 0 auto`, `padding: 2rem 1.5rem`).
   - Título principal con fuente `Merriweather`, color blanco y sombra para máxima legibilidad sobre el fondo verde con rayas del sitio.
   - Subtítulo en tono gris claro verdoso (`#d2ddd4`).
   - Bloque de sesión de usuario y barra de navegación flotante adaptados al estilo de `player-filter`, con fondo translúcido verde oscuro y bordes dorados.
   - Tarjeta de acceso denegado adaptada al fondo marrón (`#dfc29d`) con bordes y botones temáticos.

2. `frontend/src/components/admin/SyncSection/SyncSection.css`
   - Contenedor con fondo marroncito (`#dfc29d`), bordes dorados tenues y sombra.
   - Tarjetas de programación (`.sync-section__card`) con fondo claro apergaminado, inputs legibles con foco dorado y badges consistentes con las píldoras de posiciones de jugadores.
   - Botón primario con gradiente naranja institucional y botón secundario claro con bordes definidos.
   - Tabla de historial reciente estilizada de forma idéntica a las tablas de catálogo y ranking (`thead` en `var(--ink-blue)`, encabezados en `var(--teal)`, filas en `#dfc29d` con hover en tinte verde oliva, y badges de estado redondeados).

3. `frontend/src/components/admin/StrategySection/StrategySection.css`
   - Contenedor con fondo marroncito (`#dfc29d`).
   - Tarjetas de estrategia con tarjeta activa destacada con borde verde bosque.
   - Grilla de pesos por posición con paneles individuales apergaminados y valores de pesos numéricos con fuente monoespaciada para óptima alineación.

4. `frontend/src/components/admin/CoefficientSection/CoefficientSection.css`
   - Contenedor con fondo marroncito (`#dfc29d`).
   - Tabla de coeficientes con el mismo patrón visual: encabezado azul marino profundo (`var(--ink-blue)`), texto en teal (`var(--teal)`), hover suave y campos de entrada numéricos con foco dorado.

5. `frontend/src/components/admin/AdminFeedback/AdminFeedback.css`
   - Mensajes de estado (`.admin-status`) con bordes y contraste reforzados.
   - Diálogo modal de confirmación (`.admin-dialog__panel`) con fondo marroncito `#dfc29d`, bordes dorados, tipografía `Merriweather` y botones temáticos.

## Verificación
- `npm run typecheck`: exitoso (0 errores).
- `npm run lint`: exitoso (0 warnings / errores).
- `npm run knip`: exitoso (0 exports sin usar).
- `npm run test`: exitoso (14/14 suites pasaron, 49 tests aprobados).
