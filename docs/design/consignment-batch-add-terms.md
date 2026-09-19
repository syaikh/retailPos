# Consignment Batch Add Terms

> **Status:** Proposed

## Problem

Currently, adding terms is a **one-at-a-time** process:
1. Click "Add Term" → Select ONE product → Set price/share → Save
2. Repeat for each product

When a consignment supplier brings **many new products** (e.g., 10-50 SKUs), this becomes tedious and time-consuming.

## Solution

Redesign the "Add Term" modal to support **batch product selection**:
1. Select **multiple products** from the dropdown (with checkboxes)
2. Set **shared pricing** (price, share type, share value) applied to all selected products
3. Optionally **customize per-product** pricing after initial creation
4. Save all terms at once

### Business Flow (After Implementation)

```
Supplier arrives with 20 new products
  → Create consignment supplier
  → Create arrangement
  → Add term
  → Select 20 products from dropdown (multi-select)
  → Set shared price: Rp 50,000
  → Set shared store share: 20%
  → Preview shows all 20 products with pricing
  → Save → 20 terms created at once
  → (Optional) Edit individual terms if pricing differs
```

## UI Mockup

### 1. Add Term Modal — Multi-Select Product Dropdown

```
┌─────────────────────────────────────────────────────────────────┐
│  Tambah Term (Batch)                                     [X]   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Produk *                                                      │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  🔍 Cari produk...                                      │   │
│  ├─────────────────────────────────────────────────────────┤   │
│  │  ☑  Widget A (SKU-001)                                  │   │
│  │  ☑  Widget B (SKU-002)                                  │   │
│  │  ☐  Gadget X (SKU-003)                                  │   │
│  │  ☑  Gadget Y (SKU-004)                                  │   │
│  │  ☐  Gadget Z (SKU-005)                                  │   │
│  │  ─────────────────────────────────────────────────────── │   │
│  │  ➕ Buat 1 produk baru                                   │   │
│  │  📋 Buat beberapa produk baru                            │   │
│  ├─────────────────────────────────────────────────────────┤   │
│  │  ✓ 3 produk dipilih                                      │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  Harga (Rp) *                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  50000                                                   │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  Tipe Hak Toko *                                               │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Persentase                                              │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  Nilai Hak Toko (Persentase) *                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  20                                                      │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ─────────────────────────────────────────────────────────────  │
│  Produk dipilih (3):                                            │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Widget A (SKU-001)      Rp 50.000    20%              │   │
│  │  Widget B (SKU-002)      Rp 50.000    20%              │   │
│  │  Gadget Y (SKU-004)      Rp 50.000    20%              │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                              [Batal] [Simpan]   │
└─────────────────────────────────────────────────────────────────┘
```

### 2. Create Multiple Products Modal (New)

```
┌─────────────────────────────────────────────────────────────────┐
│  Buat Beberapa Produk Baru                               [X]   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Tambah baris produk baru untuk dibuat sekaligus.               │
│  SKU akan di-generate otomatis.                                 │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  SKU            │ Nama Produk         │ Harga Jual (Rp) │   │
│  ├─────────────────┼─────────────────────┼─────────────────┤   │
│  │  SKU-2026-0045  │ Kaos Polos Putih    │ 50000           │   │
│  │  SKU-2026-0046  │ Kaos Polos Hitam    │ 50000           │   │
│  │  SKU-2026-0047  │ Kaos Polos Biru     │ 50000           │   │
│  │  [+ Tambah Baris]                                    │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ℹ️  Harga modal (cost) dapat diisi nanti di halaman produk.    │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                        [Batal] [Buat 3 Produk]  │
└─────────────────────────────────────────────────────────────────┘
```

### 3. Create Multiple Products — Empty State

```
┌─────────────────────────────────────────────────────────────────┐
│  Buat Beberapa Produk Baru                               [X]   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Tambah baris produk baru untuk dibuat sekaligus.               │
│  SKU akan di-generate otomatis.                                 │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  SKU            │ Nama Produk         │ Harga Jual (Rp) │   │
│  ├─────────────────┼─────────────────────┼─────────────────┤   │
│  │  [Kosong]       │ [Ketik nama...]     │ [0]             │   │
│  │  [+ Tambah Baris]                                    │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                      [Batal] [Buat 0 Produk]    │
└─────────────────────────────────────────────────────────────────┘
```

### 4. After Selection — Products Auto-Added to Preview

