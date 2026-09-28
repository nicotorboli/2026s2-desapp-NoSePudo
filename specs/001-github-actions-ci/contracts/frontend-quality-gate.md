# Interface Contract: Frontend Quality Gate

**Feature Branch**: `001-github-actions-ci`  
**Contract Version**: 1.0.0  
**Target Files**: `frontend/package.json`, `.github/workflows/ci.yml` (frontend job)  

---

## 1. Active Validation Suite Contract

`frontend/package.json` MUST expose the following scripts:

```json
{
  "scripts": {
    "typecheck": "tsc --noEmit",
    "lint": "eslint .",
    "knip": "knip"
  }
}
```

### 1.1 Step Contracts

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

## 2. Exit Code Contract

| Scenario | Status | Exit Code |
|---|---|---|
| All checks pass | `success` | `0` |
| TypeScript error | `failure` | Non-zero (`1`) |
| ESLint error | `failure` | Non-zero (`1`) |
| Knip dead-code error | `failure` | Non-zero (`1`) |
