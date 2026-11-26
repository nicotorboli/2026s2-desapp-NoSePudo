# 2026-11-26 — Constructivist album cover and catalog pass

Replace the current dark-green reskin with an ink album cover and paper sticker sheets: golden-angle
palette anchored on `#e83b20`, 60/30/10 surface proportion, 61.8/38.2 splits at 768px, Anton
display type over Space Grotesk. P1 lays the tokens and the shell; P2 applies them to the catalog.
No behavior, data, copy, route, or test change.

## Route and placement

No route change: the `View` union stays as declared at `frontend/src/App.tsx:13` and its state stays
at `App.tsx:16-18`. The `?token=` read (`App.tsx:22-29`) is untouched.

P1 styles the shell: the `:root` block at `frontend/src/index.css:1-12`, the `body` rules at
`index.css:26-35`, `.app-container` at `index.css:43-48`, and the whole Navbar block at
`frontend/src/components/Navbar/Navbar.css:1-175`.

P2 styles the catalog surface: `frontend/src/pages/PlayerListPage/PlayerListPage.css:1-163` and the
three components it composes, `PlayerFilter.css:1-66`, `PlayerTable.css:1-107`, `Pagination.css:1-36`.
The catalog markup at `PlayerListPage.tsx:96-164` is not edited.

## Component tree (reuse vs create)

```text
PlayerListPage                     reuse, frontend/src/pages/PlayerListPage/PlayerListPage.tsx:96
├── PlayerFilter                   reuse, frontend/src/components/PlayerFilter/PlayerFilter.tsx:21
├── PlayerTable                    reuse, frontend/src/components/PlayerTable/PlayerTable.tsx:28
│   └── ScoreBar                   reuse, styled in P2 because the table renders it
└── Pagination                     reuse, frontend/src/components/Pagination/Pagination.tsx:25
```

| Node | Reuse or create | File | Reason |
| --- | --- | --- | --- |
| Navbar | reuse | `frontend/src/components/Navbar/Navbar.tsx:36` | existing block, restyled only |
| PlayerFilter | reuse | `PlayerFilter.tsx:21` | existing grid and labels |
| PlayerTable | reuse | `PlayerTable.tsx:28` | existing container and table |
| Pagination | reuse | `Pagination.tsx:25` | existing nav and two buttons |

No component, file, or BEM block is created in P1 or P2.

## State ownership (local/context/URL)

No state is added, moved, or removed.

| State | Owner | Why |
| --- | --- | --- |
| current view | `App.tsx` local `useState` | `App.tsx:16`, unchanged |
| selected player | `App.tsx` local `useState` | `App.tsx:17`, unchanged |
| filters and page | `PlayerListPage` local `useState` | one consumer, unchanged |
| session user | `AuthContext` | read in `Navbar.tsx:18`, unchanged |

No timer, interval, or request is introduced, so there is nothing to clean up.

## src/api functions

None. P1 and P2 make no request and add no function; the catalog keeps calling the existing
`getPlayersRanking`. No URL string and no axios import moves.

## States matrix

The six states keep their current markup and copy; only their colours and frames change.

| State | Render | Copy or behavior | Citation |
| --- | --- | --- | --- |
| loading | existing spinner block, paper on ink | existing "Cargando catálogo de jugadores..." | `PlayerListPage.tsx:111-116` |
| empty | existing empty sheet, stamp button | existing copy | `PlayerListPage.tsx:131-145` |
| error | existing alert card, press-red left rule | existing copy and "Reintentar" | `PlayerListPage.tsx:118-129` |
| unauthorized | N/A | the catalog is public | `docs/ui.md:192-193` |
| expired | N/A | no session on this surface | `docs/ui.md:194-198` |
| success | table plus pagination | existing rows and footer | `PlayerListPage.tsx:147-162` |

## BEM blocks and tokens

No new block. Extended blocks, all existing: `.navbar` (`Navbar.css:1`), `.player-list-page`
(`PlayerListPage.css:1`), `.player-filter` (`PlayerFilter.css:1`), `.player-table-container` and
`.player-table` (`PlayerTable.css:1`, `:9`), `.pagination` (`Pagination.css:1`). Elements and
modifiers already in the markup are reused; decoration is `::before`/`::after` with `content: ""`
and shapes only, no generated text.

Tokens added to `:root` (`index.css:1-12`):

| Token | Value | Role |
| --- | --- | --- |
| `--press` | `#e83b20` | 8°, primary accent; ink text on the fill reads 4.75:1, and as text on paper only 3.47:1 |
| `--press-ink` | `#b02918` | small red text on paper (5.6:1) |
| `--gold` | `#dcb207` | 60.5°, foil accent on ink (9.2:1) |
| `--green` | `#1a9e57` | 145.5°, fills only |
| `--green-ink` | `#11703c` | positive text on paper (5.2:1) |
| `--teal` | `#17a2b8` | 198°, secondary accent on ink (6.1:1) |
| `--violet` | `#7b2fb0` | 283°, admin accent on paper (6.1:1) |
| `--paper` | `#f4ead8` | 60% surface; 15.6:1 with ink |
| `--paper-deep` | `#e7d7bb` | alternating rows and wells |
| `--paper-edge` | `#d8c4a0` | die-cut edges |
| `--ink` | `#0d0b09` | 30% surface, type, rules |
| `--ink-2` | `#2a2521` | cover gradient |
| `--ink-muted` | `#5c5348` | secondary text on paper (7.1:1) |