```
┌─────────────────────────────────────────────────────────────────┐
│  Tambah Term (Batch)                                     [X]   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Produk *                                                      │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  4 produk dipilih                           🔍 ▼        │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ... (pricing fields)                                           │
│                                                                 │
│  ─────────────────────────────────────────────────────────────  │
│  Produk dipilih (4):                                            │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Widget A (SKU-001)      Rp 50.000    20%         [×]   │   │
│  │  Widget B (SKU-002)      Rp 50.000    20%         [×]   │   │
│  │  Gadget Y (SKU-004)      Rp 50.000    20%         [×]   │   │
│  │  Kaos Polos (SKU-045)    Rp 50.000    20%         [×]   │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                              [Batal] [Simpan]   │
└─────────────────────────────────────────────────────────────────┘
```

## Files to Change

### 1. `web/src/modules/consignment/components/TermsEditor.svelte`

**Major refactor — replace single-select with multi-select batch flow**

```typescript
// State
let selectedProductIds = $state<number[]>([]);
let showAddModal = $state(false);
let newRow = $state<TermRow>({
  price: 0,
  store_share_type: SHARE_TYPE_PERCENTAGE,
  store_share_value: 20,
});

// Handle multi-select
function toggleProductSelection(productId: number) {
  if (selectedProductIds.includes(productId)) {
    selectedProductIds = selectedProductIds.filter(id => id !== productId);
  } else {
    selectedProductIds = [...selectedProductIds, productId];
  }
}

// Handle "create new" callback
function onProductCreated(productId: number) {
  showCreateProductModal = false;
  selectedProductIds = [...selectedProductIds, productId];
  loadProducts(); // Refresh list
}

// Submit batch terms
async function submitBatch() {
  if (selectedProductIds.length === 0) {
    toast.error(labels.consignmentSelectAtLeastOneProduct);
    return;
  }
  // Validation...
  
  // Merge existing terms + new batch terms
  const payload: SetTermsPayload[] = terms.map((t) => ({
    product_id: t.product_id,
    price: t.price,
    store_share_type: t.store_share_type,
    store_share_value: t.store_share_value,
  }));
  
  // Add new batch terms
  for (const productId of selectedProductIds) {
    payload.push({
      product_id: productId,
      price: newRow.price,
      store_share_type: newRow.store_share_type,
      store_share_value: newRow.store_share_value,
    });
  }
  
  const saved = await setTerms(arrangement.id, payload);
  terms = saved;
  toast.success(labels.consignmentTermsSaved);
  showAddModal = false;
  selectedProductIds = [];
  onsaved?.();
  await loadProducts();
}
```

### 2. `web/src/modules/consignment/components/AddProductInline.svelte`

No changes needed — same as before (creates single product, returns ID).

### 3. `web/src/modules/consignment/components/AddBatchProductsInline.svelte` (New)

**Batch product creation modal — spreadsheet-like input**

