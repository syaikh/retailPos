<script lang="ts">
  import { Badge, Button, Skeleton, SortableHeader } from '$shared/ui';
  import { Package, Copy } from 'lucide-svelte';
  import ProductActionsDropdown from '$modules/product/components/ProductActionsDropdown.svelte';

  let {
    products = [],
    loading = false,
    searchQuery = '',
    selectedIds = $bindable(new Set()),
    sortBy = $bindable('name'),
    sortDir = $bindable('asc'),
    showCopySuccess = $bindable(null as Set<string> | null),
    canEdit = false,
    canDelete = false,
    canAdjustStock = false,
    warningThreshold = 10,
    criticalThreshold = 5,
    onselectall = () => {},
    onselect = (id: number) => {},
    onsort = (col: string) => {},
    onproductclick = (product: any) => {},
    oncopy = (value: string, field: string, productId: number) => {},
    onedit = (product: any) => {},
    ondelete = (product: any) => {},
    onadjuststock = (product: any) => {},
  }: {
    products: any[];
    loading: boolean;
    searchQuery: string;
    selectedIds: Set<number>;
    sortBy: string;
    sortDir: string;
    showCopySuccess: Set<string> | null;
    canEdit: boolean;
    canDelete: boolean;
    canAdjustStock: boolean;
    warningThreshold: number;
    criticalThreshold: number;
    onselectall?: () => void;
    onselect?: (id: number) => void;
    onsort?: (col: string) => void;
    onproductclick?: (product: any) => void;
    oncopy?: (value: string, field: string, productId: number) => void;
    onedit?: (product: any) => void;
    ondelete?: (product: any) => void;
    onadjuststock?: (product: any) => void;
  } = $props();

  let allSelected = $derived(products.length > 0 && products.every(p => selectedIds.has(p.id)));
  let someSelected = $derived(selectedIds.size > 0 && !allSelected);

  function toggleSelectAll() {
    if (allSelected) {
      selectedIds = new Set();
    } else {
      selectedIds = new Set(products.map(p => p.id));
    }
    onselectall();
  }

  function toggleSelect(id: number) {
    const next = new Set(selectedIds);
    if (next.has(id)) { next.delete(id); } else { next.add(id); }
    selectedIds = next;
    onselect(id);
  }

  function statusInfo(status?: string): { variant: 'success' | 'muted' | 'danger'; label: string } {
    switch ((status || '').toLowerCase()) {
      case 'active': return { variant: 'success', label: 'Active' };
      case 'draft':
      case 'inactive':
        return { variant: 'muted', label: (status || 'Draft').charAt(0).toUpperCase() + (status || 'draft').slice(1) };
      case 'discontinued':
      case 'archived':
        return { variant: 'danger', label: status!.charAt(0).toUpperCase() + status!.slice(1) };
      default: return { variant: 'muted', label: '- ' };
    }
  }
</script>

