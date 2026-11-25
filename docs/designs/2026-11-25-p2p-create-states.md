# 2026-11-25 — P2P create loading, empty, and error states

The offer form currently reports "no tokens" while the portfolio is loading and after the request fails; separate the three outcomes so a seller with tokens is not told they own none.

## Route and placement

Existing `p2p-create` view (`frontend/src/App.tsx:13`); no route or URL change. The change is inside the form branch of `frontend/src/pages/P2PCreatePage/P2PCreatePage.tsx:186-286`. The `?token=` deep link (`frontend/src/App.tsx:22-29`) is untouched.

## Component tree (reuse vs create)

```text
P2PCreatePage                    edit, frontend/src/pages/P2PCreatePage/P2PCreatePage.tsx
├── p2p-create__alert            reuse, error
├── p2p-create__loading          create element, loading
└── p2p-create__empty            reuse, empty
```

## State ownership (local/context/URL)

| State | Owner | Why |
| --- | --- | --- |
| `positions` | local `useState` | one consumer |
| `isLoading` | local `useState` | covers the portfolio load and the submit |
| `error` | local `useState` | existing single alert |
| `selectedPlayerId` | local `useState` | existing select |

`loadPositions` depends on `selectedPlayerId` (`P2PCreatePage.tsx:29-44`) and the effect depends on `loadPositions` (`:47-50`), so selecting a player refetches. Make the callback stable: drop `selectedPlayerId` from the dependencies and use `setSelectedPlayerId(previous => previous === '' ? available[0].playerId : previous)`. Otherwise a loading branch would replace the form on every selection. The copied-link timeout (`:120`) is unrelated.

## src/api functions

```ts
getUserPortfolio(userId: number): Promise<PortfolioResponseDTO>             // GET /users/{id}/portfolio, frontend/src/api/portfolio.ts:52
createP2PSession(data: CreateP2PSessionRequestDTO): Promise<P2PSessionDTO>  // POST /p2p/sessions, frontend/src/api/p2p.ts:50
```

No new function, and no API URL string in the component.

## States matrix

| State | Render | Copy or behavior | Citation |
| --- | --- | --- | --- |
| loading | `p2p-create__loading`, `role="status"` | "Cargando tus posiciones..." | new |
| empty | existing `p2p-create__empty` | existing copy and "Ir al Catálogo" | `P2PCreatePage.tsx:186-200` |
| error | existing alert with `role="alert"`; do not render empty | existing message | `P2PCreatePage.tsx:132` |
| unauthorized | existing anonymous gate | existing copy | `P2PCreatePage.tsx:52-70` |
| expired | N/A | no session exists before creation | — |
| success | existing success card | existing copy | `P2PCreatePage.tsx:134-183` |

Branches: loading when `isLoading && positions.length === 0 && !error`; empty when `!isLoading && !error && positions.length === 0`; otherwise the form. The submit path keeps its button label change (`:267-269`).

## BEM blocks and tokens

Existing block `p2p-create`. Add `.p2p-create__loading` beside `.p2p-create__empty` (`P2PCreatePage.css:199`), reusing `.p2p-create__alert--error`. Use the page's secondary text colour `#64748b`, as the empty block does; the page predates tokens and `docs/ui.md` records that debt. No inline styles.

## 768px behavior

No change. The container is `max-width: 650px` with `padding: 0 1rem` (`P2PCreatePage.css:1-5`); the new block is text plus a spinner and adds no columns.

## Accessibility

- Loading block: `role="status"` and `aria-live="polite"`, announcing progress without moving focus.
- Error alert: add `role="alert"` to the existing div at `:132`; it is currently visual only.
- Empty card: static content, no live role.
- The submit button already disables itself at `:267`.

## New copy

| String (Spanish) | Where |
| --- | --- |
| "Cargando tus posiciones..." | loading block |

No existing string is rephrased.

## File checklist

- [ ] `frontend/src/pages/P2PCreatePage/P2PCreatePage.tsx` — three branches, stable load callback, `role="alert"`.
- [ ] `frontend/src/pages/P2PCreatePage/P2PCreatePage.css` — `.p2p-create__loading`.

## Non-goals

- A retry button on the error alert; the P1 asks for separation, not recovery.
- `PortfolioPage` loading state and the other P1 findings.
- Removing the hard-coded colour literals recorded in `docs/ui.md`.
- Adding tests; this is a feature task, not a testing task.
