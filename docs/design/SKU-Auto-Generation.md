# SKU Auto-Generation with User Override

**Status:** Proposed
**Date:** 2026-09-03
**Scope:** Product creation modal, backend validation, seeder

---

## 1. Problem

The SKU field in the Add Product modal is 100% user-typed with no guidance. This leads to:

- Inconsistent formats across staff (`sku-001`, `SKU001`, `001`, random strings)
- Typos that are hard to detect until barcode scanning fails
- Cognitive overhead — staff must invent a unique code for every product

The backend already has a `GET /products/next-sku` endpoint backed by a PostgreSQL `sku_seq` sequence, but **no frontend code calls it**. This is a dead feature waiting to be wired up.

---

## 2. Decision

**Auto-generate SKU# on modal open, but allow user override.**

Format: `SKU-YYYY-NNNNNN`

| Component | Example |
|-----------|---------|
| Prefix | `SKU` (fixed) |
| Year | `2026` (Jakarta time, for readability) |
| Sequence | `000042` (6-digit zero-padded, from `sku_seq`) |

The sequence does **not** reset yearly. The year is a readability prefix only. A product created in 2027 after 5000 total products gets `SKU-2027-005001`.

---

## 3. Why Allow Override

| Reason | Example |
|--------|---------|
| Supplier SKUs | Product already has a code from the supplier; staff prefers to use it for lookup |
| Barcode-as-SKU | Retailer uses the barcode number as the SKU |
| Migration continuity | Existing codes from another system need to be preserved |
| Multi-supplier | Same product from different suppliers may need a store-internal code |

---

## 4. Current State

| Layer | SKU Handling |
|-------|-------------|
| Database | `products.sku VARCHAR(50) UNIQUE NOT NULL` — uniqueness enforced at DB level only |
| Backend validation (`validateProduct`) | **None** — only checks name, price, cost, stock |
| Backend `GetNextSKU` | Exists, generates `SKU-000001` (no year prefix) |
| Backend `sku_seq` | Exists, starts at 1, increments by 1 |
| Frontend | Does NOT call `GET /products/next-sku` — SKU is 100% user-typed |
| Frontend validation | Only checks `sku` is not empty |
| Seeder `generateSKU` | Uses `SKU-%05d` (5-digit, no year, loop-index-based) |

---

## 5. Implementation Plan

### 5.1 Backend — SKU Format Change

**File:** `internal/product/repository.go` — `GetNextSKU()`

```go
func (r *Repository) GetNextSKU(ctx context.Context) (string, error) {
    var skuNum int
    err := r.db.QueryRow(ctx, "SELECT nextval('sku_seq')").Scan(&skuNum)
    if err != nil {
        return "", fmt.Errorf("failed to get next SKU: %w", err)
    }
    year := time.Now().In(shared.JakartaLocation()).Year()
    return fmt.Sprintf("SKU-%d-%06d", year, skuNum), nil
}
```

### 5.2 Backend — SKU Validation

**File:** `internal/product/handler.go` — `validateProduct()`

Add to existing validation:

```go
if strings.TrimSpace(p.SKU) == "" {
    return fmt.Errorf("sku is required")
}
if len(p.SKU) > 50 {
    return fmt.Errorf("sku must not exceed 50 characters")
}
```

No format regex enforcement — the `SKU-YYYY-NNNNNN` format is a convenience default. Manual entries like `BRAND-XYZ-123` are valid business intent. Hard rules: required, max 50 chars, unique.

### 5.3 Backend — Uniqueness Error Handling

**File:** `internal/product/handler.go` — `CreateProduct()`

Wrap the `CreateProduct` call to catch `pgconn.PgError` with code `23505` on `products_sku_key`:

```go
if err := h.svc.CreateProduct(c.Request.Context(), &product); err != nil {
    if isUniqueViolation(err) {
        c.JSON(http.StatusConflict, gin.H{"error": "SKU already exists"})
        return
    }
    shared.InternalError(c, err)
    return
}
```

Where `isUniqueViolation` checks:

```go
func isUniqueViolation(err error) bool {
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) && pgErr.Code == "23505" {
        return strings.Contains(pgErr.ConstraintName, "sku")
    }
    return false
}
```

