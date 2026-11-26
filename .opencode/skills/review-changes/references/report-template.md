# Review report template

Write the consolidated report to `plans/<task>/review.md` using this structure. Keep the
whole document under 120 lines; the human reads the verdict and P0/P1 first.

```markdown
# Review: <task>

- Branch: `<branch>` → `<base>`
- Diff: <n> files, +<added> / -<removed>
- Date: <YYYY-MM-DD>
- Verdict: **block** | **merge with P1 tracked** | **ship**

## P0 — fix before merge

### 1. `path/to/file.go:88` — one-line summary

- Evidence: what the code does today and the path that reaches it.
- Impact: what breaks for a user or for the data.
- Fix: the minimal change, no refactor included.
- Confidence: high | medium | low
- Area: data | contract | frontend

Repeat for each P0. If there are none, write `None.`

## P1 — should fix

Same format. If none, write `None.`

## P2 — backlog

One line each: `path:line` — summary. No evidence required.

## Verified OK

- Contract: <what was checked and passed>
- Data: <what was checked and passed>
- Frontend: <what was checked and passed>
- Prohibitions: <tests untouched, no spec-kit writes, no dependency changes>

## Gaps

- <check that could not be performed and why: missing Docker, ambiguous requirement, etc.>

## Files to open

Only the files a human must read, in review order:

1. `path` — why
2. `path` — why
```
