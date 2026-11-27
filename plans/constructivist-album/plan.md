# Constructivist album — P1 (foundation) and P2 (catalog)

Design: `docs/designs/2026-11-26-constructivist-album.md`. Branch `redesign-front`. No commit made.

## P1 — foundation and cover

- `frontend/src/index.css` — rebuilt `:root`: Anton + Space Grotesk stacks, golden-angle palette (`--press` `#e83b20`, `--press-ink`, `--gold`, `--green`, `--green-ink`, `--teal`, `--violet`, `--paper`, `--paper-deep`, `--paper-edge`, `--ink` `#0d0b09`, `--ink-2`, `--ink-muted`), five legacy aliases (`--ink-blue`, `--golden`, `--radiant-orange`, `--green-forest`, `--orange-green`), type scale `--t1`-`--t8` plus `--display`, Fibonacci spacing `--s1`-`--s6`, grid tokens, rule/shadow/radius/tilt tokens, `--halftone`, `--halftone-tile`, `--grain`. Body is the ink cover with the paper halftone over the grain. Added base `h1`-`h3`, `::selection`, one global `:focus-visible`.
- `frontend/src/components/Navbar/Navbar.css` — album cover chrome: ink gradient, double gold bottom rule, Anton gold wordmark with a press-red underline, square stamp tabs (border blended from paper and ink), press-red active fill with hard shadow, gold auth stamp, paper-edge logout button, teal/gold links.
- `frontend/index.html` — font link now requests Anton and Space Grotesk; Rubik and Merriweather stay until the final pass.
- `docs/ui.md` — section 1 palette table, golden-angle wording, aliases, token families, typeface paragraph, hex counts.

## P2 — catalog

- `frontend/src/pages/PlayerListPage/PlayerListPage.css` — the catalog is a paper sheet on the cover (halftone, ink rule, press-tinted hard shadow); Anton header with a press band that becomes the rotated 38.2% column at 768px (grid areas `title band` / `subtitle band`); ink error card with a press left rule and stamp buttons; dashed empty slot; spinner in press.
- `frontend/src/components/PlayerFilter/PlayerFilter.css` — `paper-deep` band, uppercase labels, paper inputs with a press focus outline, press reset stamp, golden spans `2fr 1fr 1fr 1fr auto` at 768px.
- `frontend/src/components/PlayerTable/PlayerTable.css` — ink head with paper uppercase cells on one line, zebra ledger body scoped to `.player-table__body`, press-tinted hover, green-ink credits stamp, square position stamps on press/green/teal/gold, `min-width: 780px` inside the overflow wrapper.
- `frontend/src/components/Pagination/Pagination.css` — stacked mobile footer with the info first, stamp buttons in press, ink-muted disabled state, `auto 1fr auto` at 768px.
- `frontend/src/components/ScoreBar/ScoreBar.css` — ink-bordered `paper-deep` track, press fill, tabular value, uppercase non-italic `N/R`, `10rem` minimum so the track reads inside the table.
- `docs/ui.md` — section 5 media-query count and citations, hex count and hex-free list in section 1.

## Parent review fixes applied after the passes

- `--ink` darkened from `#171310` to `#0d0b09` so ink text on `--press` reaches 4.75:1 instead of 4.47:1.
- Navbar tab border changed from `--ink-2` (1.2:1, invisible) to a paper/ink blend at 3.4:1.
- Header media block rewritten with grid areas: the subtitle no longer landed in the band column and the band now spans both rows.
- Zebra rule changed from a bare `tbody tr:nth-child(even)` (global leak into every table) to `.player-table__body .player-table__row:nth-child(even)`.
- Mobile: navbar horizontal padding reduced to `--s3` so the three tabs fit one row, `.navbar__right` right-aligned, filter button `nowrap`, table header cells `nowrap`, table `min-width` 780px.
- Design doc and `docs/ui.md` updated for the new ink value, the 780px table width, and the shifted line numbers.

## P2 amendments after the first visual pass

- `frontend/src/components/PlayerFilter/PlayerFilter.css` — grid overflow above 768px fixed. The 768px
  template is now `repeat(4, minmax(0, 1fr)) auto`; `.player-filter__group` carries `min-width: 0`;
  the inputs and selects carry `width: 100%` and `min-width: 0`. The controls no longer force their
  intrinsic width, so the band shrinks inside the sheet instead of scrolling the page sideways. The
  equal tracks, rather than the former `2fr` search column, keep "Todas las ligas" and "Todas las
  posiciones" whole at 1200px and above; between roughly 768px and 1000px the selects still clip,
  which is the accepted trade-off of the chosen option.
