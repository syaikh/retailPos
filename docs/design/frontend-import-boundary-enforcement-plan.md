# Plan: Frontend Import-Boundary Enforcement (Revisit of ADR On Hold)

| Field | Value |
|-------|-------|
| Status | **Implemented** (2026-10-21) — enforcement at `warn` severity; shared→auth debt resolved same day (§7) |
| Date | 2026-10-21 |
| Supersedes | `docs/adr/ADR_Frontend_Module_Import_Boundaries.md` (On Hold, 2026-08-20) |
| Scope | `web/src/modules/` cross-module imports, `web/eslint.config.js` |
| Related | `docs/adr/ADR_Modular_Monolith_Module_Boundaries.md` (backend equivalent) |

## 1. Why revisit

The 2026-08-20 ADR put import-boundary enforcement on hold with three
assumptions. All three have gone stale:

1. **"There is no ESLint in the web project"** — no longer true. Flat config
   exists at `web/eslint.config.js` (typescript-eslint + svelte), and CI runs
   `npm run lint` in the `frontend-lint` job (`.github/workflows/ci.yml:264`).
   The setup cost that justified "no enforcement" is already paid.
2. **"~20+ violations — manageable without tooling"** — the count has grown to
   **~27 cross-module deep imports**. The ADR's own revisit trigger
   ("violations accumulate faster than they are caught") is approaching.
3. **"Fix violations incrementally"** — no longer purely free: at least four
   test files assert the deep-import strings literally exist in source
   (e.g. `StockOpnamesPage.svelte.test.ts`, `ProductsPage.svelte.test.ts`,
   `PricingRulesPage.svelte.test.ts`, `RackStockPanel.svelte.test.ts`), so any
   cleanup must edit those tests too.

Additionally, one violation class the ADR does not cover exists:
`shared/stores/settings.svelte.ts` imports `$modules/admin/services/...` —
a **dependency inversion** (shared code depending on a module). This is the
only violation with real architectural risk (import-cycle potential).

## 2. Current violations (audited 2026-10-21)

~27 cross-module deep imports, by importer:

| Importer module | Deep target(s) |
|-----------------|----------------|
| `stock-opname` | `product/services`, `supplier/services`, `storage-location/services` |
| `purchase-orders` | `supplier/services` ×2, `stores/services` |
| `product` | `pricing/services`, `pricing/types`, `inventory/components` |
| `consignment` | `stores/services`, `product/services` ×2 |
| `pos` | `sales/types` ×2, `customers/types` |
| `pricing` | `product/services` |
| `storage-location` | `product/services` |
| `inventory` | `storage-location/services`, `storage-location/types` |
| `shifts` | `stores/services`, `stores/types` |
| `customer-groups` | `admin/types` |
| `user` | `auth/components/PasswordChangeForm.svelte` |
| `dashboard` | `consignment/components/PendingSettlementsModal.svelte` |
| **`shared`** | **`admin/services` (inversion — fix first)** |

`app/main.svelte` also deep-imports auth/consignment/dashboard components, but
the app shell is the composition root and is **exempt** by design.

## 3. Plan

### Step 1 — Fix inversion + one focused cleanup PR

- Fix `shared/stores/settings.svelte.ts` → `admin` inversion first.
- Add missing barrel re-exports per ADR §4 (`product`, `supplier`,
  `storage-location`, `stores`, `customers`, `sales` — note
  `web/src/modules/product/index.ts` is only partially done today).
- Convert the ~27 deep imports to barrel paths (`$modules/<name>`).
- Update the four source-asserting tests to match the new import paths.

### Step 2 — Enforce with core ESLint only (no new plugin)

The ADR weighed `eslint-plugin-boundaries` (heavy, per-module config,
Svelte 5 rune false-positive risk). Not needed: ESLint's built-in
`no-restricted-imports` with `patterns` + `except` suffices, generated
programmatically in `eslint.config.js`:

```js
// eslint.config.js (sketch) — auto-adapts when a new module is added
import { readdirSync } from 'node:fs';

const modules = readdirSync('src/modules'); // module names
// For each module X, in files under src/modules/X/**:
//   forbid '$modules/*/**' except '$modules/X/**'
// Files under src/app/** are exempt (composition root).
```

Rationale:

- Zero new dependencies; directly addresses the ADR's cost table.
- Start the rule as `'warn'` — CI runs plain `eslint .` (no `--max-warnings`),
  so warnings do not break builds.
- Flip to `'error'` only after Step 1 lands.
- Exempt `app/`; same-module deep imports stay allowed (ADR scope is
  cross-module only).

### Step 3 — Update the ADR

Change status from **On Hold** to **Accepted (lightweight enforcement)** with
the new rationale, and cross-link this plan.

## 4. Verification

- `cd web && npx eslint src` reports zero warnings on the new rule after Step 1.
- `npm run lint` in CI stays green (`warn` phase does not fail the job).
- The four source-asserting tests pass with updated expectations.
- No runtime behavior changes — import paths only.

