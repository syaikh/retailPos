# Port Configuration Centralization Plan

## Status: ✅ IMPLEMENTED (commit 3b474bf)

## Implementation Summary

All changes have been successfully implemented and tested:

### Completed Changes:
1. **`.env` & `.env.example`** - Added `FRONTEND_PORT=5173`, `BACKEND_PORT=9095`, `DATABASE_PORT=5433`
2. **`web/vite.config.js`** - Uses `dotenv` to load `.env`, configures port/proxy from env vars
3. **`web/src/lib/composables/useWebSocket.ts`** - Uses `import.meta.env.VITE_FRONTEND_PORT`/`VITE_BACKEND_PORT`
4. **`web/package.json`** - Added `dotenv` dependency
5. **`run-dev.sh` & `seed-dev.sh`** - Source `.env` file, use `$BACKEND_PORT`/`$DATABASE_PORT`
6. **`deploy/.env.example`** - Added port variables documentation
7. **`tests/e2e/fixtures.ts`** - Exports `API_BASE`, `FRONTEND_BASE`, `API_URLS` from env
8. **Test files** - Updated 5 spec files to use centralized base URLs
9. **`playwright.config.js`** - `baseURL` from `FRONTEND_BASE_URL` env var

### Test Results:
- All 23 tests pass
- 37 tests skipped (features not implemented)
- Build succeeds

## Overview
Centralize all hardcoded port values across frontend, backend, tests, and infrastructure into environment variables sourced from `.env` files, following twelve-factor app methodology.

## Goals
- Single source of truth for all port configurations
- Easy environment switching (dev/staging/prod) without code changes
- Consistent port references across all components
- Reduce duplication and human error when changing ports

## Current State Analysis

### Hardcoded Ports
| Location | Port | Purpose |
|----------|------|---------|
| `web/vite.config.js` | 5173 (dev frontend) / 8080 (was backend) | Frontend dev server & API proxy target |
| `web/src/lib/composables/useWebSocket.ts` | 9095 (dev), 8080 (was default) | WebSocket dev backend |
| `run-dev.sh` | 9095 | Backend dev server port |
| `deploy/docker-compose.yml` | 8080 (backend) | Backend container healthcheck |
| `tests/e2e/*/*.ts` | 8080 / 5173 / 9095 | Test API URLs |
| `playwright.config.js` | 5173 | Test base URL |
| `deploy/docker-compose.yml` | 80/443 (frontend) | Frontend container ports |
| `tests/e2e/login.spec.ts` | 5173 | Frontend test navigation |

### Existing .env Files
- `.env` - Database config (no port vars)
- `deploy/.env.example` - Docker compose env vars (no port vars)
- No `.env.example` at root for dev instructions

## Recommended Port Structure

### Development Environment (Default)
```bash
# .env (root)
FRONTEND_PORT=5173
BACKEND_PORT=9095
DATABASE_PORT=5433
HEALTH_PORT=9095    # Same as backend (health is on /health endpoint)
```

### Docker Compose Environment
```bash
# deploy/.env (already implicitly supported)
BACKEND_PORT=8080          # Container mapped port (internal container uses 8080 via ENV)
FRONTEND_PORT=80           # Container mapped port
DATABASE_PORT=5432         # Container mapped port
```

**Note:** Docker container ports are internal service ports (not mapped host ports). The mapping goes in docker-compose.yml. Variable names indicate intent not host exposure.

## Proposed Changes

### 1. Add Port Variables to .env Files

**File: `.env` (root)**
- Add: `FRONTEND_PORT=5173`
- Add: `BACKEND_PORT=9095`
- Add: `DATABASE_PORT=5433`

**File: `.env.example` (root) - NEW**
- Create with port documentation for dev environment

**File: `deploy/.env.example`**
- Add: `FRONTEND_PORT=80`
- Add: `BACKEND_PORT=8080`
- Add: `DATABASE_PORT=5432`

### 2. Update Code to Use Environment Variables

**File: `web/vite.config.js`**
- Add `dotenv` package for loading `.env`
- Configure `port` from `process.env.FRONTEND_PORT`: `port: Number(process.env.FRONTEND_PORT) || 5173`
- Update proxy target: `http://localhost:${process.env.BACKEND_PORT}`
- Load `.env` file at top of config

**File: `web/src/lib/composables/useWebSocket.ts`**
- Import `BACKEND_PORT` from Vite env: `const backendPort = $env('FRONTEND_PORT') || '5173'`
- Wait - Svelte 5 doesn't have `$env`. Need to use `import.meta.env.VITE_BACKEND_PORT`
- Update backend host logic to use `import.meta.env.VITE_BACKEND_PORT || '9095'`

**File: `run-dev.sh`**
- Remove hardcoded `export PORT=${PORT:-9095}`
- Source `.env` file if exists
- Use default values as fallback

