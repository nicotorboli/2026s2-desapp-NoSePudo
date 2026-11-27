# 2026-11-26 — Constructivist album cover, catalog and remaining surfaces

Replace the current dark-green reskin with an ink album cover and paper sticker sheets: golden-angle
palette anchored on `#e83b20`, 60/30/10 surface proportion, 61.8/38.2 splits at 768px, Anton
display type over Space Grotesk. P1 lays the tokens and the shell; P2 applies them to the catalog;
P3 applies them to the player detail; P4-P7 carry the same cover/sheet language across the remaining
surfaces (quote and statistics, trade, auth/ranking/portfolio, P2P and admin); P8 removes the legacy
aliases and the Rubik/Merriweather requests. No behavior, data, copy, route, or test change.

## Route and placement

No route change: the `View` union stays as declared at `frontend/src/App.tsx:13` and its state stays
at `App.tsx:16-18`. The `?token=` read (`App.tsx:22-29`) is untouched.

P1 styles the shell: the `:root` block at `frontend/src/index.css:1-11`, the `body` rules at
`index.css:25-34`, `.app-container` at `index.css:42-47`, and the whole Navbar block at
`frontend/src/components/Navbar/Navbar.css:1-179`.

P2 styles the catalog surface: `frontend/src/pages/PlayerListPage/PlayerListPage.css:1-173` and the
three components it composes, `PlayerFilter.css:1-85`, `PlayerTable.css:1-116`, `Pagination.css:1-62`,
plus `ScoreBar.css:1-39`. The catalog markup at `PlayerListPage.tsx:96-164` is not edited.

P3 styles the detail surface: `frontend/src/pages/PlayerDetailPage/PlayerDetailPage.css:1-131`,
`frontend/src/components/PlayerCard/PlayerCard.css:1-117` and
`frontend/src/components/PlayerScore/PlayerScore.css:1-102`. The detail markup is not restructured
beyond the two presentational touches listed under Component tree.

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

No component, file, or BEM block is created in P1 or P2. P3 adds two presentational touches only:
`player-card__value--shirt` on the existing value span (`PlayerCard.tsx:59`) and a scroll wrapper
around the score history table (`PlayerScore.tsx:39`) so the ledger cannot widen the page at 360px.
P4-P8 add no component; P4 adds the `player-stats__history-wrap` scroll wrapper (the P3
`player-score__history-wrap` pattern) and P5-P6 move the inline styles listed under P4-P8 into
classes, and nothing else.

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
modifiers already in the markup are reused; the header caption is the existing subtitle element
restyled as the press block, not a pseudo-element, and other decoration is `::before`/`::after` with
`content: ""` and shapes only, no generated text.

Tokens added to `:root` (`index.css:1-11`):

| Token | Value | Role |
| --- | --- | --- |
| `--press` | `#e83b20` | 8°, primary accent; ink text on the fill reads 4.75:1, and as text on paper only 3.47:1 |
| `--press-ink` | `#b02918` | small red text on paper (5.5:1) |
| `--gold` | `#dcb207` | 60.5°, foil accent on ink (9.7:1) |
| `--green` | `#1a9e57` | 145.5°, fills only |
| `--green-ink` | `#11703c` | positive text on paper (5.2:1) |
| `--teal` | `#17a2b8` | 198°, secondary accent on ink (6.4:1) |
| `--violet` | `#7b2fb0` | 283°, admin accent on paper (6.1:1) |
| `--paper` | `#f4ead8` | 60% surface; 16.5:1 with ink |
| `--paper-deep` | `#e7d7bb` | alternating rows and wells |
| `--paper-edge` | `#d8c4a0` | die-cut edges |
| `--ink` | `#0d0b09` | 30% surface, type, rules |
| `--ink-2` | `#2a2521` | cover gradient |
| `--ink-muted` | `#5c5348` | secondary text on paper (6.3:1) |

The five legacy names survived as aliases through P7 so the 10 files still using them kept
resolving: `--ink-blue: var(--ink)`, `--golden: var(--gold)`, `--radiant-orange: var(--press)`,
`--green-forest: var(--green-ink)`, `--orange-green: var(--gold)`. P8 removed them; every stylesheet
now uses the token names directly.