### 5.4 Frontend — `getNextSku` Service

**File:** `web/src/modules/product/services/product-service.ts`

Add:

```ts
export async function getNextSku(): Promise<string> {
    const r = await apiClient.get('/products/next-sku');
    return r.data.data;
}
```

### 5.5 Frontend — ProductFormModal Changes

**File:** `web/src/modules/product/components/ProductFormModal.svelte`

Changes:

1. Add `getNextSku` prop (async callback):

```ts
let {
    // ... existing props ...
    getNextSku = undefined as (() => Promise<string>) | undefined,
} = $props();
```

2. On modal open in "add" mode, auto-fill SKU:

```ts
let skuLoaded = $state(false);

$effect(() => {
    if (open && mode === 'add' && getNextSku && !skuLoaded) {
        getNextSku().then(sku => { form.sku = sku; skuLoaded = true; });
    }
    if (!open) { skuLoaded = false; }
});
```

3. Add refresh handler:

```ts
async function refreshSku() {
    if (!getNextSku) return;
    const sku = await getNextSku();
    form.sku = sku;
}
```

4. SKU field — wrap in relative div with RefreshCw button:

```svelte
<div>
    <label for="prod-sku" class="block text-sm font-medium text-text-secondary mb-2">
        {labels.sku} <span class="text-destructive">*</span>
    </label>
    <div class="relative">
        <Input
            id="prod-sku"
            bind:value={form.sku}
            type="text"
            error={fieldErrors.sku}
            required
            class="pr-10"
            maxlength={50}
        />
        {#if mode === 'add' && getNextSku}
            <button
                type="button"
                onclick={refreshSku}
                class="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-text-muted hover:text-text-primary transition-colors rounded-lg hover:bg-surface-hover"
                title={labels.refreshSku}
                aria-label={labels.refreshSku}
            >
                <RefreshCw size={14} />
            </button>
        {/if}
    </div>
    {#if mode === 'add'}
        <p class="text-[11px] text-text-muted mt-1">{labels.skuFormatHint}</p>
    {/if}
</div>
```

5. Validation — add max length check:

```ts
if (!form.sku.trim()) errors.sku = labels.errorSkuRequired;
if (form.sku.length > 50) errors.sku = labels.errorSkuTooLong;
```

### 5.6 Frontend — ProductsPage Wiring

**File:** `web/src/modules/product/components/ProductsPage.svelte`

1. Import `getNextSku`:

```ts
import { getNextSku } from '../services/product-service';
```

2. Pass to modal:

```svelte
<ProductFormModal
    bind:open={showModal}
    bind:mode={modalMode}
    bind:form
    {getNextSku}
    ...
/>
```

3. In `resetForm()`, **remove** `sku: ''` — let the modal's `$effect` handle pre-fill:

```ts
function resetForm() {
    const defaultTaxId = taxClasses.find(tc => tc.name === 'PPN 11%')?.id ?? null;
    form = {
        name: '', sku: '', /* sku will be auto-filled by modal */
        // ... rest unchanged
    };
    modalCategorySearch = '';
}
```

### 5.7 Frontend — i18n Labels

**Files:** `web/src/shared/i18n/id.ts`, `web/src/shared/i18n/en.ts`

Add:

```ts
// id.ts
errorSkuTooLong: 'SKU maksimal 50 karakter',
refreshSku: 'Generate ulang SKU',
skuFormatHint: 'Format: SKU-TAHUN-NNNNNN (contoh: SKU-2026-000042)',

// en.ts
errorSkuTooLong: 'SKU must not exceed 50 characters',
refreshSku: 'Refresh SKU',
skuFormatHint: 'Format: SKU-YEAR-NNNNNN (e.g.: SKU-2026-000042)',
```

### 5.8 Seeder — Format Update

**File:** `cmd/dummy/main.go` — `generateSKU()`

```go
func generateSKU(category string, index int) string {
    year := time.Now().In(shared.JakartaLocation).Year()
    return fmt.Sprintf("SKU-%d-%05d", year, index+1)
}
```

