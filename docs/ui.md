# UI reference

What the React frontend looks like today: tokens, styling convention, inventory, routing, the
one breakpoint, which states each surface implements, and the rules that hold with their known
exceptions. Every claim is checkable against the code; citations are `path:line`.

Run the inventory from the repository root:

```bash
find frontend/src -type f \( -name '*.tsx' -o -name '*.css' \) -not -name '*.test.tsx' | sort
```

Reality check: an earlier inventory expected 12 component folders. There are 15 (11 top-level
plus 4 under `components/admin/`), each with one `.tsx` and one `.css`. Pages are 7, as expected.

## 1. Palette and tokens

`frontend/src/index.css:1-11` defines one `:root` block with the font stack, the palette, the type
scale, spacing, grid, rule, and texture tokens. It is the only `:root` in `frontend/src`. Spacing,
type, and colour come from tokens; only intrinsic sizes (table minimum widths, stamp padding) remain
literal `px`/`rem`. There is no theme switch.

| Token | Value | Role in practice |
| --- | --- | --- |
| `--press` | `#e83b20` | primary accent at 8°; stamp fills and rules |
| `--press-ink` | `#b02918` | red text on paper |
| `--gold` | `#dcb207` | foil accent at 60.5°, Navbar title and brand |
| `--green` | `#1a9e57` | positive fills at 145.5° |
| `--green-ink` | `#11703c` | positive text on paper |
| `--teal` | `#17a2b8` | secondary accent at 198°, Navbar links |
| `--violet` | `#7b2fb0` | admin accent at 283°: section header rules, session edge, logout hover |
| `--paper` | `#f4ead8` | 60% surface: sheets and text on ink |
| `--paper-deep` | `#e7d7bb` | alternating rows and wells |
| `--paper-edge` | `#d8c4a0` | die-cut edges |
| `--ink` | `#0d0b09` | 30% surface: cover, type, rules |
| `--ink-2` | `#2a2521` | cover gradient top |
| `--ink-muted` | `#5c5348` | secondary text on paper |

The palette is the golden-angle set anchored on `--press`. The old `--ink-blue`, `--golden`,
`--radiant-orange`, `--green-forest` and `--orange-green` aliases were removed in P8; every
stylesheet now uses the token names directly.

The block also defines `--t1`-`--t8` and `--display` at `index.css:6`, Fibonacci spacing
`--s1`-`--s6` at `index.css:7`, grid tokens (`--page`, `--gutter`, `--split`, `--split-rev`) at
`index.css:8`, rules and print tokens (`--rule`, `--shadow-hard`, `--radius`, `--tilt`) at
`index.css:9`, and textures (`--halftone`, its 6px `--halftone-tile`, `--grain`) at `index.css:10`.
The body tiles the paper halftone over the grain on the ink cover at `index.css:29-31`.

Typeface: Space Grotesk is the base family, declared as `--font-body` at
`frontend/src/index.css:4` and applied to `:root` at `index.css:2` and `body` at `index.css:32`;
Anton is the display face, `--font-display` at `index.css:4`, and it drives the base `h1`-`h3`
rule at `index.css:49-55`. Both are the only families loaded from Google Fonts by the `<link>` in
`frontend/index.html:6`. Space Grotesk ships 300-700 and no italic. Fonts are requested from the
network at page load; there is no self-hosted fallback.

All hard-coded colours now live in the token block. The 23 CSS files under `frontend/src` contain 13
hex literals, all inside the `:root` palette at `index.css:5`; the other 22 files are hex-free.
`body` reads `--ink` at `index.css:29` and `--paper` at `index.css:33` instead of the old `#16241b`
and `#232742`. Keep it that way: new work takes an existing token.

## 2. BEM convention

Classes are plain global CSS, one file per folder, imported by that folder's component. The
intended block name is the folder name in kebab-case, with `block__element` and
`block--modifier` for the rest. Example: `PlayerListPage` imports its stylesheet at
`frontend/src/pages/PlayerListPage/PlayerListPage.tsx:11` and renders
`className="player-list-page"` at `PlayerListPage.tsx:96`.