```svelte
<script lang="ts">
  import { Button, Input, Modal, NumberInput } from '$shared/ui';
  import { Plus, Trash2 } from 'lucide-svelte';
  import { toast } from '$shared/stores/toast.svelte';
  import { getApiErrorMessage } from '$shared/utils/error-utils';
  import { labels } from '$shared/i18n';
  import { createProduct, getNextSku } from '$modules/product/services/product-service';

  interface ProductRow {
    sku: string;
    name: string;
    price: number;
  }

  let {
    open = $bindable(false),
    oncreated,
  }: {
    open: boolean;
    oncreated?: (productIds: number[]) => void;
  } = $props();

  let saving = $state(false);
  let rows = $state<ProductRow[]>([{ sku: '', name: '', price: 0 }]);

  async function loadSku() {
    try {
      const nextSku = await getNextSku();
      if (rows.length === 1 && !rows[0].sku) {
        rows[0].sku = nextSku;
      }
    } catch {
      // User can enter manually
    }
  }

  function addRow() {
    rows = [...rows, { sku: '', name: '', price: 0 }];
  }

  function removeRow(index: number) {
    if (rows.length > 1) {
      rows = rows.filter((_, i) => i !== index);
    }
  }

  function updateRow(index: number, field: keyof ProductRow, value: string | number) {
    rows = rows.map((row, i) => (i === index ? { ...row, [field]: value } : row));
  }

  async function submit() {
    // Validate
    const validRows = rows.filter(r => r.name.trim());
    if (validRows.length === 0) {
      toast.error(labels.consignmentAtLeastOneProduct);
      return;
    }

    for (const row of validRows) {
      if (!row.sku.trim()) {
        toast.error(labels.consignmentSkuRequired);
        return;
      }
      if (row.price < 0) {
        toast.error(labels.consignmentPriceNotNegative);
        return;
      }
    }

    saving = true;
    const createdIds: number[] = [];

    try {
      for (const row of validRows) {
        const id = await createProduct({
          sku: row.sku.trim(),
          name: row.name.trim(),
          price: row.price,
          cost: 0,
          stock: 0,
          status: 'active',
        });
        createdIds.push(id);
      }
      toast.success(labels.consignmentProductsCreated(createdIds.length));
      oncreated?.(createdIds);
    } catch (e: unknown) {
      toast.error(getApiErrorMessage(e, labels.consignmentProductCreateError));
    } finally {
      saving = false;
    }
  }

  $effect(() => {
    if (open) {
      rows = [{ sku: '', name: '', price: 0 }];
      loadSku();
    }
  });
</script>

<Modal bind:open title={labels.consignmentNewProducts} size="lg">
  <div class="space-y-4">
    <p class="text-sm text-text-secondary">
      {labels.consignmentBatchProductHint}
    </p>
    
    <div class="border border-border rounded-lg overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-muted/50">
          <tr class="text-left text-xs uppercase tracking-wider text-text-secondary">
            <th class="p-3 w-36">SKU</th>
            <th class="p-3">{labels.consignmentProductName}</th>
            <th class="p-3 w-36">{labels.consignmentProductPrice} (Rp)</th>
            <th class="p-3 w-12"></th>
          </tr>
        </thead>
        <tbody>
          {#each rows as row, i (i)}
            <tr class="border-t border-border">
              <td class="p-2">
                <Input
                  value={row.sku}
                  oninput={(e) => updateRow(i, 'sku', e.currentTarget.value)}
                  placeholder="SKU"
                  class="h-8 text-sm"
                />
              </td>
              <td class="p-2">
                <Input
                  value={row.name}
                  oninput={(e) => updateRow(i, 'name', e.currentTarget.value)}
                  placeholder={labels.consignmentProductNamePlaceholder}
                  class="h-8 text-sm"
                />
              </td>
              <td class="p-2">
                <NumberInput
                  min="0"
                  value={row.price}
                  oninput={(e) => updateRow(i, 'price', Number(e.currentTarget.value))}
                  class="h-8 text-sm"
                />
              </td>
              <td class="p-2">
                <button
                  type="button"
                  onclick={() => removeRow(i)}
                  disabled={rows.length === 1}
                  class="p-1 text-text-muted hover:text-danger disabled:opacity-30"
                >
                  <Trash2 size={14} />
                </button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    
    <button
      type="button"
      onclick={addRow}
      class="flex items-center gap-1.5 text-sm text-primary hover:text-primary/80"
    >
      <Plus size={14} />
      {labels.consignmentAddRow}
    </button>
    
    <p class="text-xs text-text-muted">
      {labels.consignmentCostHint}
    </p>
  </div>
  {#snippet footer()}
    <div class="flex justify-end gap-3 w-full">
      <Button variant="secondary" onclick={() => (open = false)}>
        {labels.cancel}
      </Button>
      <Button onclick={submit} disabled={saving || rows.filter(r => r.name.trim()).length === 0}>
        {saving ? labels.saving : labels.consignmentCreateNProducts(rows.filter(r => r.name.trim()).length)}
      </Button>
    </div>
  {/snippet}
</Modal>
```

### 4. `web/src/shared/i18n/en.ts`

Add/modify labels:

```typescript
// Batch term labels
consignmentAddTerm: 'Add Term (Batch)',
consignmentSelectProduct: 'Select products',
consignmentSearchProduct: 'Search products',
consignmentProductSelected: '{count} products selected',
consignmentSelectAtLeastOneProduct: 'Select at least one product',
consignmentProductsPreview: 'Selected products ({count})',
consignmentRemoveProduct: 'Remove',

// Single product creation
consignmentCreateNewProduct: '+ Create 1 new product',
consignmentNewProduct: 'New Product',

// Batch product creation
consignmentCreateMultipleProducts: '+ Create multiple products',
consignmentNewProducts: 'Create Multiple Products',
consignmentBatchProductHint: 'Add rows for new products to create at once. SKU will be auto-generated.',
consignmentAddRow: 'Add row',
consignmentCostHint: 'Cost price can be filled later in the product page.',
consignmentProductNamePlaceholder: 'Type product name...',
consignmentAtLeastOneProduct: 'Enter at least one product',
consignmentSkuRequired: 'SKU is required',
consignmentPriceNotNegative: 'Price must not be negative',
consignmentProductsCreated: (count: number) => `${count} products created`,
consignmentCreateNProducts: (count: number) => count > 0 ? `Create ${count} product${count > 1 ? 's' : ''}` : 'Create',
consignmentProductCreateError: 'Failed to create product',
consignmentProductCreated: 'Product created successfully',
```

