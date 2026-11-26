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

## Checks

`npm run typecheck`, `npm run lint`, `npm run knip`, `npm run test` (16 files, 58 tests) and `npx vite build` all pass from `frontend/` after P1, P2, the P2 amendments and P3. No test, `package.json`, backend, or `specs/` file was touched; the only `.tsx` edits are the two presentational touches listed above. Visual checks with Firefox headless against the built CSS at 1440px and 390px for the catalog and the detail page, plus 768px and 900px for the filter and header.

## Open items for the next passes

- P4: `PlayerQuote`, `PlayerStatistics`.
- P5-P7 as listed in the design doc; P8 removes the legacy aliases and the Rubik/Merriweather requests.
- `PlayerTrade` is the only remaining old cream card on the detail page.
