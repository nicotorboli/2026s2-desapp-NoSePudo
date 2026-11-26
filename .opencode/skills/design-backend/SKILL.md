---
name: Design backend change
description: Produce a one-page backend design decision (layers, invariants, contract delta, file checklist) before coding. Use when the user asks to design or plan a backend change.
---
# Design backend change

Produces a design decision document and a short chat summary. This skill writes no code and runs
no builds.

## 1. Goal

Agree on the smallest backend change that satisfies the request: which layers move, which
invariants must still hold, what changes in the HTTP contract, and exactly which files the
implementer will touch. The output is a decision the human can accept before any code exists.

## 2. Inputs

Read only:

- `AGENTS.md` for the language, spec-kit, and layering rules.
- `docs/architecture-backend.md` for the layer table, P2P walkthrough, and known asymmetries.
- `docs/domain.md` for invariants and their owning packages.
- `docs/errors.md` for sentinel ownership and status mappings.
- `specs/<feature>/spec.md` if it exists and the user names the feature.

Never read `specs/*/plan.md`, `specs/*/tasks.md`, or `specs/*/research.md`. Never invoke
`speckit-*` skills.

## 3. Procedure

1. **Scope the request.** Name the affected area (auth, player, sync, scoring, market, P2P) and
   the layers the docs say it belongs to.
2. **Ask at most three questions**, each with a stated default. Good questions resolve a contract
   ambiguity, a unit or precision choice, or an authorization boundary. If the docs answer it, do
   not ask.
3. **Present two or three options** with repo-specific trade-offs: layers touched, approximate
   file count, migration cost, failure risk, and what breaks for an existing client. Recommend
   one and say why in one sentence.
4. **Write `docs/designs/<YYYY-MM-DD>-<topic>.md`** from
   `references/decision-template.md`. If the file already exists, ask before replacing it; if the
   user prefers, choose a suffix such as `-v2`.
5. **Stop.** No code, no `go build`, no tests, no writes under `plans/`, no commits. The next step
   is human review of the decision.

## 4. Hard rules

- The decision fits one page plus the file checklist. If it does not fit, cut rejected options
  first, then background prose, never invariants or the contract delta.
- Cite `path:line` for every structural claim and verify the line before writing it. Do not guess
  line numbers; when a fact spans a block, cite the first and last line.
- Keep English prose. Product strings, UI copy, test names, and existing Spanish comments stay in
  Spanish and are quoted, never translated.
- Do not read `specs/*/{plan,tasks,research}.md`, do not invoke `speckit-*`, and do not create
  `specs/NNN-*` directories.
- Do not propose a refactor that is unrelated to the request. Document existing asymmetries as
  facts.
- No emojis. No code comments.

## 5. Template

Use `references/decision-template.md`. Its sections are mandatory; an empty section says `None.`
with one line of justification.