Type: `--font-display: "Anton", "Arial Narrow", Impact, sans-serif` and
`--font-body: "Space Grotesk", system-ui, sans-serif` (the `"Rubik"` fallback was dropped in P8).
Space Grotesk delivers 300-700 and no
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
the header is one column with the subtitle as a full-width press caption, the filter stacks, the
table keeps its horizontal overflow wrapper (`PlayerTable.css:3`), and the pagination footer stacks
with full-width stamp buttons. At or above it the header is a `1.618fr 1fr` row: the title in the
wide column and the subtitle, styled as the rotated press block, in the narrow one, so the header
height is the taller of the two and no empty band remains. The filter keeps a single row of four
equal flexible tracks plus the auto-width reset stamp, with `min-width: 0` on the groups and
controls so the row shrinks instead of overflowing. The pagination footer keeps its single row.
`docs/ui.md` section 5 records the resulting count, four queries in total after P2.

P4-P8: two more queries, both `@media (min-width: 768px)`. `PlayerTrade.css` turns the book columns
into a two-track `minmax(0, 1fr)` grid and `P2PSessionPage.css` does the same for the session detail
grid; both stack below the breakpoint. No other file gains a query, so `frontend/src` ends with six
media queries and a single breakpoint. The auth dialog, ranking sheet, portfolio, P2P cards and admin
sections adapt with the `auto-fit` grids and overflow wrappers already listed in `docs/ui.md`
section 5; P6 and P7 only restyle them.

## Accessibility

Contrast is fixed by role, not by taste: ink on paper 16.5:1, paper on ink 16.5:1, ink on press
4.75:1, gold on ink 9.7:1, teal on ink 6.4:1, `--press-ink` on paper 5.5:1, `--green-ink` on paper
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

## P4-P8 — remaining surfaces

These passes finish the migration started in P1-P3. The visual grammar does not change: an ink cover,
paper sheets laid on it, `--paper-deep` wells inside the sheets, hard offset shadows instead of soft
blur, square stamps instead of rounded buttons.

Rules that hold across all five passes:

- CSS only, plus the presentational markup touches listed per pass. No copy, state, route, request,
  dependency, backend or test change.
- Existing BEM block names are reused exactly as the folders spell them: `player-quote`,
  `player-stats`, `player-trade`, `auth-modal`, `ranking-page`, `portfolio-page`, `portfolio-card`,
  `portfolio-table`, `p2p-create`, `p2p-session`, `admin-page`, `admin-dialog`, `admin-status`,
  `coefficient-section`, `strategy-section`, `sync-section`. Elements and modifiers already in the
  markup are restyled, never renamed.
- Type: `--font-display` for titles and for numerals that act as a stamp (price, total, timer,
  rank); `--font-body` for body and labels; `font-variant-numeric: tabular-nums` wherever numbers
  align in a column.
- Colour by role: `--press` fill with `--ink` text for the primary action and the active tab;
  `--green` fill with `--ink` text for positive/buy; `--teal` for informational badges;
  `--gold` as a fill for the admin accent; `--ink-muted` for secondary and disabled. `--press` as
  text on paper is forbidden (3.47:1); use `--press-ink` (5.5:1) instead, and likewise
  `--green-ink` for green text on paper, never on a green fill.
- No new token and no new hex literal. Each pass leaves its files hex-free and alias-free.
- The five legacy aliases stay defined in `:root` until P8, so the passes can land in any order.
- One breakpoint only, `@media (min-width: 768px)`, mobile first. A pass adds at most one query per
  file and never a second breakpoint.

### P4 — detail leftovers

`player-quote` and `player-stats` are the last cream cards on the detail sheet, next to `PlayerTrade`.

- Quote: paper slip with `--halftone`, ink rule, press-tinted hard shadow. `.player-quote__title` in
  Anton; `.player-quote__subtitle` as an uppercase label. `.player-quote__current` becomes the price
  box on `--paper-deep` with an ink rule, `.player-quote__price` in Anton tabular numerals and
  `.player-quote__date` in `--ink-muted`. The search row is a paper form: `.player-quote__search`,
  `.player-quote__search-label`, `.player-quote__search-button` as a press stamp. `.player-quote__history`
  and `.player-quote__history-block` become a ledger with a scoped zebra rule.
  `.player-quote__error` mirrors the catalog error card and `.player-quote__empty` the dashed slot.
- Statistics: the same ledger as `PlayerScore`, which it sits next to. `.player-stats__title` in
  Anton over an ink rule; `.player-stats__tabs`/`__tab`/`__tab--active` as stamp tabs, active press
  (this replaces the blue `#2b6cb0`); `.player-stats__grid` keeps its `auto-fit` behaviour;
  `.player-stats__tile` is a square stamp on `--paper-deep` with a bold tabular value, and
  `__tile--rating` takes `--gold`; `.player-stats__history` is a scoped-zebra ledger;
  `.player-stats__updated` is `--ink-muted`; `__empty` and `--empty` mirror the catalog slot.