## 5. Rollback

Delete the generated `no-restricted-imports` block from
`web/eslint.config.js`; barrel re-exports added in Step 1 are harmless and
can stay.

## 6. Implementation notes (2026-10-21)

- **ESLint 9.39 note:** `no-restricted-imports` `patterns` objects no longer
  support `except`. Implemented with the `regex` pattern form instead, embedding
  the module name per generated config block:
  `^\\$modules/(?!<name>(\\/|$))[^/]+\\/` — matches any deep cross-module path
  while allowing bare barrels (`$modules/<other>`) and own-module internals.
- **Inversion fix:** `git mv` of
  `modules/admin/services/app-settings-service.ts` →
  `shared/services/app-settings-service.ts` (it only ever depended on
  `$shared/api/http-client`). Importers updated: `shared/stores/settings.svelte.ts`,
  its test mock, and `admin/components/SettingsPage.svelte`.
- **Barrel re-exports added:** `product` (getProductsByIds, getNextSku,
  getProductOptions, getWarehouses, ProductOption), `supplier` (getSuppliers,
  getProductsBySupplier, ProductSupplier), `pricing` (getPricingRules,
  PricingRule), `inventory` (StockAdjustModal, RackStockPanel),
  `consignment` (PendingSettlementsModal), `auth` (PasswordChangeForm).
- **~27 deep imports converted** to barrel paths; component imports from
  barrels use named syntax (`import { X } from "$modules/<name>"`).
- **Tests updated** (source-asserting strings): stock-opname, inventory,
  pricing, dashboard, product (ProductsPage `import { StockAdjustModal }`).

### Verification result

`npx eslint src` → **0 errors**, 5 warnings:
1 pre-existing `no-unused-vars` in `CustomerGroupsPage.svelte`, plus **4
warnings (3 files) of newly-surfaced shared→module debt**: `shared/api/http-client.ts`,
`shared/composables/useRBAC.svelte.ts`, and its test import `$modules/auth`
(http-client ↔ auth is a genuine cycle). Left as warn-phase debt — fixing
requires relocating the auth session/store into `$shared`; tracked as follow-up.
Targeted vitest for all touched test files: 143/143 pass.

### Review fixes (same day)

- `PasswordChangeForm.svelte` now imports `changePassword` from
  `../services/auth-service` (relative) instead of the `$modules/auth`
  barrel, removing the cycle the new barrel export would have created
  (`auth/index.ts` → component → `auth/index.ts`).
- Dropped the unused `ProductOption` type re-export from `product/index.ts`
  (no consumer).
- Added justified `eslint-disable-next-line no-restricted-imports` comments
  to the 4 known shared→auth debt lines (http-client, useRBAC + its test),
  referencing this ADR — the lint output now shows only genuinely new
  violations (plus the pre-existing unrelated unused-var warning).
  *(These comments were removed the same day — §7 resolved the debt.)*
- Known follow-up (not fixed, low severity): Svelte components exported from
  barrels (`consignment`, `auth`, `inventory`) may defeat bundler
  tree-shaking and pull extra code into consumer chunks — verify chunk sizes
  on the next CI build; if retained, annotate with `/* @__PURE__ */` or
  accept.

## 7. Session-kernel relocation — shared→auth debt resolved (2026-10-21)

The 4 warn-phase debt lines shared→`$modules/auth` (http-client, useRBAC,
useRBAC test) were fixed the same day by relocating the session/refresh
machinery into `$shared`, inverting the dependency instead of suppressing it:

- `modules/auth/lib/session.ts` → `shared/api/session.ts` (pure token
  helpers); `modules/auth/stores/auth-store.svelte.ts` →
  `shared/stores/auth.svelte.ts`; `User` type → `shared/types/user.ts`.
- New `shared/auth/refresh-kernel.ts` owns `authApi`, the refresh
  lock/queue (`doRefresh`/`refreshAccessToken`), proactive refresh, the
  428 password-rotation gate, axios interceptors, `logout` and
  `getAuthHeaders`. `modules/auth/services/auth-service.ts` keeps the flows
  (login/restore/checkAuth/changePassword/updatePreferences/
  handleCrossTabLogout) and re-exports the kernel API unchanged, so the
  `$modules/auth` public surface is stable.
- Old module paths remain as **pure re-export shims**; a runtime identity
  test (`modules/auth/services/__tests__/kernel-identity.test.ts`) asserts
  that importing via `$modules/auth` or `$shared` resolves to the same
  module instances — guarding the refresh-dedup and cross-tab-leader
  singletons against a future re-split.
- All 4 `eslint-disable` debt comments removed; `npx eslint src` → **0
  errors, 0 warnings**.
- Follow-up (PR2, optional): delete the shims, repoint the ~33
  `useAuthStore`/`getAuthToken` import sites (36 files, incl. tests) at
  `$shared`, and flip the boundary rule to `error`.