- `frontend/src/pages/PlayerListPage/PlayerListPage.css` — the empty `::after` press band is gone.
  The existing subtitle element is the band at every width: press fill, halftone, ink text at 4.75:1,
  `--t3`. At 768px the header is a single `--split` row with the title in column 1 and the rotated
  subtitle block in column 2, so the header height is `max(title, caption)` instead of title plus
  subtitle plus strip.
- `docs/ui.md` section 5 and the design doc updated with the new header and filter descriptions and
  re-measured citations.

## P3 — detail page

- `frontend/src/pages/PlayerDetailPage/PlayerDetailPage.css` — rewritten as a paper sheet (halftone,
  ink rule, press-tinted hard shadow, `width: calc(100% - var(--s3) * 2)`, `flex: 1`). Back button is
  a stamp; loading spinner press on paper-edge; error card mirrors the catalog error. The 768px block
  replaces `1.05fr 1fr` with `minmax(0, 1.618fr) minmax(360px, 1fr)`, `gap: var(--s4)` and
  `min-width: 0` on both columns. Deleted the 75-line `.player-detail-page__card-col .player-card*`
  override block and the 64-line dead `__unauth*` / `__auth-button*` block (no markup renders those
  classes).
- `frontend/src/components/PlayerCard/PlayerCard.css` — album sticker: halftone paper, ink rule, 6px
  paper die-cut ring, ink offset shadow, static `-1.5deg` tilt, Anton uppercase name, press-ink club,
  square status stamps, two-column field ledger with paper-edge hairlines, stamped shirt-number slot.
  Dropped `max-width: 40%`; the detail page is the only consumer.
- `frontend/src/components/PlayerCard/PlayerCard.tsx` — one presentational class,
  `player-card__value--shirt`, appended to the existing value class.
- `frontend/src/components/PlayerScore/PlayerScore.css` — stats ledger: paper panel, ink rule, hard
  shadow, Anton heading with an ink rule, aligned breakdown with tabular numerals, ledger history
  table with scoped zebra, non-italic uppercase empty state.
- `frontend/src/components/PlayerScore/PlayerScore.tsx` — presentational scroll wrapper
  `.player-score__history-wrap` around the history table; the table carries `min-width: 520px` so it
  scrolls inside its panel at 360px instead of widening the page. No logic change.
- `docs/ui.md` — section 1 hex count (569 to 505, 16 to 14 files, nine hex-free) and section 5 detail
  grid description and citations.
- Design doc — component tree note recording the two presentational markup touches.

Accepted intermediate states: `PlayerQuote` and `PlayerStatistics` keep the old cream-card palette on
the new paper sheet until P4; `PlayerTrade` keeps its old card in the narrow column until P5. Above
roughly 1000px the split is 61.8/38.2; between 768px and that point the 360px trade floor compresses
it toward 50/50, which is deliberate.

## P4 — quote and statistics (implemented, uncommitted)

- `frontend/src/components/PlayerQuote/PlayerQuote.css` — paper slip with halftone and hard shadow;
  Anton title over an ink rule; `__current` price box on `--paper-deep` with tabular Anton price;
  scoped-zebra history ledger; search form as paper controls plus a press stamp; catalog-shaped
  error card and dashed empty slot. 13 hex and 18 alias uses removed.
- `frontend/src/components/PlayerStatistics/PlayerStatistics.css` — stats ledger matching
  `PlayerScore`; stamp tabs with a press active state (replaces the blue `#2b6cb0`); square tiles
  with a gold rating tile; tabular values; scoped-zebra history. 18 hex removed.
- `frontend/src/components/PlayerStatistics/PlayerStatistics.tsx` — one presentational wrapper,
  `.player-stats__history-wrap`, around the 13-column history table (the same pattern as
  `player-score__history-wrap` in P3).
- Review fixes: the table keeps `display: table` and scrolls through the wrapper instead of relying
  on `display: block`; the history price column is ink, because `--green-ink` on zebra
  `--paper-deep` rows is 4.35:1.

## P5 — trade (implemented, uncommitted)

- `frontend/src/components/PlayerTrade/PlayerTrade.css` — order slip: ruled header, stamp tabs,
  paper form, receipt block, order-book ledger on `--paper-deep`, catalog-shaped auth gate.
  `Merriweather` and the four `"Courier New"` blocks replaced with the display/body stacks and
  tabular numerals. One `@media (min-width: 768px)` splits `.player-trade__book-columns` into
  `repeat(2, minmax(0, 1fr))`; it stacks below. 107 hex removed.
- Review fix: the buy/sell book column titles are ink with a 3px `--green`/`--press` bottom rule
  (`--green-ink` on `--paper-deep` was 4.35:1).
- `frontend/src/components/PlayerTrade/PlayerTrade.tsx` — the inline reserved-token note becomes
  `.player-trade__reserved`, styled from the stylesheet with `--t1`, `--ink-muted` and
  `calc(var(--s1) / 2)`.