- One presentational touch: the 13-column history table is wrapped in `.player-stats__history-wrap`
  and the wrapper scrolls horizontally (`min-width: 820px` on the table), the same pattern as
  `player-score__history-wrap` in P3. The table keeps `display: table`, so its semantics survive.
  No other markup change.

### P5 — trade

`player-trade` is the largest stylesheet in the app (491 lines before the pass, 107 hex) and the only
remaining card on the detail page. It becomes an order slip: a paper sheet with a reeded
`.player-trade__title`, then a receipt body.

- Tabs and actions: `.player-trade__tabs`/`__tab--buy`/`__tab--sell` as stamps (`--green` fill for
  buy, `--press` fill for sell, ink text on both); `.player-trade__submit-btn` variants follow the
  same buy/sell colours. `.player-trade__unauth*` mirrors the catalog auth gate.
- Form: `.player-trade__form`, `__field`, `__label`, `__input`, `__input-row`, `__max-btn`,
  `__radios`, `__radio-label`, `__type-hint`, `__hint` on `--paper` with ink rules and the global
  press focus ring.
- Receipt: `.player-trade__quote-box`, `__summary-box`, `__summary-row`, `__summary-row--total`
  (Anton, tabular), `__stats-row`, `__user-stats`, `__alert--error`/`--success`.
- Book: `.player-trade__book-section`, `__book-toggle`, `__book-badge`, `__book-columns`,
  `__book-column`, `__book-col-title--buy`/`--sell` (ink text with a green/press bottom rule, never
  coloured text on `--paper-deep`), `__book-content`, `__book-list`, `__book-item`,
  `__book-item--mine`, `__book-empty` as a two-column ledger on `--paper-deep`.
- Typography debt is repaid here: `Merriweather` at `:21` and the four `"Courier New"` blocks
  (`:55`, `:198`, `:278`, `:465`) become `--font-display` / `--font-body` plus tabular numerals.
- Breakpoint: `.player-trade__book-columns` is a fixed `1fr 1fr` at `:417`; it becomes
  `repeat(2, minmax(0, 1fr))` inside the file's single `min-width: 768px` query and stacks below.
- Presentational touch: the inline `fontSize`/`marginLeft` span at `PlayerTrade.tsx:292` becomes
  `.player-trade__reserved`, styled with `--t1`, `--ink-muted` and `calc(var(--s1) / 2)`.

### P6 — auth, ranking and portfolio

- `auth-modal`: a paper ticket on an ink overlay. `.auth-modal__overlay` is ink plus grain;
  `__dialog` is paper plus halftone with ink rule and hard shadow. `__tabs`/`__tab--active` are
  stamps, `__field`/`__label`/`__input` are paper controls, `__submit-btn` is a press stamp,
  `__alert--error`/`--info` are the catalog alert shapes, `__switch-link`/`--inline` use
  `--press-ink`, `__close-btn` is a stamp. The `#dfaf0f` double border at `:31` and the
  `Merriweather` heading at `:58` are removed (gold becomes `--rule` + `--gold`).
- `ranking-page`: the catalog sheet with a rank column. Anton `.ranking-page__title`,
  `.ranking-page__subtitle` as the press caption used by the catalog, `.ranking-page__table` as the
  ledger (ink head, scoped zebra, `__cell--rank` tabular), `__position--attacker/defender/goalkeeper/
  midfielder` as square stamps, `__sort`, `__row`; `__loading`/`__error`/`__empty` mirror the catalog
  states. `Merriweather` at `:14` is removed.
- `portfolio-page`: cash summary and three tabs on the same sheet. `__tabs`/`__tab`/`__tab--active`
  and `__tab-badge` as stamps; `portfolio-card` and its `--positive`/`--negative` badges as stat
  cards on `--paper-deep`; `portfolio-table` as the ledger with `--buy`/`--sell` badges and a stamp
  `__action-btn`/`--cancel`; `__pagination`/`__page-btn` as the catalog stamp footer;
  `__alert--error`/`--success`; `__empty`. The three `Merriweather` uses at `:14`, `:58`, `:244` are
  removed.
- Optional presentational touch: the inline styles at `PortfolioPage.tsx:211` (which embeds the
  literal `color: '#666'`) and `:370` move to classes. This closes the inline-style debt recorded
  in `docs/ui.md` section 6.

### P7 — P2P and admin

