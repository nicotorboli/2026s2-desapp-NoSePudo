# <YYYY-MM-DD> — <topic>

Size rule: everything above the file checklist fits one page. If it does not, cut a section or
split the feature; do not add pages. Citation rule: every reference to existing code is
`path:line`, read and verified before writing. One-line summary of the change and its user
outcome belongs here.

## Route and placement

Where the change lives and why there: existing `View` value in `frontend/src/App.tsx:13`, a new
one, or no view change. Name the page folder and the section of that page. State what happens
to the currently selected player or P2P token, if relevant.

## Component tree (reuse vs create)

```text
PageName
├── ExistingComponent        reuse, frontend/src/components/X/X.tsx:NN
└── NewComponent/            create, block new-component
```

| Node | Reuse or create | File | Reason |
| --- | --- | --- | --- |
| | | | |

## State ownership (local/context/URL)

| State | Owner | Why |
| --- | --- | --- |
| current data | local `useState` in the page | one consumer, no sharing |
| session user | `AuthContext` | read through `useAuth` |
| deep link | URL query | only if reload or sharing must work |
| selected player | `App.tsx` local state | existing pattern |

Name cleanup: intervals, timers, aborted requests, `isMounted` guards. Cite the existing pattern
being copied.

## src/api functions

Exact signatures only; no API URL strings and no direct `axios`.

```ts
getPlayerById(id: number): Promise<PlayerDetailDTO>            // frontend/src/api/players.ts:110
newFunction(input: InputDTO): Promise<OutputDTO>               // frontend/src/api/<module>.ts — new
```

State which functions are new, which are reused, and which endpoint each maps to. If a function
does not exist, the design is not ready.

## States matrix

| State | Render | Copy or behavior | Citation |
| --- | --- | --- | --- |
| loading | | | |
| empty | | | |
| error | | | |
| unauthorized | | | |
| expired | | | |
| success | | | |

Use `N/A` only when the surface genuinely cannot reach the state, and say why. Copy is Spanish,
verbatim in the cell.

## BEM blocks and tokens

New or extended block, kebab-case, equal to the component folder unless this doc says otherwise.
List elements and modifiers that will exist, the CSS file that owns them, and the tokens used
(`frontend/src/index.css:5-11`). No inline styles; if a dynamic value is unavoidable, name it and
justify it like `ScoreBar.tsx:20`.

## 768px behavior

Describe the layout below 768px and at or above it. The app has one breakpoint,
`frontend/src/pages/PlayerDetailPage/PlayerDetailPage.css:100`. If the change reuses auto-fit
grids or an overflow wrapper, name them. If no responsive change is needed, say "no change" and
why the layout already adapts.

## Accessibility

Roles and labels for new controls, keyboard path for clickable rows or cards, focus placement on
open/close, `aria-live` for async feedback, and the accessible name of icon-only buttons. Reuse
the existing patterns cited here.

## New copy

| String (Spanish) | Where |
| --- | --- |
| | |

Write "none" when the change adds no copy. Do not rephrase existing strings.

## File checklist

- [ ] `frontend/src/pages/X/X.tsx` — edit
- [ ] `frontend/src/pages/X/X.css` — edit
- [ ] `frontend/src/api/<module>.ts` — new function
- [ ] `docs/...` — only if the change alters a documented rule

Target at most 6 files. More than 10 means the change is too large; split it.

## Non-goals

- What this change deliberately does not fix, refactor, or migrate.
- Related debt recorded in `docs/ui.md` that stays untouched.