## P6 — auth, ranking and portfolio (implemented, uncommitted)

- `frontend/src/components/AuthModal/AuthModal.css` — paper ticket on the ink/grain overlay; gold top
  band replaces the `#dfaf0f` double border; stamp tabs, paper fields, press submit, paper alert
  cards, press-ink switch links. 46 hex and the `Merriweather` heading removed.
- `frontend/src/pages/RankingPage/RankingPage.css` — catalog sheet and ledger: Anton header, press
  subtitle band, ink table head, scoped zebra, position stamps on press/green/teal/gold, catalog
  loading/error/empty. 28 hex and `Merriweather` removed.
- `frontend/src/pages/PortfolioPage/PortfolioPage.css` — sheet, stamp tabs with count badge,
  `--paper-deep` stat cards, ink-head ledger with buy/sell stamps, stamp pagination footer, paper
  alerts, dashed empty slot. 66 hex and three `Merriweather` uses removed.
- `frontend/src/pages/PortfolioPage/PortfolioPage.tsx` — two inline styles moved to
  `portfolio-table__reserved` and `portfolio-page__pagination-actions`; no other markup change.
- Review fixes: P&L badges get a `--paper` chip so `--green-ink`/`--press-ink` text sits on paper
  (5.17:1 / 5.54:1) instead of `--paper-deep`.

Checks after P4-P6: `npm run typecheck`, `npm run lint`, `npm run knip`, `npm run test` (16 files,
58 tests) all pass. Inventory now 227 hex literals in 8 files, 15 hex-free; five media queries, all
`min-width: 768px`.

## P7 — P2P and admin (implemented, uncommitted)

- `frontend/src/pages/P2PCreatePage/P2PCreatePage.css` — offer docket: paper sheet, ruled header,
  paper form controls, receipt summary, share-link ticket, green success stamp, catalog-shaped
  states. 32 hex and the `outline: none` removed.
- `frontend/src/pages/P2PSessionPage/P2PSessionPage.css` — two-party receipt: paper card with
  press/teal/ink-muted/green state edges, stamp badges, Anton timer with an urgent press state,
  participant ledger, offer/grid receipt blocks, stamp actions. 43 hex removed; one
  `@media (min-width: 768px)` splits the receipt grid.
- `frontend/src/pages/AdminPage/AdminPage.css` — paper admin shell with `--violet` section rules,
  stamp nav, session well, logout stamp. 16 hex removed.
- `frontend/src/components/admin/AdminFeedback/AdminFeedback.css` — shared dialog on the ink/grain
  overlay with a violet top edge, and status strips with green/press/teal left rules. 23 hex removed.
- `frontend/src/components/admin/CoefficientSection/CoefficientSection.css` — coefficient ledger,
  paper inputs, press button; `outline: none` removed. 24 hex removed.
- `frontend/src/components/admin/StrategySection/StrategySection.css` — strategy cards with a press
  active rule, tabular weights ledger, `monospace` replaced. 22 hex removed.
- `frontend/src/components/admin/SyncSection/SyncSection.css` — schedules, on/off badges, status
  strips, progress bar, history ledger; `outline: none` removed. 54 hex removed.
- Review: no fixes needed. Class parity, scoped selectors, focus rings, violet usage (paper text on
  violet only) and contrast all verified.

## P8 — removal (implemented, uncommitted)

- `frontend/src/index.css` — deleted the five legacy aliases and dropped `"Rubik"` from
  `--font-body`. The 13 palette literals are untouched.
- `frontend/index.html` — the Google Fonts request now asks only for Anton and Space Grotesk.
- `docs/ui.md` and this design doc were updated to the final inventory.

Final inventory: 13 hex literals, all in `index.css:5`; 22 hex-free CSS files; six
`@media (min-width: 768px)` queries; no `Merriweather`, `Rubik`, `Courier New` or legacy alias
anywhere under `frontend/`.

## Status

All passes P1-P8 are implemented. The only outstanding item is the commit; `ScoreBar.tsx:20` is the
only remaining inline style in `frontend/src` (a dynamic width).

## Checks

`npm run typecheck`, `npm run lint`, `npm run knip`, `npm run test` (16 files, 58 tests) and
`npx vite build` all pass from `frontend/` after P1-P8. No test, `package.json`, backend, or
`specs/` file was touched. The `.tsx` edits are presentational only: the P3
`player-card__value--shirt` class and `player-score__history-wrap`, the P4
`player-stats__history-wrap`, and the presentational moves in `PlayerTrade.tsx` and
`PortfolioPage.tsx`. Visual
checks with Firefox headless against the built CSS at 1440px and 390px for the catalog and the
detail page, plus 768px and 900px for the filter and header; P4-P8 were reviewed statically
(class parity, hex/alias counts, scoped selectors, contrast) and against the shared build.
