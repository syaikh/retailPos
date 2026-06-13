<script lang="ts">
  import { onMount } from 'svelte';
  import apiClient from '$lib/api/client';
  import { auth } from '$lib/stores/auth';
  import { debounce } from '$lib/utils/debounce';

  import Badge from '$lib/components/ui/Badge.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import Pagination from '$lib/components/ui/Pagination.svelte';
  import StockAdjustModal from '$lib/components/inventory/StockAdjustModal.svelte';
  import {
    Search, AlertTriangle, Loader2, ArrowUpDown, X, Package
  } from 'lucide-svelte';

  const toast = {
    success: (message, timeout = 3000) => {
      console.log('✓', message);
      const notification = document.createElement('div');
      notification.textContent = message;
      notification.style.cssText = `
        position: fixed; top: 20px; right: 20px; background: #10b981; color: white;
        padding: 12px 16px; border-radius: 8px; box-shadow: 0 4px 12px rgba(0,0,0,0.15);
        z-index: 9999; animation: slideIn 0.3s ease;
      `;
      document.body.appendChild(notification);
      setTimeout(() => notification.remove(), timeout);
    },
    error: (message, timeout = 3000) => {
      console.error('✗', message);
      const notification = document.createElement('div');
      notification.textContent = message;
      notification.style.cssText = `
        position: fixed; top: 20px; right: 20px; background: #ef4444; color: white;
        padding: 12px 16px; border-radius: 8px; box-shadow: 0 4px 12px rgba(0,0,0,0.15);
        z-index: 9999; animation: slideIn 0.3s ease;
      `;
      document.body.appendChild(notification);
      setTimeout(() => notification.remove(), timeout);
    },
  };

  let loading = $state(true);
  let products = $state([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state('');
  let lowStockOnly = $state(false);
  let isSearching = $state(false);
  let canAdjustStock = $state(false);
  let warningThreshold = $state(10);
  let criticalThreshold = $state(5);
  const allowedStockRoles = ['superadmin', 'admin', 'manager', 'staff'];

  let stockAdjustProduct = $state(null);
  let showAdjustStockModal = $state(false);
  let adjustingStock = $state(false);
  let stockAdjustForm = $state({
    product_id: null,
    quantity_change: 0,
    notes: ''
  });

  let sortBy = $state('name');
  let sortDir = $state('asc');

  let lowStockBtnStyle = $derived(lowStockOnly
    ? 'background: rgba(124,58,236,0.12); border-color: rgba(124,58,236,0.35); color: #c4b5fd;'
    : 'background: rgba(30,27,36,0.7); border-color: #374151; color: #9ca3af;'
  );

  async function fetchThresholds() {
    try {
      const r = await apiClient.get('/stock-thresholds');
      warningThreshold = r.data.warning ?? 10;
      criticalThreshold = r.data.critical ?? 5;
    } catch (err) {
      warningThreshold = 10;
      criticalThreshold = 5;
    }
  }

  async function fetchProducts(newOffset?, newLimit?) {
    if (newOffset !== undefined) offset = newOffset;
    if (newLimit !== undefined) limit = newLimit;
    try {
      loading = true;
      const params = new URLSearchParams({
        limit: limit.toString(),
        offset: offset.toString(),
        search: searchQuery
      });
      if (lowStockOnly) params.append('maxStock', criticalThreshold.toString());
      const r = await apiClient.get(`/products?${params.toString()}`);
      products = r.data.data || [];
      total = r.data.total || 0;
    } catch (err) {
      toast.error('Failed to load stock data');
    } finally {
      loading = false;
      isSearching = false;
    }
  }

  const debouncedSearch = debounce(() => {
    offset = 0;
    fetchProducts(0, limit);
  }, 400);

  let previousSearchQuery = '';

  function handleSearchInput() {
    offset = 0;
    if (searchQuery === '') {
      debouncedSearch.cancel();
      isSearching = false;
      previousSearchQuery = '';
      fetchProducts(0, limit);
      return;
    }
    if (searchQuery === previousSearchQuery) return;
    previousSearchQuery = searchQuery;
    isSearching = true;
    debouncedSearch();
  }

  function clearSearch() {
    searchQuery = '';
    offset = 0;
    isSearching = false;
    previousSearchQuery = '';
    fetchProducts(0, limit);
  }

  function getUserRoleName() {
    const user = $auth.user;
    if (!user) return '';
    if (typeof user.role === 'string') return user.role.toLowerCase();
    if (user.role && typeof user.role === 'object' && user.role.name) return user.role.name.toLowerCase();
    if (user.role_id === 1) return 'superadmin';
    if (user.role_id === 2) return 'admin';
    if (user.role_id === 3) return 'cashier';
    if (user.role_id === 4) return 'manager';
    return '';
  }

  function classifyStock(stock: number): string {
    return stock <= criticalThreshold ? 'destructive' : 'default';
  }

  function openAdjustStock(product) {
    stockAdjustProduct = product;
    stockAdjustForm = {
      product_id: product.id,
      quantity_change: 0,
      notes: ''
    };
    showAdjustStockModal = true;
  }

  async function handleAdjustStock() {
    if (stockAdjustForm.quantity_change === 0) {
      toast.error('Quantity change must be non-zero');
      return;
    }
    const trimmedNotes = stockAdjustForm.notes?.trim();
    if (!trimmedNotes) {
      toast.error('Notes are required - please provide a reason for adjustment');
      return;
    }
    adjustingStock = true;
    try {
      await apiClient.post('/inventory/adjust', {
        product_id: stockAdjustForm.product_id,
        quantity_change: stockAdjustForm.quantity_change,
        notes: trimmedNotes
      });
      toast.success('Stock adjusted successfully');
      showAdjustStockModal = false;
      stockAdjustProduct = null;
      await fetchProducts(offset, limit);
    } catch (err) {
      const errorMsg = err.response?.data?.error || err.message || 'Failed to adjust stock';
      toast.error(errorMsg);
    } finally {
      adjustingStock = false;
    }
  }

  $effect(() => {
    canAdjustStock = allowedStockRoles.includes(getUserRoleName());
  });

  function handleSort(column) {
    if (sortBy === column) {
      sortDir = sortDir === 'asc' ? 'desc' : 'asc';
    } else {
      sortBy = column;
      sortDir = 'asc';
    }
    sortProducts();
  }

  function sortProducts() {
    products.sort((a, b) => {
      let aVal, bVal;
      switch (sortBy) {
        case 'name': aVal = a.name.toLowerCase(); bVal = b.name.toLowerCase(); break;
        case 'category': aVal = (a.category_name || '').toLowerCase(); bVal = (b.category_name || '').toLowerCase(); break;
        case 'stock': aVal = a.stock || 0; bVal = b.stock || 0; break;
        default: return 0;
      }
      if (sortDir === 'asc') {
        return aVal < bVal ? -1 : aVal > bVal ? 1 : 0;
      } else {
        return aVal > bVal ? -1 : aVal < bVal ? 1 : 0;
      }
    });
  }

  onMount(async () => {
    await fetchThresholds();
    await fetchProducts(0, limit);
  });
</script>

<div class="space-y-6">
  <div class="card p-4">
    <div class="flex items-center gap-4">
      <div class="relative flex-2">
        <Search size={16} class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none" />
        <input
          type="text"
          placeholder="Search products by name or SKU..."
          bind:value={searchQuery}
          oninput={handleSearchInput}
          class="input pl-10 pr-12 h-10"
        />
        {#if isSearching}
          <Loader2 size={14} class="absolute right-4 top-1/2 -translate-y-1/2 text-primary-light animate-spin" />
        {:else if searchQuery}
          <button
            onclick={clearSearch}
            class="absolute right-4 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-secondary transition-colors"
            title="Clear search"
          >
            <X size={14} />
          </button>
        {/if}
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={lowStockOnly}
        onclick={() => { lowStockOnly = !lowStockOnly; offset = 0; fetchProducts(0, limit); }}
        class="flex items-center gap-[9px] h-10 px-[14px] rounded-lg shrink-0 transition-all duration-200"
        style={lowStockBtnStyle}
      >
        <span
          class="block rounded-full shrink-0 transition-colors duration-200"
          style="width: 8px; height: 8px; background: {lowStockOnly ? '#c4b5fd' : '#6b7280'}; box-shadow: {lowStockOnly ? '0 0 6px rgba(196,181,253,0.7)' : 'none'};"
        ></span>
        <span class="text-[13px] font-medium whitespace-nowrap">Low Stock</span>
      </button>
    </div>
  </div>

  <div class="card overflow-hidden">
    {#if loading}
      <table class="w-full table-fixed">
        <thead class="bg-muted/50">
          <tr>
            <th class="text-left p-4 font-semibold">PRODUCT NAME</th>
            <th class="text-left p-4 font-semibold w-60">CATEGORY</th>
            <th class="text-right p-4 font-semibold w-28">STOCK</th>
            <th class="text-left p-4 font-semibold w-32">STATUS</th>
            <th class="text-left p-4 font-semibold w-28">ACTIONS</th>
          </tr>
        </thead>
        <tbody>
          {#each Array(5) as _}
            <tr class="border-t border-border">
              <td class="p-4 min-w-0"><Skeleton class="h-4 w-full" /></td>
              <td class="p-4 w-60"><Skeleton class="h-4 w-3/4" /></td>
              <td class="p-4 text-right w-28"><Skeleton class="h-4 w-1/3 ml-auto" /></td>
              <td class="p-4 w-32"><Skeleton class="h-6 w-20 rounded-full" /></td>
              <td class="p-4 w-28"><Skeleton class="h-8 w-20" /></td>
            </tr>
          {/each}
        </tbody>
      </table>
    {:else if products.length === 0}
      <div class="px-4 py-12 text-center">
        <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
          <Package size={32} class="text-text-muted" />
        </div>
        <p class="text-text-primary font-semibold mt-4">No stock data found</p>
        <p class="text-text-muted text-sm mt-1">
          {#if searchQuery || lowStockOnly}
            Try adjusting your filters
          {:else}
            Stock data will appear here once products are added
          {/if}
        </p>
      </div>
    {:else}
      <table class="w-full table-fixed">
        <thead class="bg-muted/50">
          <tr>
            <th class="text-left p-4 font-semibold">
              <button class="flex items-center gap-1 hover:text-primary transition-colors" onclick={() => handleSort('name')}>
                PRODUCT NAME <ArrowUpDown size={14} class="text-text-muted" />
              </button>
            </th>
            <th class="text-left p-4 font-semibold w-60">
              <button class="flex items-center gap-1 hover:text-primary transition-colors" onclick={() => handleSort('category')}>
                CATEGORY <ArrowUpDown size={14} class="text-text-muted" />
              </button>
            </th>
            <th class="text-right p-4 font-semibold w-28">
              <button class="flex items-center gap-1 hover:text-primary transition-colors justify-end" onclick={() => handleSort('stock')}>
                STOCK <ArrowUpDown size={14} class="text-text-muted" />
              </button>
            </th>
            <th class="text-left p-4 font-semibold w-32">STATUS</th>
            <th class="text-left p-4 font-semibold w-28">ACTIONS</th>
          </tr>
        </thead>
        <tbody>
          {#each products as product}
            <tr class="border-t border-border hover:bg-surface-hover/50 transition-colors">
              <td class="p-4 min-w-0">
                <div class="font-medium truncate w-full" title={product.name}>{product.name}</div>
                <div class="flex items-baseline gap-2 mt-1 text-xs text-text-muted">
                  <span>{product.sku}</span>
                </div>
              </td>
              <td class="p-4 w-60">{product.category_name || '-'}</td>
              <td class="p-4 text-right w-28">{product.stock}</td>
              <td class="p-4 w-32">
                {#if product.stock <= criticalThreshold}
                  <Badge variant="destructive">Critical</Badge>
                {:else if product.stock <= warningThreshold}
                  <Badge variant="warning">Low</Badge>
                {:else}
                  <Badge variant="success">In Stock</Badge>
                {/if}
              </td>
              <td class="p-4 w-28">
                {#if canAdjustStock}
                  <button
                    class="btn btn-secondary text-xs rounded-lg px-3 py-1.5"
                    onclick={() => openAdjustStock(product)}
                    title="Adjust stock for {product.name}"
                  >
                    Adjust
                  </button>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>

  <Pagination {total} {limit} {offset} onPageChange={fetchProducts} />
</div>

<StockAdjustModal
  bind:open={showAdjustStockModal}
  bind:stockAdjustProduct
   bind:stockAdjustForm
  {adjustingStock}
  onSubmit={handleAdjustStock}
  onCancel={() => { showAdjustStockModal = false; stockAdjustProduct = null; }}
/>