Verified exceptions where the block name differs from the folder:

| Folder | Actual block(s) | Citation |
| --- | --- | --- |
| `PlayerStatistics/` | `player-stats` | `frontend/src/components/PlayerStatistics/PlayerStatistics.tsx:27` |
| `PlayerTable/` | `player-table-container`, `player-table` | `frontend/src/components/PlayerTable/PlayerTable.tsx:28`, `:29` |
| `P2PCreatePage/` | `p2p-create` | `frontend/src/pages/P2PCreatePage/P2PCreatePage.tsx:124` |
| `P2PSessionPage/` | `p2p-session` | `frontend/src/pages/P2PSessionPage/P2PSessionPage.tsx:288` |
| `AdminFeedback/` | `admin-status`, `admin-dialog` | `frontend/src/components/admin/AdminFeedback/AdminFeedback.tsx:13`, `:46` |
| `PortfolioPage/` | `portfolio-page`, `portfolio-card`, `portfolio-table` | `frontend/src/pages/PortfolioPage/PortfolioPage.tsx:100`, `:115`, `:186` |

The app shell block `.app-container` lives in `frontend/src/index.css:43`, outside any folder.
`AdminFeedback` is the only folder that exports two components (`StatusMessage`,
`ConfirmDialog`), and its CSS defines two blocks. Because CSS is global, a block name is a
namespace: new blocks must not collide with the 27 existing block prefixes.

## 3. Inventory

Component files: 15 `.tsx`, 15 folders — 11 top-level and 4 under `components/admin/`. Every
folder has exactly one component `.tsx` and one `.css` (test files excluded).

| Top-level component | Responsibility |
| --- | --- |
| `AuthModal/` | Login/register dialog; validates fields and maps 401/409 to Spanish messages. |
| `Navbar/` | Top bar: view buttons, authenticated links, admin link, session email and logout. |
| `Pagination/` | Previous/next controls with page and total; renders nothing when there is at most one page and no rows. |
| `PlayerCard/` | Read-only player identity card: club, position, shirt number, league, nationality, dates. |
| `PlayerFilter/` | Search, league, club and position inputs; resets to page 1 on change. |
| `PlayerQuote/` | Current quote, quote history table, and lookup by date. |
| `PlayerScore/` | Score bar, percentile, league coefficient, strategy name, score history. |
| `PlayerStatistics/` | Latest statistics grid and history table, with a current/history tab. |
| `PlayerTable/` | Catalog table with name and score sorting, position badges and credit price. |
| `PlayerTrade/` | Buy/sell order form, market/limit choice, validation, feedback, order book. |
| `ScoreBar/` | Score percentage bar; renders `N/R` when the score is null. |

| Admin component | Responsibility |
| --- | --- |
| `AdminFeedback/` | Shared `StatusMessage` (success/error/info) and `ConfirmDialog` primitives. |
| `CoefficientSection/` | League coefficient table with per-row load, edit and save. |
| `StrategySection/` | Scoring strategy list, weight tables, and activation with confirmation. |
| `SyncSection/` | Sync schedules (enable, interval), manual runs, polling, and recent run history. |

Pages: 7 `.tsx`, 7 folders, each with one `.css`.

| Page | Responsibility |
| --- | --- |
| `AdminPage/` | Superuser gate and layout; composes the three admin sections. |
| `P2PCreatePage/` | Sell-side P2P offer form over portfolio positions; produces a shareable `?token=` link. |
| `P2PSessionPage/` | Buyer/seller view of one P2P session: countdown, accept, cancel, terminal states. |
| `PlayerDetailPage/` | Player detail layout; composes card, score, quote, statistics and trade widget. |
| `PlayerListPage/` | Catalog with filters, sortable table, pagination, loading/empty/error states. |
| `PortfolioPage/` | Cash summary, positions, open orders with cancel, and paginated transactions. |
| `RankingPage/` | Score ranking with chunked loading (100 rows per request, 25 per page). |