- `p2p-create`: the sell-offer form as a docket. Anton `__title`, `__card`, `__form-group`,
  `__label`, `__input`, `__select`, `__btn--primary`/`--secondary`, `__back-btn` as stamps;
  `__link-box`/`__link-input` as the stamped share ticket; `__summary`/`__summary-item`/`__total` as
  a receipt; `__success-card`/`__success-badge`/`__success-title`/`__success-details`/`__success-info`;
  `__empty`, `__loading`, `__alert--error`.
- `p2p-session`: one session as a two-party receipt. `__header`/`__title`; `__card` and its
  `--expired`/`--completed`/`--cancelled`/`--success`/`--center` variants; `__badge--live`/`--info`/
  `--success`/`--warning`/`--error`; `__timer`/`__timer-value`/`__timer--urgent` in Anton tabular;
  `__participants`/`__party`/`__party-arrow`/`__party-email`; `__offer-box`/`__grid-item`/
  `__grid-item--highlight`/`__total`; `__btn--primary`/`--danger`/`--secondary`; `__tag`, `__text`,
  `__alert--error`.
- Breakpoint: `.p2p-session__grid` is `repeat(2, 1fr)` at `:175`; it becomes
  `repeat(2, minmax(0, 1fr))` inside `min-width: 768px` and stacks below, closing the second fixed
  grid in `docs/ui.md` section 5.
- `admin-page`: the shell on paper. Anton `__title`, `__subtitle` as a caption, `__nav`/
  `__nav-link` as stamp tabs, `__session`/`__email`, `__logout` as a stamp, `__sections` grid,
  `__denied-card`/`--denied`. `Merriweather` at `:20`, `:137` is removed.
- `admin-dialog`/`admin-status`: shared by the sections - the dialog panel, actions and buttons as
  stamps; status banners as full-width stamped strips in `--green`/`--press`/`--teal`.
  `Merriweather` at `:50` is removed.
- `coefficient-section`: paper table and inputs, `.coefficient-section__league`/`__code` tabular,
  `__table-wrapper` keeps horizontal scrolling. `Merriweather` at `:17` is removed.
- `strategy-section`: cards with `__card--active` marked by a press left rule, `__badge`,
  `__weight-list`/`__weight`/`__weight-value` as a tabular ledger; `monospace` at `:166` becomes
  tabular body type; `Merriweather` at `:17` is removed.
- `sync-section`: `__card`, `__badge--on`/`--off`, `__status--succeeded`/`--failed`/`--running`/
  `--partial`/`--skipped` as stamped strips (green/press/teal/ink-muted), `__progress`, `__grid`/
  `__field`/`__check`, `__history` plus `__table`, `__loading`, `__message`. `Merriweather` at `:17`
  and `:218` is removed.

### P8 — removal and final pass

- `frontend/src/index.css`: the five aliases that sat at `:6` were deleted and `"Rubik"` was dropped
  from `--font-body`.
  Gate: `grep -rE "var\(--(ink-blue|golden|radiant-orange|green-forest|orange-green)" frontend/src`
  returns nothing.
- `frontend/index.html:6`: the Google Fonts link requests only Anton and Space Grotesk.
- `docs/ui.md` was updated in the same pass: alias paragraph, typeface paragraph, hex inventory,
  media-query count and debt lines. The inventory after P8 is 13 hex literals, all inside `:root`
  at `index.css:5`, one file with hex and 22 hex-free files; six media queries, all
  `min-width: 768px`.
- No `Merriweather`, `Rubik` or `Courier New` reference remains under `frontend/`.

Expected behaviour change across P4-P8: none. The states matrix, the API surface, the copy and the
routing stay exactly as documented above.

## File checklist

P1 — foundation and cover (4 files):

- [x] `frontend/src/index.css` — token layer, aliases, base type, grain, focus ring
- [x] `frontend/src/components/Navbar/Navbar.css` — ink cover, foil rule, stamp tabs
- [x] `frontend/index.html` — add Anton and Space Grotesk, keep Rubik and Merriweather until the last pass
- [x] `docs/ui.md` — section 1 palette, fonts, and token table

P2 — catalog (6 files):

- [x] `frontend/src/pages/PlayerListPage/PlayerListPage.css` — paper sheet, golden header band, states
- [x] `frontend/src/components/PlayerFilter/PlayerFilter.css` — filter band on paper, golden spans at 768px
- [x] `frontend/src/components/PlayerTable/PlayerTable.css` — ledger sheet and position stamps
- [x] `frontend/src/components/Pagination/Pagination.css` — stamp footer
- [x] `frontend/src/components/ScoreBar/ScoreBar.css` — ink-bordered track and press fill, now that the table renders on paper
- [x] `docs/ui.md` — section 5 media-query count and citations, section 1 hex inventory

