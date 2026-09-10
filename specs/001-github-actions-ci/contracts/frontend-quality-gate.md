# Interface Contract: Frontend Quality Gate

**Feature Branch**: `001-github-actions-ci`  
**Contract Version**: 1.0.0  
**Target Files**: `frontend/package.json` (upon scaffolding), `.github/workflows/ci.yml` (frontend job)  

---

## 1. Scaffold Guard Contract (FR-008)

The frontend quality gate defines a conditional guard step executed on `ubuntu-latest`:

```bash
if [ ! -f "frontend/package.json" ]; then
  echo "Frontend is not yet scaffolded (frontend/package.json not found)."
  echo "Skipping frontend verification cleanly."
  echo "scaffolded=false" >> $GITHUB_OUTPUT
else
  echo "scaffolded=true" >> $GITHUB_OUTPUT
fi
```

### Invariant
- When `frontend/package.json` is missing: The `frontend` job reports success (`exit 0`), logging an informational message. The pull request status check is marked green.
- When `frontend/package.json` is present: The job transitions to executing the active validation suite.

---

## 2. Active Validation Suite Contract (Once Scaffolded)

Once the frontend is scaffolded, `frontend/package.json` MUST expose the following scripts:

```json
{
  "scripts": {
    "typecheck": "tsc --noEmit",
    "lint": "eslint .",
    "knip": "knip"
  }
}
```

### 2.1 Step Contracts

1. **`typecheck`**:
   - Command: `npm run typecheck`
   - Purpose: Validates TypeScript compilation without code emission.
   - Pass Criteria: Clean exit 0, zero type mismatch or undefined symbol errors.
2. **`lint`**:
   - Command: `npm run lint`
   - Purpose: Validates code against ESLint configuration (react, typescript-eslint).
   - Pass Criteria: Clean exit 0, zero lint violations.
3. **`knip`**:
   - Command: `npx knip`
   - Purpose: Static analysis for unused files, unused exports, and unused dependencies.
   - Pass Criteria: Clean exit 0, zero dead-code warnings.

---

## 3. Exit Code Contract

| Scenario | Scaffolded | Status | Exit Code |
|---|---|---|---|
| Unscaffolded frontend | No | `success` | `0` |
| All checks pass | Yes | `success` | `0` |
| TypeScript error | Yes | `failure` | Non-zero (`1`) |
| ESLint error | Yes | `failure` | Non-zero (`1`) |
| Knip dead-code error | Yes | `failure` | Non-zero (`1`) |
