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

## Checks

`npm run typecheck`, `npm run lint`, `npm run knip`, `npm run test` (16 files, 58 tests) and `npx vite build` all pass from `frontend/`. No `.tsx`, test, `package.json`, backend, or `specs/` file was touched. Visual check with Firefox headless at 1440px and 390px against the built CSS.

## Open items for the next passes

- P3: `PlayerDetailPage`, `PlayerCard`, `PlayerScore`; delete the scoped `.player-card` overrides at `PlayerDetailPage.css:107-181`.
- P4-P7 as listed in the design doc; P8 removes the legacy aliases and the Rubik/Merriweather requests.
- `PlayerScore` still renders ink text on a translucent card on the detail page (pre-existing contrast debt, unchanged by P2).