Outside those folders: `App.tsx` (view switch), `main.tsx` (root render),
`context/AuthContext.tsx` (JWT user state), `index.css` (tokens and shell), `api/` (seven
modules), and `utils/` (`format.ts`, `leagues.ts`, `translations.ts`). There is no shared UI
primitive folder beyond `AdminFeedback`; markup patterns are duplicated per page.

## 4. Routing

There is no router. `react-router` is not a dependency; `frontend/package.json` lists only
`axios`, `react`, and `react-dom` as dependencies. `frontend/src/App.tsx:13` declares a `View`
union of seven strings, and `App.tsx:16` holds the current view in `useState`. Navigation is a
callback:
`Navbar` receives `onNavigate` at `App.tsx:55-59`, and `App.tsx:61-94` conditionally renders one
page per view. Selecting a player sets local state (`App.tsx:36-39`); browser back/forward is
not supported, and a reload returns to the catalog except for the P2P session opened through
`?token=`.

The one URL input is the P2P token. On mount, `App.tsx:22-29` reads `?token=` from
`window.location.search` and switches to `p2p-session`. `P2PCreatePage` builds that link as
`${origin}/?token=${token}` at `frontend/src/pages/P2PCreatePage/P2PCreatePage.tsx:110-113`.
The parameter is read only on initial load; nothing removes it from the address bar after the
session ends.

## 5. Responsive

There are six media queries under `frontend/src`, all at the same breakpoint:

```bash
grep -rn "@media" frontend/src
# frontend/src/components/PlayerFilter/PlayerFilter.css:80
# frontend/src/components/PlayerTrade/PlayerTrade.css:555
# frontend/src/components/Pagination/Pagination.css:51
# frontend/src/pages/P2PSessionPage/P2PSessionPage.css:371
# frontend/src/pages/PlayerDetailPage/PlayerDetailPage.css:121
# frontend/src/pages/PlayerListPage/PlayerListPage.css:146
```

Each one is `@media (min-width: 768px)`, so 768px is the only breakpoint and mobile is the default
layout. No file has more than one media query. `PlayerDetailPage.css:121-131` switches the detail
grid from one column to `minmax(0, 1.618fr) minmax(360px, 1fr)` at `:127`, so the golden split is
exact above roughly 1000px and the trade column never drops below 360px. `PlayerListPage.css:146`
widens the catalog sheet padding and
turns its header into the `--split` grid: the title sits in the wide column (`:159`) and the subtitle
itself is the narrow column, a press block with a halftone texture in the base rule (`:49`) rotated
at `:171`. The header height is therefore the taller of the two, with no empty band.
`PlayerFilter.css:80` replaces the `auto-fit` columns of `PlayerFilter.css:3` with four equal
flexible tracks plus the auto-width reset stamp at `PlayerFilter.css:82`; the groups and the controls
carry `min-width: 0` and the controls `width: 100%` (`:18`, `:39-40`) so the row shrinks instead of
overflowing. `Pagination.css:51` puts the footer on one `auto 1fr auto` row and restores the info
line order at `Pagination.css:59`. `PlayerTrade.css:555` splits the order book from one stacked
column into `repeat(2, minmax(0, 1fr))`; below 768px the two buy/sell columns stay stacked.
`P2PSessionPage.css:371` does the same for the session receipt grid.

The catalog table has no media query: it keeps its horizontal overflow wrapper at
`PlayerTable.css:3` and its `min-width: 780px` at `PlayerTable.css:12` at every width.

Everything else adapts without breakpoints: `repeat(auto-fit, minmax(...))` grids in
`PlayerCard.css:74`, `PlayerStatistics.css:72`, `PortfolioPage.css:42`,
`StrategySection.css:132`, `SyncSection.css:43`; horizontal table scrolling in
`PlayerTable.css:3`, `RankingPage.css:100`, `PortfolioPage.css:154`, `SyncSection.css:264`,
`CoefficientSection.css:37`; and `min-width: 320px` on `body` (`frontend/src/index.css:27`).
Every previously fixed two-column grid now stacks in its base rule and splits at 768px, and the
remaining wide tables scroll inside their wrappers.