Legacy names stay as aliases so the 20 unmigrated files keep resolving: `--ink-blue: var(--ink)`,
`--golden: var(--gold)`, `--radiant-orange: var(--press)`, `--green-forest: var(--green-ink)`,
`--orange-green: var(--gold)`, `--violet` and `--teal` repointed to the values above. Aliases are
removed in the final pass, not here.

Type: `--font-display: "Anton", "Arial Narrow", Impact, sans-serif` and
`--font-body: "Space Grotesk", "Rubik", system-ui, sans-serif`. Space Grotesk delivers 300-700 and no
italic (verified against the Google Fonts API), so labels cap at 700 and there are no italic
accents. Scale `--t1..--t8` at 1.272, `--display: 4.236rem`.

Space: Fibonacci `--s1..--s6` = 8/13/21/34/55/89px. Grid: `--page: 1200px`,
`--gutter: 21px`, `--split: 1.618fr 1fr`, `--split-rev: 1fr 1.618fr`. Print: `--rule: 2px solid var(--ink)`,
`--rule-thin: 1px solid var(--ink)`, `--shadow-hard: 4px 4px 0 var(--ink)`,
`--shadow-hard-sm: 2px 2px 0 var(--ink)`, `--radius: 0`, `--tilt: 1.5deg`, `--halftone`, `--grain`.
Textures are gradients; no image asset and no dependency is added.

## 768px behavior

P1: no change. The Navbar keeps its single flex row; below 768 it wraps rather than adding a
breakpoint.

P2: one new `@media (min-width: 768px)` per file at most, at the existing 768px breakpoint. Below it
the header is one column, the filter stacks, the table keeps its horizontal overflow wrapper
(`PlayerTable.css:3`), and the pagination footer stacks with full-width stamp buttons. At or above it
the header band splits `1.618fr 1fr`, the filter uses golden spans 2/1/1/1 instead of the current
`auto-fit` (`PlayerFilter.css:3`), and the pagination footer keeps its single row. `docs/ui.md:135-151`
currently claims exactly one media query in `frontend/src`; P2 updates that count.

## Accessibility

Contrast is fixed by role, not by taste: ink on paper 16.5:1, paper on ink 16.5:1, ink on press
4.75:1, gold on ink 9.2:1, teal on ink 6.1:1, `--press-ink` on paper 5.6:1, `--green-ink` on paper
5.2:1. `--press` on paper is 3.47:1, so it is used for display type, rules, and filled blocks with
ink text, never for body copy on paper. `--gold` on paper is 1.69:1 and is never text there.

Focus: one global `:focus-visible` rule in `index.css`, `outline: 3px solid var(--press)` with
`outline-offset: 2px` plus a `--paper` halo so it reads on both cover and sheet. Existing accessible
names are kept: `Navbar.tsx:43`, `Pagination.tsx:25`, `:31`, `:45`, and the row keyboard contract at
`PlayerTable.tsx:54-66`. The table keeps `role="button"`, `tabIndex`, and Enter/Space handling; no
clickable card is added. Hover tilt and lifts are wrapped in
`@media (prefers-reduced-motion: no-preference)`.

## New copy

None.

## File checklist

P1 — foundation and cover (4 files):

- [ ] `frontend/src/index.css` — token layer, aliases, base type, grain, focus ring
- [ ] `frontend/src/components/Navbar/Navbar.css` — ink cover, foil rule, stamp tabs
- [ ] `frontend/index.html` — add Anton and Space Grotesk, keep Rubik and Merriweather until the last pass
- [ ] `docs/ui.md` — section 1 palette, fonts, and token table

P2 — catalog (5 files):

- [ ] `frontend/src/pages/PlayerListPage/PlayerListPage.css` — paper sheet, golden header band, states
- [ ] `frontend/src/components/PlayerFilter/PlayerFilter.css` — filter band on paper, golden spans at 768px
- [ ] `frontend/src/components/PlayerTable/PlayerTable.css` — ledger sheet and position stamps
- [ ] `frontend/src/components/Pagination/Pagination.css` — stamp footer
- [ ] `frontend/src/components/ScoreBar/ScoreBar.css` — ink-bordered track and press fill, now that the table renders on paper

## Non-goals

- No `tsx` edit: P1 and P2 are CSS plus one `index.html` line. Markup at `PlayerListPage.tsx:96-164` stays.
- No other page or component: detail, trade, portfolio, P2P, admin, ranking, and auth modal stay on the old palette until their pass.
- No alias removal, no Rubik/Merriweather removal, no `docs/ui.md` rewrite beyond the palette and breakpoint facts.
- No test creation or edit; only regression runs.
- No backend, `src/api`, state, copy, or dependency change.
- `PortfolioPage` inline-style debt (`docs/ui.md:230-236`) is not fixed here.
