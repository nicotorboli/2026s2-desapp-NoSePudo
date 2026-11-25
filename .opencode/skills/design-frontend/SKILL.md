---
name: Design frontend change
description: Produce a one-page frontend design decision (placement, component tree, states, API functions, file checklist) before coding. Use when the user asks to design or plan a UI change.
---
# Design frontend change

Produces `docs/designs/<YYYY-MM-DD>-<topic>.md` and a short chat summary. The goal is a decision
the human can approve or reject before any code exists. This skill never writes code.

## 1. Inputs

Read only:

- `AGENTS.md` — language, frozen strings, frontend HTTP and BEM rules.
- `docs/ui.md` — sections 3 (inventory), 4 (routing), 5 (responsive), 6 (state patterns),
  7 (rules and exceptions).
- `docs/review.md`, section 4 (Frontend) — the acceptance checklist the change will face.
- `specs/<feature>/spec.md` if the user names a feature and the file exists.
- The existing frontend files the change would touch. Read them; do not edit them.

Never read `specs/*/plan.md`, `specs/*/tasks.md`, or `specs/*/research.md`. Never invoke a
`speckit-*` skill.

## 2. Ask at most three questions

Each question must carry the default you will use if the user does not answer:

| Question | Default |
| --- | --- |
| Which surface hosts the change: existing page, new page, or existing component? | the page already in the flow that triggers it |
| Does it need a new endpoint, or can it reuse a typed `src/api` function? | reuse the closest existing function |
| Is new user-facing copy required? | no new copy; if required, Spanish and listed in the design |

If the user answers none, proceed with the defaults and record them in the document. Do not ask
a fourth question.

## 3. Present options before writing

Offer 2 or 3 placements, for example: a new section on the current page; a new page with a
`View` value; a self-contained component under `components/`. Compare each on:

- files touched and whether a route is needed (`frontend/src/App.tsx:13`)
- where state lives and whether a request is duplicated
- effect on the existing page's loading/empty/error states
- behavior below and above 768px
- review surface: new API functions, new copy, new BEM blocks

Recommend one and state its cost and risk in one line. Wait for the user to choose when the
options are not clearly ordered; otherwise proceed with the recommendation.

## 4. Write the decision document

Path: `docs/designs/<YYYY-MM-DD>-<topic>.md`, using the current date and a kebab-case topic.
Fill `.opencode/skills/design-frontend/references/decision-template.md`.

- If the file exists, ask before replacing it. Offer a suffixed name instead of overwriting.
- Keep the body to one page; the file checklist is the only part allowed to extend it.
- Cite `path:line` for every claim about existing code, and verify each citation by reading
  the file at that line.

The document must answer, in this order: route and placement; component tree with reuse versus
create; state ownership; exact `src/api` signatures; the six-state matrix; BEM blocks and
tokens; 768px behavior; accessibility; new copy; file checklist; non-goals.

## 5. Stop

Stop after writing the document. No code, no edits to `frontend/src`, no `npm run` commands, no
writes under `plans/` or `specs/`, no commits. Answer in chat with: the chosen option, the file
count, the open questions the user still owns, and the document path.

## Hard rules

- One page plus the file checklist. If the design needs more, split the feature instead of
  growing the document.
- Every existing-code reference is `path:line`, verified. No guessed line numbers.
- Respect the rules in `docs/ui.md` section 7: HTTP calls only through `src/api/`, no new inline
  styles, Spanish copy, amounts through `src/utils/format.ts`.
- Spanish product strings stay Spanish; do not translate or invent Spanish variants of existing
  copy.
- Prefer reused components over new ones. A new component or BEM block needs a stated reason.
- Target at most 6 files in the checklist; more than 10 means propose a split.
- English prose throughout the document. No emojis.
- Do not read `specs` plan, tasks, or research files; do not invoke `speckit-*`.