## 6. State patterns

Six states are used across the app. The table lists only what the code actually renders; "—"
means the surface has no such state.

| Surface | Loading | Empty | Error | Unauthorized | Expired | Success |
| --- | --- | --- | --- | --- | --- | --- |
| `PlayerListPage` | `PlayerListPage.tsx:111` | `:131` | `:118` (retry) | — | — | table `:147` |
| `PlayerDetailPage` | `PlayerDetailPage.tsx:121` | delegated to children | `:128` | via `PlayerTrade.tsx:75` | via `PlayerTrade.tsx:252` | grid `:142` |
| `RankingPage` | `RankingPage.tsx:70` | `:83` | `:77` (no retry) | — | — | table `:89` |
| `PortfolioPage` | flag only, `PortfolioPage.tsx:23`, `:41` | `:174`, `:252`, `:321` | `:108` | `:74` | — | `:61` |
| `P2PCreatePage` | button label only, `P2PCreatePage.tsx:267` | `:186` | `:132` | `:52` | — | `:134` |
| `P2PSessionPage` | `P2PSessionPage.tsx:102` | — | `:112`, `:303` | `:81` | `:130`, `:226` | `:182` |
| `AdminPage` | delegated to sections | delegated | delegated | `:16`, `:34` | — | delegated |
| `PlayerTrade` | book `PlayerTrade.tsx:492` | book `:500`, `:527` | `:326` | `:75` | `:252` | `:230`, `:243` |
| `PlayerQuote` | `PlayerQuote.tsx:89` | `:80`, `:91` | `:135` | — | — | `:141` |
| `PlayerStatistics` | — | `PlayerStatistics.tsx:25` | — | — | — | grid `:39` |
| `PlayerScore` | — | `PlayerScore.tsx:16` | — | — | — | `:20` |
| `ScoreBar` | — | `ScoreBar.tsx:9` (`N/R`) | — | — | — | — |
| `SyncSection` | `SyncSection.tsx:212`, `:343` | `:214`, `:345` | `:210`, `:341` | — | — | `:209` |
| `StrategySection` | `StrategySection.tsx:110` | `:112` | `:108` | — | — | `:107` |
| `CoefficientSection` | per row `CoefficientSection.tsx:148` | — | `:127` | — | — | `:126` |
| `AuthModal` | submit disabled `AuthModal.tsx:221` | — | `:148` | — | — | closes `:82` |

Facts behind the table:

- Loading. Most pages use a `role="status"` block with a spinner
  (`PlayerListPage.tsx:112`) or a plain text line (`P2PSessionPage.tsx:106`).
  `PortfolioPage` and `P2PCreatePage` keep an `isLoading` flag but render no loading UI: the
  portfolio shows zeroed cards until data arrives, and the P2P form only changes the submit
  label to `Generando Enlace...` (`P2PCreatePage.tsx:269`).
- Empty. Catalog, ranking, all three portfolio tabs, and the P2P form each have a dedicated
  block. `PlayerDetailPage` has none; an empty score, statistics list, or quote is handled by
  the child component.
- Error. Load errors and action errors are separate. `PlayerListPage` offers a retry button
  (`:121-127`); `RankingPage` and `PlayerDetailPage` do not. Mutation feedback is a single
  `feedback` object in `PortfolioPage.tsx:24` and `PlayerTrade.tsx:36`, and a `message` object in
  the admin sections (`SyncSection.tsx:64`).
- Unauthorized. Two shapes: anonymous gate (`AdminPage.tsx:16`, `PortfolioPage.tsx:74`,
  `P2PCreatePage.tsx:52`, `P2PSessionPage.tsx:81`, `PlayerTrade.tsx:75`) and privilege gate
  (`AdminPage.tsx:34`, non-superuser). Catalog and ranking are public and have no gate.