**File: `deploy/docker-compose.yml`**
- Use variable for backend healthcheck port: `http://localhost:${BACKEND_PORT:-8080}/api/health`
- Document port variable usage

### 3. Update Tests

**File: `tests/e2e/fixtures.ts`**
- Add `API_BASE_URL` and `FRONTEND_BASE_URL` from environment
- Export: `export const API_BASE = process.env.API_BASE_URL || 'http://localhost:9095'`
- Export: `export const FRONTEND_BASE = process.env.FRONTEND_BASE_URL || 'http://localhost:5173'`

**File: `tests/e2e/inventory.spec.ts`**
- Replace hardcoded URLs with `API_BASE`

**File: `tests/e2e/pos-flow.spec.ts`**
- Replace hardcoded URLs with `API_BASE`

**File: `tests/e2e/reports.spec.ts`**
- Replace hardcoded URLs with `API_BASE`

**File: `tests/e2e/api-integration.spec.ts`**
- Replace URLs with `FRONTEND_BASE` / `API_BASE`

**File: `playwright.config.js`**
- Use `process.env.FRONTEND_BASE_URL || 'http://localhost:5173'` for `baseURL`

## Implementation Plan

### Phase 1: Foundation (.env Setup)
**Priority:** Low risk, non-breaking

1. Create `.env.example` at root with documented port vars
2. Update `.env` to include port variables (commented for clarity)
3. Update `deploy/.env.example` with Docker port vars
4. Options:
   - a) Install and configure `dotenv` for Node tooling
   - b) Manual sourcing approach - no npm dependencies needed

**Recommendation: Option b - Manual sourcing** - Most common for Go scripts, minimal dependencies

### Phase 2: Code Updates

5. Update `web/vite.config.js` - use `process.env` for ports
6. Update `web/src/lib/composables/useWebSocket.ts` - use `import.meta.env.VITE_BACKEND_PORT`
7. Update `run-dev.sh` - source `.env` and use env vars
8. Update `seeds-dev.sh` - reference port vars in connection string

### Phase 3: Test Infrastructure

9. Update `tests/e2e/fixtures.ts` - export base URLs from env
10-15. Update 6 test files to use exported base URLs
16. Update `playwright.config.js` baseURL from env

### Phase 4: Docker Updates

17. Update `deploy/docker-compose.yml` - use `${BACKEND_PORT}` in healthcheck
18. Verify docker-compose still works with new vars

## Check If Impact

### Expected Benefits
- One-command environment switching: `FRONTEND_PORT=3001 BACKEND_PORT=3002 npm run dev`
- CI/CD can easily run against different ports
- Port documentation in `.env.example` serves as onboarding reference
- No more grep-for-ports across codebase when environment changes

### Minimal Side Effects
- **Breaking changes**: None if default values align with current hardcoded values
- **New dependencies**: `dotenv` if we choose npm-based approach (RECOMMENDED: skip - use native sourcing)
- **CI adjustments**: Any CI scripts that reference ports need updating

### Risk Assessment
- **Risk Level:** Low
- **Runtime impact:** None (defaults match current values)
- **Rollback:** Remove env var references, restore hardcoded values
- **Breaking change risk:** None (backward compatible)

## Suggested Implementation Order

1. `.env.example` & `.env` - Document and set values
2. `run-dev.sh` & `seed-dev.sh` - Scripts benefit immediately
3. `web/src/lib/composables/useWebSocket.ts` - Vite env variables
4. `web/vite.config.js` - dotenv + proxy config
5. `tests/e2e/fixtures.ts` - Base URL exports
6. Test files - Reference exported URLs
7. `playwright.config.js` - baseURL from env
8. `deploy/docker-compose.yml` & `deploy/.env.example` - Document for Docker

## Verification Steps

```bash
# 1. Dev server uses .env ports
FRONTEND_PORT=5173 npm run dev  # should work

# 2. Port check
curl http://localhost:${FRONTEND_PORT}   # should work
curl http://localhost:${BACKEND_PORT}/health  # should work

# 3. E2E tests still pass
npm run test:e2e  # 23 passed

# 4. Docker (optional)
docker-compose up -d  # healthcheck should still work
```

## Files to Edit (Summary)

| File | Change |
|------|--------|
| `.env` | Add port variables |
| `.env.example` | Create with documented port variables |
| `deploy/.env.example` | Add Docker port variables |
| `web/vite.config.js` | Use env vars for port + proxy target |
| `web/src/lib/composables/useWebSocket.ts` | Use `import.meta.env.VITE_BACKEND_PORT` |
| `run-dev.sh` | Source `.env`, use `$BACKEND_PORT` |
| `seed-dev.sh` | Update DATABASE_URL port reference |
| `tests/e2e/fixtures.ts` | Export base URLs from env |
| `tests/e2e/*.spec.ts` (6 files) | Use exported base URLs |
| `playwright.config.js` | baseURL from env |
| `deploy/docker-compose.yml` | `${BACKEND_PORT}` in healthcheck |