P3 — detail (7 files):

- [x] `frontend/src/pages/PlayerDetailPage/PlayerDetailPage.css` — paper sheet, split grid, stamp back button, states
- [x] `frontend/src/components/PlayerCard/PlayerCard.css` — album sticker
- [x] `frontend/src/components/PlayerCard/PlayerCard.tsx` — `player-card__value--shirt` presentational class
- [x] `frontend/src/components/PlayerScore/PlayerScore.css` — stats ledger
- [x] `frontend/src/components/PlayerScore/PlayerScore.tsx` — `player-score__history-wrap` scroll wrapper
- [x] `docs/ui.md` — sections 1 and 5
- [x] this document — component tree note for the two markup touches

P4 — detail leftovers (4 files):

- [x] `frontend/src/components/PlayerQuote/PlayerQuote.css` — paper slip, price box, search form, ledger, states
- [x] `frontend/src/components/PlayerStatistics/PlayerStatistics.css` — stats ledger matching PlayerScore
- [x] `frontend/src/components/PlayerStatistics/PlayerStatistics.tsx` — `player-stats__history-wrap` scroll wrapper
- [x] `docs/ui.md` — hex inventory

P5 — trade (3 files):

- [x] `frontend/src/components/PlayerTrade/PlayerTrade.css` — order slip, receipt, book ledger, breakpoint, typography debt
- [x] `frontend/src/components/PlayerTrade/PlayerTrade.tsx` — `player-trade__reserved` class replaces the inline note
- [x] `docs/ui.md` — hex inventory, fixed-grid debt line

The reserved-token note is now `.player-trade__reserved`; `ScoreBar.tsx:20` is the only remaining
inline style in `frontend/src` (a dynamic width).

P6 — auth, ranking and portfolio (5 files):

- [x] `frontend/src/components/AuthModal/AuthModal.css` — paper ticket on the ink overlay
- [x] `frontend/src/pages/RankingPage/RankingPage.css` — catalog sheet and ledger
- [x] `frontend/src/pages/PortfolioPage/PortfolioPage.css` — tabs, stat cards, ledger, footer
- [x] `frontend/src/pages/PortfolioPage/PortfolioPage.tsx` — only the inline styles at `:211` and `:370`
- [x] `docs/ui.md` — hex inventory, inline-style debt line

P7 — P2P and admin (8 files):

- [x] `frontend/src/pages/P2PCreatePage/P2PCreatePage.css` — offer docket and success ticket
- [x] `frontend/src/pages/P2PSessionPage/P2PSessionPage.css` — two-party receipt, breakpoint
- [x] `frontend/src/pages/AdminPage/AdminPage.css` — admin shell
- [x] `frontend/src/components/admin/AdminFeedback/AdminFeedback.css` — dialog and status strips
- [x] `frontend/src/components/admin/CoefficientSection/CoefficientSection.css` — coefficient table
- [x] `frontend/src/components/admin/StrategySection/StrategySection.css` — strategy cards and weights
- [x] `frontend/src/components/admin/SyncSection/SyncSection.css` — schedules, statuses, history
- [x] `docs/ui.md` — hex inventory, media-query count, fixed-grid debt line

P8 — removal (4 files):

- [x] `frontend/src/index.css` — delete the five aliases and the Rubik fallback
- [x] `frontend/index.html` — Anton and Space Grotesk only
- [x] `docs/ui.md` — palette, typeface, hex inventory, media-query count
- [x] this document — final metrics


## Non-goals

- No `tsx` restructuring in P1-P8. P3 added two presentational touches; P5 replaced the inline
  reserved-token note with a class and P6 moved two inline styles. Detail markup at
  `PlayerDetailPage.tsx:107-162`, `PlayerCard.tsx:35-87` and `PlayerScore.tsx:14-58` stays.
- No new page, component, route, block or dependency. P4-P7 restyle the surfaces already in the
  inventory; nothing else is pulled in.
- No alias removal and no Rubik/Merriweather removal before P8. `docs/ui.md` only tracks facts that
  each pass actually changed.
- No test creation or edit; only regression runs.
- No backend, `src/api`, state, copy, or dependency change.
- `PortfolioPage` inline-style debt (`docs/ui.md` section 6) is fixed in P6, not in P1-P3.
