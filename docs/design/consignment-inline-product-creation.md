# Consignment Inline Product Creation

> **Status:** Proposed — Superseded by [consignment-batch-add-terms.md](./consignment-batch-add-terms.md) for batch flow

## Problem

When a consignment supplier arrives with **entirely new products** (not yet in the product catalog), the user cannot add them as terms because the product dropdown only shows existing active products. The current flow requires:

1. Navigate to Products page → Create new product
2. Return to Consignment → Create supplier → Create arrangement
3. Add term (selecting the newly created product)

This disrupts the workflow and is unintuitive for first-time consignment setup.

## Solution

Add a **"+ Create new product"** option at the bottom of the product dropdown in the Add Term modal. Selecting it opens a **secondary modal** with a minimal product creation form. After creation, the product is automatically selected in the dropdown, and the user can proceed with adding the term.

### Business Flow (After Implementation)

```
Supplier arrives with new products
  → Create consignment supplier
  → Create arrangement
  → Add term
  → Product dropdown → Click "+ Create new product"
  → Fill SKU + Name + Price → Save
  → Product auto-selected → Fill store share → Save term
  → Create receipt (receiving goods)
```

## UI Mockup

### 1. Add Term Modal — Product Dropdown with "Create New" Option

```
┌─────────────────────────────────────────────────────────────┐
│  Tambah Term                                         [X]   │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Produk *                                                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Cari produk...                              🔍     │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │  Widget A (SKU-001)                                │   │
│  │  Widget B (SKU-002)                                │   │
│  │  Gadget X (SKU-003)                                │   │
│  │  ─────────────────────────────────────────────────  │   │
│  │  ➕ Buat produk baru                                │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  Harga (Rp) *                                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  0                                                   │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  Tipe Hak Toko *                                           │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Persentase                                          │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  Nilai Hak Toko (Persentase) *                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  20                                                  │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ℹ️  Terms berlaku untuk stok yang belum terjual; tidak     │
│     mengubah penjualan yang sudah tercatat.                 │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│                                              [Batal] [Simpan]│
└─────────────────────────────────────────────────────────────┘
```

### 2. Create New Product Modal (Secondary)

```
┌─────────────────────────────────────────────────────────────┐
│  Buat Produk Baru                                    [X]   │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  SKU *                                                     │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  SKU-2026-00045  (auto-generated)                   │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  Nama Produk *                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Contoh: Kaos Polos Putih                           │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  Harga Jual (Rp) *                                         │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  0                                                   │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  Harga Modal (Rp)                                          │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  0                                                   │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│                                           [Batal] [Buat]    │
└─────────────────────────────────────────────────────────────┘
```

### 3. After Product Creation — Auto-Selected

```
┌─────────────────────────────────────────────────────────────┐
│  Tambah Term                                         [X]   │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Produk *                                                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Kaos Polos Putih (SKU-2026-00045)           ✓     │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  Harga (Rp) *                                              │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  0                                                   │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ... (rest of form)                                         │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Files to Change

### 1. `web/src/modules/consignment/components/TermsEditor.svelte`

**Change A — Add state for inline product creation (after line 48)**

```typescript
let showCreateProductModal = $state(false);
```

**Change B — Add "create new" option to product dropdown (line 70-80)**

```typescript
async function loadProducts() {
  try {
    const opts = await listAddTermProductOptions(arrangement.id);
    productOptions = [
      ...opts.map((p) => ({
        value: p.id,
        label: p.sku ? `${p.name} (${p.sku})` : p.name,
      })),
      { value: -1, label: labels.consignmentCreateNewProduct },
    ];
  } catch {
    productOptions = [{ value: -1, label: labels.consignmentCreateNewProduct }];
  }
}
```

**Change C — Handle selection of "create new" option (after line 82)**

```typescript
function handleProductChange(value: number) {
  if (value === -1) {
    newRow.product_id = undefined;
    showCreateProductModal = true;
  }
}
```

**Change D — Handle product creation callback**

```typescript
function onProductCreated(productId: number) {
  showCreateProductModal = false;
  newRow.product_id = productId;
  loadProducts(); // Refresh list to include new product
}
```

**Change E — Update SelectSearch in template (line 236-242)**

```svelte
<SelectSearch
  bind:value={newRow.product_id}
  options={productOptions}
  placeholder={labels.consignmentSelectProduct}
  searchPlaceholder={labels.consignmentSearchProduct}
  notFoundText={labels.consignmentProductNotFound}
  onchange={handleProductChange}