{#if loading}
  <div aria-busy="true" aria-label="Loading products" aria-live="polite" class="overflow-x-auto">
  <table class="w-full table-fixed min-w-[900px]">
    <thead class="bg-muted/50">
      <tr>
        <th class="p-4 font-semibold w-12"></th>
        <th class="text-left p-4 font-semibold" style="width: 28%;">PRODUCT NAME</th>
        <th class="text-left p-4 font-semibold w-52">CATEGORY</th>
        <th class="text-left p-4 font-semibold w-28">BRAND</th>
        <th class="text-left p-4 font-semibold w-24">UOM</th>
        <th class="p-4 font-semibold w-28 text-right"><span class="flex items-center justify-end gap-1">PRICE</span></th>
        <th class="p-4 font-semibold w-20 text-right"><span class="flex items-center justify-end gap-1">STOCK</span></th>
        <th class="text-left p-4 font-semibold w-20">STATUS</th>
        <th class="text-left p-4 font-semibold w-10"></th>
      </tr>
    </thead>
    <tbody>
      {#each Array(5) as _}
        <tr class="border-t border-border">
          <td class="p-4 w-12"><Skeleton class="h-4 w-4" /></td>
          <td class="p-4 min-w-0"><Skeleton class="h-4 w-full" /></td>
          <td class="p-4 w-52"><Skeleton class="h-4 w-3/4" /></td>
          <td class="p-4 w-28"><Skeleton class="h-4 w-2/3" /></td>
          <td class="p-4 w-24"><Skeleton class="h-4 w-1/2" /></td>
          <td class="p-4 text-right w-28"><Skeleton class="h-4 w-1/2 ml-auto" /></td>
          <td class="p-4 text-right w-20"><Skeleton class="h-4 w-1/3 ml-auto" /></td>
          <td class="p-4 w-20"><Skeleton class="h-6 w-16 rounded-full" /></td>
          <td class="p-4 w-10"><Skeleton class="h-4 w-8" /></td>
        </tr>
      {/each}
    </tbody>
  </table>
  </div>
{:else if products.length === 0}
  <div class="px-4 py-12 text-center" role="status" aria-live="polite">
    <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
      <Package size={32} class="text-text-muted" />
    </div>
    <p class="text-text-primary font-semibold mt-4">No products found</p>
    <p class="text-text-muted text-sm mt-1">Try adjusting your filters or start by adding your first product</p>
  </div>
{:else}
  <div class="overflow-x-auto">
  <table class="w-full table-fixed min-w-[900px]">
    <thead class="bg-muted/50">
      <tr>
        <th class="p-4 font-semibold w-12">
          <input type="checkbox" class="h-4 w-4 rounded border-border bg-surface text-primary accent-primary" checked={allSelected} bind:indeterminate={someSelected} onchange={toggleSelectAll} aria-label="Select all products" />
        </th>
        <th class="text-left p-4 font-semibold" style="width: 40%;">
          <SortableHeader label="PRODUCT NAME" column="name" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
        </th>
        <th class="text-left p-4 font-semibold w-52">
          <SortableHeader label="CATEGORY" column="category" sortColumn={sortBy} sortDirection={sortDir} {onsort} />
        </th>
        <th class="text-left p-4 font-semibold w-28">BRAND</th>
        <th class="text-left p-4 font-semibold w-24">UOM</th>
        <th class="p-4 font-semibold w-28">
          <SortableHeader label="PRICE" column="price" sortColumn={sortBy} sortDirection={sortDir} {onsort} align="right" />
        </th>
        <th class="p-4 font-semibold w-20">
          <SortableHeader label="STOCK" column="stock" sortColumn={sortBy} sortDirection={sortDir} {onsort} align="right" />
        </th>
        <th class="text-left p-4 font-semibold w-20">STATUS</th>
        <th class="text-left p-4 font-semibold w-10"></th>
      </tr>
    </thead>
    <tbody>
      {#each products as product}
        <tr
          class="border-t border-border hover:bg-surface-hover/50 transition-colors cursor-pointer"
          onclick={() => onproductclick(product)}
          tabindex="0"
          role="button"
          onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onproductclick(product); } }}
        >
          <td class="p-4 w-12" onclick={(e) => e.stopPropagation()}>
            <input type="checkbox" class="h-4 w-4 rounded border-border bg-surface text-primary accent-primary" checked={selectedIds.has(product.id)} onchange={() => toggleSelect(product.id)} aria-label="Select {product.name}" />
          </td>
          <td class="p-4 pr-6" style="width: 40%;">
            <div class="font-medium truncate" title={product.name}>{product.name}</div>
            <div class="flex items-baseline gap-2 mt-1 text-xs text-text-muted">
              <span class="flex items-center gap-1">
                <button type="button" class="text-left hover:text-primary transition-colors truncate max-w-[120px]" title="Salin SKU" onclick={(e) => { e.stopPropagation(); oncopy(product.sku, `sku_${product.id}`, product.id); }}>{product.sku}</button>
                <button type="button" class="p-0.5 hover:text-primary transition-colors w-5 h-5 flex items-center justify-center shrink-0" title="Salin SKU" aria-label="Salin SKU" onclick={(e) => { e.stopPropagation(); oncopy(product.sku, `sku_${product.id}`, product.id); }}>
                  {#if showCopySuccess?.has(`sku_${product.id}`)}
                    <span class="text-xs text-primary font-bold leading-none">✓</span>
                  {:else}
                    <Copy size={14} class="text-text-muted hover:text-primary" />
                  {/if}
                </button>
              </span>
              {#if product.barcode}
                <span class="flex items-center gap-1 ml-4">
                  <button type="button" class="text-left hover:text-primary transition-colors truncate max-w-[140px]" title="Salin barcode" onclick={(e) => { e.stopPropagation(); oncopy(product.barcode, `barcode_${product.id}`, product.id); }}>{product.barcode}</button>
                  <button type="button" class="p-0.5 hover:text-primary transition-colors w-5 h-5 flex items-center justify-center shrink-0" title="Salin barcode" aria-label="Salin barcode" onclick={(e) => { e.stopPropagation(); oncopy(product.barcode, `barcode_${product.id}`, product.id); }}>
                    {#if showCopySuccess?.has(`barcode_${product.id}`)}
                      <span class="text-xs text-primary font-bold leading-none">✓</span>
                    {:else}
                      <Copy size={14} class="text-text-muted hover:text-primary" />
                    {/if}
                  </button>
                </span>
              {/if}
            </div>
          </td>
          <td class="p-4 w-52">{product.category_name || '-'}</td>
          <td class="p-4 w-28 truncate" title={product.brand_name || ''}>{product.brand_name || '-'}</td>
          <td class="p-4 w-24">{product.unit_of_measure || '-'}</td>
          <td class="p-4 text-right w-28 font-semibold">{product.price?.toLocaleString('id-ID')}</td>
          <td class="p-4 text-right w-20">
            {#if product.stock === 0}
              <Badge variant="danger" size="sm">0</Badge>
            {:else if product.stock <= criticalThreshold}
              <Badge variant="danger" size="sm">{product.stock}</Badge>
            {:else if product.stock <= warningThreshold}
              <Badge variant="warning" size="sm">{product.stock}</Badge>
            {:else}
              <Badge variant="success" size="sm">{product.stock}</Badge>
            {/if}
          </td>
          <td class="p-4 w-24">
            <Badge variant={statusInfo(product.status).variant} size="sm">{statusInfo(product.status).label}</Badge>
          </td>
          <td class="p-4 w-20" style="width: 80px;" onclick={(e) => e.stopPropagation()}>
            <ProductActionsDropdown
              {product}
              {canEdit}
              {canDelete}
              {canAdjustStock}
              onView={() => onproductclick(product)}
              onEdit={() => onedit(product)}
              onDelete={() => ondelete(product)}
              onAdjustStock={() => onadjuststock(product)}
            />
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
  </div>
{/if}