- Expired. `P2PSessionPage` maps HTTP 410 on load to a message (`:39-40`), derives `isExpired`
  from status plus countdown (`:130`), renders the expired card (`:226-244`), and maps 410 on
  accept (`:146-148`). `PlayerTrade` maps 401 to "Tu sesión ha expirado" and reopens auth
  (`PlayerTrade.tsx:252-257`). `AuthContext` drops a JWT whose `exp` has passed
  (`frontend/src/context/AuthContext.tsx:71-77`). No other surface has an expired state.
- Success. P2P creation replaces the form with a success card (`P2PCreatePage.tsx:134-183`);
  P2P accept renders a trade receipt (`P2PSessionPage.tsx:182-223`); order placement and cancel
  show inline alerts (`PlayerTrade.tsx:230`, `PortfolioPage.tsx:61`); auth success closes the
  modal (`AuthModal.tsx:82`); admin actions use `StatusMessage`.

Known gaps, stated as facts: `PortfolioPage` has no loading view; `P2PCreatePage` loads
positions with no spinner; `RankingPage` cannot retry; `PlayerDetailPage` silently degrades
failed statistics, score, and history requests to empty (`PlayerDetailPage.tsx:72-100`).
Expired or unavailable is distinguished from a generic failure only in `PlayerTrade` (401) and
`P2PSessionPage` (404, 410); every other surface collapses a rejected request into one message.

## 7. Rules and verified exceptions

### HTTP calls only through `src/api/`

```bash
grep -rn "from ['\"]axios" frontend/src --include=*.ts --include=*.tsx
# frontend/src/api/client.ts:1
```

The result is a single hit, in the shared client. All six endpoint modules import that client, and
components call typed functions such as `getPlayersRanking` (`PlayerListPage.tsx:43`) or
`acceptP2PSession` (`P2PSessionPage.tsx:136`). No component builds an API URL or imports axios.
Keep it that way: one typed function per endpoint in `src/api/<module>.ts`.

### No inline styles, with one verified exception

```bash
grep -rn "style={{" frontend/src --include=*.tsx
```

| Location | What it does | Assessment |
| --- | --- | --- |
| `frontend/src/components/ScoreBar/ScoreBar.tsx:20` | sets `width: ${width}%` | dynamic value; unavoidable without a CSS variable |

New components must not add more. Everything else, including the reserved-token note in
`PlayerTrade`, is a BEM class styled from its component stylesheet.

### UI copy is Spanish

User-facing copy is written in Spanish, and the document language is Spanish
(`frontend/index.html:2`, `lang="es"`). A few domain abbreviations stay in English: `Score`,
`Rating`, `ID`, `FOK`, `GTC`, and `N/R` (`PlayerTable.tsx:46`, `PlayerStatistics.tsx:113`,
`PlayerTrade.tsx:393`, `ScoreBar.tsx:10`). `frontend/src/utils/translations.ts` maps backend
enums (positions, stats) to Spanish labels; `frontend/src/utils/format.ts:61-74` maps order
statuses. There is no i18n library and no translation catalog: copy is written inline where it
is rendered. Never translate, paraphrase, or deduplicate these strings.

### Amounts render through `src/utils/format.ts`

`formatCredits`, `formatCreditDecimal`, and `parseCredits` are the only money formatters, used in
6 page and component files. Credits are decimal strings on the wire; `parseCredits` converts to
`number` for arithmetic (`format.ts:35-40`), `formatCredits` formats with at most two decimals and
a comma separator (`format.ts:42-54`), and `formatCreditDecimal` normalizes the API decimal string
without rounding (`format.ts:56-59`). Do not call `toFixed` to display a credit amount, and do not
build `${amount} Créditos` by hand; the label is added at the call site.

Related exceptions, not amounts: coefficients and weights use `toFixed` directly
(`CoefficientSection.tsx:151`, `StrategySection.tsx:33`, `PlayerScore.tsx:29`), and five dates
bypass `formatDateTime` with inline `toLocaleString`: `PlayerScore.tsx:50`,
`PlayerStatistics.tsx:12`, `PlayerCard.tsx:29`, `P2PSessionPage.tsx:207` (no locale argument),
and `PortfolioPage.tsx:342`. Prefer `formatDateTime` in new work.