After seeding, sync `sku_seq` to avoid collision with production-generated SKUs. Add to `main()` after `injectProducts()`:

```go
// Sync sku_seq past the highest seeded SKU
_, err = db.Exec(`SELECT setval('sku_seq',
    GREATEST(COALESCE((SELECT MAX(CAST(substring(sku FROM 'SKU-\\d+-(\\d+)') AS bigint))
                       FROM products WHERE sku ~ '^SKU-\\d+-\\d+$'), 1), 1))`)
if err != nil {
    log.Printf("warning: failed to sync sku_seq: %v", err)
}
```

---

## 6. Mockup — SKU Field in Add Product Modal

```
┌─────────────────────────────────────────────────────────┐
│  SKU# *                                                 │
│  ┌─────────────────────────────────────┬─────┐          │
│  │ SKU-2026-000042                     │  ↻  │          │
│  └─────────────────────────────────────┴─────┘          │
│  Format: SKU-TAHUN-NNNNNN (contoh: SKU-2026-000042)    │
└─────────────────────────────────────────────────────────┘
```

- `↻` (RefreshCw) button sits inside the input as a suffix icon
- Clicking it calls `GET /products/next-sku` and replaces the current value
- On initial modal open (add mode), field auto-fills with next SKU
- Helper text appears below as muted hint
- In **edit** mode: RefreshCw button is hidden; SKU is already assigned

---

## 7. Validation Rules Summary

| Rule | Frontend | Backend | Rationale |
|------|----------|---------|-----------|
| Required (non-empty) | `!form.sku.trim()` | `strings.TrimSpace(p.SKU) == ""` | Defense-in-depth |
| Max 50 chars | `maxlength={50}` + validate | `len(p.SKU) > 50` | Matches DB constraint |
| Unique | Toast from `409` response | `UNIQUE(sku)` + friendly error | DB catches, handler returns clean message |
| Trim whitespace | Implicit (user types) | Not trimmed — sent as-is | SKU spaces are intentional if user wants them |
| Format pattern | Not enforced | Not enforced | User override is a feature, not a bug |

---

## 8. Edge Cases

| Scenario | Behavior |
|----------|----------|
| User leaves SKU empty and submits | Frontend validation blocks; error "SKU is required" |
| User enters duplicate SKU | Backend returns `409 Conflict`; toast shows "SKU already exists" |
| User enters `SKU-2026-999999` manually | Accepted — sequence stays at its current position; next auto-gen picks up where it left off |
| User clicks RefreshCw multiple times | Each click consumes a sequence number; skipped numbers are normal (like invoice gaps) |
| User edits product SKU to a duplicate | `409 Conflict` on update; same error handling |
| Re-seed in different year | All products get new year prefix; no collision with old data |
| Re-seed in same year | Index-based generation may collide; `ON CONFLICT (sku) DO NOTHING` silently skips |

---

## 9. Migration Considerations

- **No schema change needed** — `sku_seq` and `products.sku UNIQUE` already exist
- **No data migration** — existing SKUs remain untouched
- **Sequence sync** — after deploying, if the seeder was used, `sku_seq` may need manual sync:
  ```sql
  SELECT setval('sku_seq',
      GREATEST(COALESCE((SELECT MAX(CAST(substring(sku FROM 'SKU-\d+-(\d+)') AS bigint))
                         FROM products WHERE sku ~ '^SKU-\d+-\d+$'), 1), 1));
  ```
- **Backward compatibility** — old-format SKUs (`SKU-000001`) still work; they just won't match the new format for new products

---

## 10. Testing Checklist

- [ ] `GET /products/next-sku` returns `SKU-2026-NNNNNN` format
- [ ] Add Product modal pre-fills SKU on open
- [ ] RefreshCw button fetches next SKU
- [ ] RefreshCw button hidden in edit mode
- [ ] Submitting empty SKU shows validation error
- [ ] Submitting duplicate SKU shows "SKU already exists" toast
- [ ] SKU > 50 chars is rejected
- [ ] Seeder generates `SKU-YYYY-NNNNNN` format
- [ ] Seeder syncs `sku_seq` after injection
- [ ] Existing products with old-format SKUs still display correctly