/>
```

**Change F — Add import for AddProductInline (line 1)**

```typescript
import AddProductInline from './AddProductInline.svelte';
```

**Change G — Add modal in template (after Modal close tag, line 299)**

```svelte
{#if showCreateProductModal}
  <AddProductInline
    bind:open={showCreateProductModal}
    oncreated={onProductCreated}
  />
{/if}
```

### 2. `web/src/modules/consignment/components/AddProductInline.svelte` (New)

```svelte
<script lang="ts">
  import { Button, Input, Modal, NumberInput } from '$shared/ui';
  import { toast } from '$shared/stores/toast.svelte';
  import { getApiErrorMessage } from '$shared/utils/error-utils';
  import { labels } from '$shared/i18n';
  import { createProduct, getNextSku } from '$modules/product/services/product-service';

  let {
    open = $bindable(false),
    oncreated,
  }: {
    open: boolean;
    oncreated?: (productId: number) => void;
  } = $props();

  let saving = $state(false);
  let skuLoading = $state(false);
  let form = $state({
    sku: '',
    name: '',
    price: 0,
    cost: 0,
  });

  async function loadSku() {
    skuLoading = true;
    try {
      form.sku = await getNextSku();
    } catch {
      // User can enter manually
    } finally {
      skuLoading = false;
    }
  }

  async function submit() {
    if (!form.sku.trim()) {
      toast.error(labels.errorSkuRequired);
      return;
    }
    if (!form.name.trim()) {
      toast.error(labels.errorNameRequired);
      return;
    }
    if (form.price < 0) {
      toast.error(labels.errorPricePositive);
      return;
    }

    saving = true;
    try {
      const productId = await createProduct({
        sku: form.sku.trim(),
        name: form.name.trim(),
        price: form.price,
        cost: form.cost,
        stock: 0,
        status: 'active',
      });
      toast.success(labels.consignmentProductCreated);
      oncreated?.(productId);
    } catch (e: unknown) {
      toast.error(getApiErrorMessage(e, labels.consignmentProductCreateError));
    } finally {
      saving = false;
    }
  }

  $effect(() => {
    if (open) {
      loadSku();
    }
  });
</script>

<Modal bind:open title={labels.consignmentNewProduct} size="sm">
  <div class="space-y-4">
    <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
      <span>SKU <span class="text-danger">*</span></span>
      <Input
        bind:value={form.sku}
        placeholder={labels.consignmentProductSkuPlaceholder}
        disabled={skuLoading}
        class="h-9 text-sm"
      />
    </label>
    <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
      <span>{labels.consignmentProductName} <span class="text-danger">*</span></span>
      <Input
        bind:value={form.name}
        placeholder={labels.consignmentProductNamePlaceholder}
        class="h-9 text-sm"
      />
    </label>
    <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
      <span>{labels.consignmentProductPrice} (Rp) <span class="text-danger">*</span></span>
      <NumberInput min="0" bind:value={form.price} class="h-9 text-sm" />
    </label>
    <label class="flex flex-col gap-1.5 text-sm font-medium text-text-secondary">
      <span>{labels.consignmentProductCost} (Rp)</span>
      <NumberInput min="0" bind:value={form.cost} class="h-9 text-sm" />
    </label>
  </div>
  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button variant="secondary" onclick={() => (open = false)}>
        {labels.cancel}
      </Button>
      <Button onclick={submit} disabled={saving}>
        {saving ? labels.saving : labels.create}
      </Button>
    </div>
  {/snippet}
</Modal>
```

### 3. `web/src/shared/i18n/en.ts`

Add after `consignmentTermsNote` (line 1898):

```typescript
consignmentCreateNewProduct: '+ Create new product',
consignmentNewProduct: 'New Product',
consignmentProductCreated: 'Product created successfully',
consignmentProductCreateError: 'Failed to create product',
consignmentProductSkuPlaceholder: 'Auto-generated SKU',
consignmentProductName: 'Product Name',
consignmentProductNamePlaceholder: 'e.g., Kaos Polos Putih',
consignmentProductPrice: 'Selling Price',
consignmentProductCost: 'Cost Price',
```

### 4. `web/src/shared/i18n/id.ts`

Add after corresponding label:

```typescript
consignmentCreateNewProduct: '+ Buat produk baru',
consignmentNewProduct: 'Produk Baru',
consignmentProductCreated: 'Produk berhasil dibuat',
consignmentProductCreateError: 'Gagal membuat produk',
consignmentProductSkuPlaceholder: 'SKU auto-generated',
consignmentProductName: 'Nama Produk',
consignmentProductNamePlaceholder: 'contoh: Kaos Polos Putih',
consignmentProductPrice: 'Harga Jual',
consignmentProductCost: 'Harga Modal',
```

## Backend Changes

### Backend Changes

The repository already returns the created product ID via `RETURNING id` (repository.go:340). The handler needs to include it in the response.

**File:** `internal/product/handler.go` (line 261-268)

```go
// Before
if err := h.svc.CreateProduct(c.Request.Context(), &product); err != nil {
  if isDuplicateKeyError(err) {
    c.JSON(http.StatusConflict, gin.H{"error": "SKU already exists"})
    return
  }
  shared.InternalError(c, err)
  return
}
// ... audit log ...
c.JSON(http.StatusCreated, gin.H{"message": "product created"})

// After
if err := h.svc.CreateProduct(c.Request.Context(), &product); err != nil {
  if isDuplicateKeyError(err) {
    c.JSON(http.StatusConflict, gin.H{"error": "SKU already exists"})
    return
  }
  shared.InternalError(c, err)
  return
}
// ... audit log ...
c.JSON(http.StatusCreated, gin.H{"data": gin.H{"id": product.ID}})
```

**File:** `web/src/modules/product/services/product-service.ts` (line 96-100)

```typescript
// Before
export async function createProduct(
  data: ProductFormData & { category_name?: string },
): Promise<void> {
  await apiClient.post("/products", data);
}

// After
export async function createProduct(
  data: ProductFormData & { category_name?: string },
): Promise<number> {
  const res = await apiClient.post("/products", data);
  return res.data.data.id;
}
```

## Edge Cases

1. **SKU Duplicate** — Backend returns 409 Conflict; show error toast "SKU already exists"
2. **Network Error** — Show generic error toast; modal stays open for retry
3. **Cancel Creation** — Close modal; product dropdown shows "Select product" (no product selected)
4. **Empty Product List** — Dropdown shows only "+ Create new product" option
5. **Ownership Type** — Product is created with default `ownership_type = 'store'`. When the first receipt is created for this product, the system updates it to `ownership_type = 'consignment'` (existing logic in receipt creation handles this automatically)

## Testing

1. **New supplier flow** — Create supplier → arrangement → add term → create new product → verify product appears in dropdown
2. **SKU auto-generation** — Verify SKU field is pre-filled with next available SKU
3. **Validation** — Test empty SKU, empty name, negative price
4. **Duplicate SKU** — Test creating product with existing SKU
5. **Product list refresh** — After creation, verify new product appears in dropdown without page refresh

## Summary

| Aspect | Detail |
|--------|--------|
| **Problem** | Cannot add new products from consignment supplier during term creation |
| **Solution** | Inline product creation via "+ Create new product" option in dropdown |
| **UI Pattern** | Secondary modal (nested modals) for minimal product creation |
| **Backend Change** | Return created product ID in `POST /products` response |
| **Files Modified** | TermsEditor.svelte, AddProductInline.svelte (new), product-service.ts, handler.go, i18n files |
| **Business Rule** | Ownership type set to 'consignment' when first receipt is created (existing logic) |
| **Risk** | Low — additive change, no existing behavior modified |
