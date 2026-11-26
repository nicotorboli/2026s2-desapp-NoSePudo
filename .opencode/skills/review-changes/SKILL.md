---
name: Review changes
description: Review the current branch before commit or merge with three read-only reviewer subagents, then write a prioritized report to plans/<task>/review.md. Use when the user asks to review a branch or the current changes, or before merging.
---
# Review changes

Produces `plans/<task>/review.md` and a short chat summary. This skill never fixes code.

## 1. Scope

```bash
BASE=origin/dev
git fetch origin dev 2>/dev/null || true
git diff --stat "$BASE...HEAD"
git diff --name-only "$BASE...HEAD"
mkdir -p /tmp/opencode
git diff "$BASE...HEAD" > /tmp/opencode/review.diff
```

- If the diff exceeds 600 changed lines or 20 files, stop. Report the breakdown by area and
  propose a split. Do not launch reviewers.
- Task name: last segment of `git branch --show-current` (for example
  `ft/p2p-token-session` → `p2p-token-session`). On `dev` or `main`, ask the user for a name.

## 2. Pre-flight

```bash
cd backend && go build ./...
cd frontend && npm run typecheck
```

If either fails, write that compiler output as the only P0 and stop. Reviewing code that
does not compile produces noise.

## 3. Partition the changes

| Area | Paths |
| --- | --- |
| `data` | `backend/internal/model/`, `backend/internal/persistence/`, `backend/db/`, `backend/cmd/` |
| `contract` | `backend/api/`, `backend/internal/server/`, `backend/internal/controller/`, `backend/internal/dto/`, `backend/internal/service/`, `backend/internal/middleware/`, `backend/internal/httphandler/` |
| `frontend` | `frontend/` |

Assign anything else to the nearest area and name it explicitly in the report. Skip an area
with no changed files and say so under Gaps.

## 4. Review in parallel

Launch the three `reviewer` subagents in the same message so they run concurrently. Each
prompt contains:

- the area name and its exact file list
- the path `/tmp/opencode/review.diff`
- "follow `docs/review.md`, sections relevant to your area"
- the output template path
  `.opencode/skills/review-changes/references/report-template.md`

Do not paste file contents into the prompt; the reviewers read what they need.

## 5. Consolidate

- Deduplicate: same root cause found twice becomes one finding at the most precise location.
- Drop everything `gofmt`, `golangci-lint`, `eslint`, or `knip` already guarantee.
- Sort P0 → P1 → P2. Cap the report at 15 findings. More than 15 is itself a P0
  ("diff too large to review"), not a longer list.
- Never promote an unproven finding to P0.
- Merge the reviewers' `Verified OK` and `Gaps` sections; a check nobody performed is a gap,
  not a pass.

## 6. Write and answer

Write `plans/<task>/review.md` using `references/report-template.md`.

The chat reply contains only: the verdict, P0 and P1 as one line each, the files the human
must open, and the gaps. No restating of P2, no code blocks of the diff.

## Rules

- Do not fix anything here. Fixes happen after the human triages the report.
- Do not run the full test suite. Reviewers read code; run a single package only to prove a
  specific finding.
- Do not review a branch twice: if `plans/<task>/review.md` exists, ask whether to replace
  it or to review only the commits added since its date.