### 5. `web/src/shared/i18n/id.ts`

Add/modify labels:

```typescript
// Batch term labels
consignmentAddTerm: 'Tambah Term (Batch)',
consignmentSelectProduct: 'Pilih produk',
consignmentSearchProduct: 'Cari produk',
consignmentProductSelected: '{count} produk dipilih',
consignmentSelectAtLeastOneProduct: 'Pilih minimal satu produk',
consignmentProductsPreview: 'Produk dipilih ({count})',
consignmentRemoveProduct: 'Hapus',

// Single product creation
consignmentCreateNewProduct: '+ Buat 1 produk baru',
consignmentNewProduct: 'Produk Baru',

// Batch product creation
consignmentCreateMultipleProducts: '+ Buat beberapa produk',
consignmentNewProducts: 'Buat Beberapa Produk',
consignmentBatchProductHint: 'Tambah baris produk baru untuk dibuat sekaligus. SKU akan di-generate otomatis.',
consignmentAddRow: 'Tambah baris',
consignmentCostHint: 'Harga modal dapat diisi nanti di halaman produk.',
consignmentProductNamePlaceholder: 'Ketik nama produk...',
consignmentAtLeastOneProduct: 'Masukkan minimal satu produk',
consignmentSkuRequired: 'SKU wajib diisi',
consignmentPriceNotNegative: 'Harga tidak boleh negatif',
consignmentProductsCreated: (count: number) => `${count} produk berhasil dibuat`,
consignmentCreateNProducts: (count: number) => count > 0 ? `Buat ${count} Produk` : 'Buat',
consignmentProductCreateError: 'Gagal membuat produk',
consignmentProductCreated: 'Produk berhasil dibuat',
```

## UX Considerations

### Dropdown Behavior

1. **Multi-select with checkboxes** — Each product row has a checkbox
2. **Selected state persists** — Dropdown can be closed/reopened without losing selection
3. **Selection counter** — Shows "X products selected" when dropdown is closed
4. **Select all / Deselect all** — Optional bulk actions in dropdown header

### Product Creation Options

1. **Single product** — Creates one product at a time (for quick additions)
2. **Batch products** — Spreadsheet-like table for creating multiple products at once
3. **Auto-selection** — Created products are automatically added to the selection

### Pricing Section

1. **Shared pricing** — Single price/share fields apply to ALL selected products
2. **Preview table** — Shows each selected product with the shared pricing
3. **Per-product customization** — (Future enhancement) Allow editing individual rows in preview

### Validation

1. **Minimum selection** — At least one product must be selected
2. **Duplicate check** — Prevent adding products that already have terms
3. **Conflict check** — Prevent products owned by other suppliers
4. **Batch product validation** — Each row must have name; SKU and price validated

### Edge Cases

1. **Empty dropdown** — Show only create options
2. **Large selection (50+ products)** — Preview table scrolls, performance OK
3. **Cancel with selection** — Clear selection when modal closes
4. **After save** — Clear selection, refresh product list, close modal
5. **Batch creation with some invalid rows** — Show error, keep modal open

## Implementation Phases

### Phase 1: Core Multi-Select (MVP)

- Multi-select dropdown with checkboxes
- Shared pricing for all selected products
- Preview table showing selected products
- Batch term creation
- Single product creation option

### Phase 2: Batch Product Creation

- Spreadsheet-like table for multiple products
- Auto-generate SKU for first row
- Add/remove rows dynamically
- Batch create all valid products

### Phase 3: Enhanced UX (Optional)

- Per-product pricing override in preview
- Select all / deselect all buttons
- Product grouping/filtering in dropdown
- Keyboard navigation for power users
- CSV/Excel import for product list

## Testing

1. **Multi-select flow** — Select 5 products → Set pricing → Save → Verify 5 terms created
2. **Single create** — Create 1 product → Auto-added to selection → Save
3. **Batch create** — Create 5 products via table → All added to selection → Save
4. **Remove from preview** — Click × to remove product from selection
5. **Validation** — Empty selection, duplicate products, conflict products
6. **Large batch** — Select 50 products → Verify performance
7. **Cancel** — Close modal → Verify selection cleared
