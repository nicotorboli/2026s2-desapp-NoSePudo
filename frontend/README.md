# Frontend Quality Gates & Contract Scripts

This directory contains the React + TypeScript frontend application for **NoSePudo**.

## Required Scripts

Per `specs/001-github-actions-ci/contracts/frontend-quality-gate.md`, `package.json` provides the following quality gate scripts:

- `npm run typecheck`: Validates TypeScript compilation without code emission (`tsc --noEmit`).
- `npm run lint`: Validates code against ESLint rules (`eslint .`).
- `npm run knip`: Detects unused files, unused exports, and unused dependencies (`knip`).
- `npm run build`: Compiles TypeScript and builds the production bundle via Vite (`tsc && vite build`).
- `npm run dev`: Starts the local Vite development server.

## CI Execution

In GitHub Actions (`.github/workflows/ci.yml`), the `frontend` job executes:
1. `npm ci`
2. `npm run typecheck`
3. `npm run lint`
4. `npx knip`

All quality gates must exit with status `0` for pull requests to pass.
