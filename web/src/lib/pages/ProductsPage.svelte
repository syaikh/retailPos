<script lang="ts">
  import { onMount } from 'svelte';
  import { fly } from 'svelte/transition';
  import apiClient from '$lib/api/client';
  import { auth } from '$lib/stores/auth';
  import { debounce } from '$lib/utils/debounce';
  import { useWebSocket } from '$lib/composables/useWebSocket';

  import Badge from '$lib/components/ui/Badge.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import CategoryFilterModal from '$lib/components/ui/CategoryFilterModal.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import Pagination from '$lib/components/ui/Pagination.svelte';
  import ProductActionsDropdown from '$lib/components/ui/ProductActionsDropdown.svelte';
  import ProductFormModal from '$lib/components/inventory/ProductFormModal.svelte';
  import SearchBar from '$lib/components/ui/SearchBar.svelte';
  import StockAdjustModal from '$lib/components/inventory/StockAdjustModal.svelte';
  import {
    Plus, Pencil, Trash2, Package,
    SlidersHorizontal, Loader2, Copy, ArrowUpDown, X, ChevronDown, AlertTriangle
  } from 'lucide-svelte';
  import { toast } from '$lib/stores/toast';

  let loading = $state(true);
  let products = $state([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state('');
  let selectedCategories = $state(['All']);
  let categories = $state(['All']);
  let showModal = $state(false);
  let showDeleteModal = $state(false);
  let selectedProduct = $state(null);
  let modalMode = $state('add');
  let saving = $state(false);
  let isDeleting = $state(false);
  let isSearching = $state(false);
  let showDetailDrawer = $state(false);
  let showCopySuccess = $state(null);
  let ws = useWebSocket();
  let brands = $state([]);
  let unitsOfMeasure = $state([]);
  let taxClasses = $state([]);
  let canManageInventory = $state(false);
  let warningThreshold = $state(10);
  let criticalThreshold = $state(5);
  const allowedInventoryRoles = ['superadmin', 'admin', 'manager', 'staff'];
  const allowedStockRoles = ['superadmin', 'admin', 'manager', 'staff'];

  let stockAdjustProduct = $state(null);
  let showAdjustStockModal = $state(false);
  let adjustingStock = $state(false);
  let stockAdjustForm = $state({
    product_id: null,
    quantity_change: 0,
    notes: ''
  });
  let lowStockOnly = $state(false);

  let previousCategories = ['All'];
  let sortBy = $state('name');
  let sortDir = $state('asc');
  let showCategoryFilterModal = $state(false);
  let modalCategorySearch = $state('');

  let categoryBtnStyle = $derived(selectedCategories.length > 0
    ? 'background: rgba(124,58,236,0.12); border-color: rgba(124,58,236,0.35); color: #c4b5fd;'
    : 'background: rgba(30,27,36,0.7); border-color: #374151; color: #9ca3af;'
  );

  const popularCategories = $derived(
    ['Makanan', 'Minuman', 'Snack', 'Lainnya'].filter(cat => categories.includes(cat))
  );

  let form = $state({
    name: '',
    sku: '',
    barcode: '',
    category: '',
    brand_id: null,
    price: 0,
    cost: 0,
    stock: 0,
    unit_of_measure_id: null,
    tax_class_id: null,
    weight_grams: null,
    description: '',
    status: 'draft'
  });

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

  function openProductDetails(product) {
    selectedProduct = product;
    showDetailDrawer = true;
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

  async function fetchCategories() {
    try {
      const r = await apiClient.get('/categories');
      const catList = r.data.data || [];
      categories = ['All', ...catList.map(c => c.name)];
      if (!form.category && catList.length > 0) {
        form.category = catList[0].name;
      }
    } catch (err) {
      toast.error('Failed to load categories');
    }
  }

  async function fetchBrands() {
    try {
      const r = await apiClient.get('/brands');
      brands = r.data.data || [];
    } catch (err) {
      toast.error('Failed to load brands');
    }
  }

  async function fetchTaxClasses() {
    try {
      const r = await apiClient.get('/tax-classes');
      taxClasses = r.data.data || [];
    } catch (err) {
      toast.error('Failed to load tax classes');
    }
  }

  async function fetchUnitsOfMeasure() {
    try {
      const r = await apiClient.get('/units-of-measure');
      unitsOfMeasure = r.data.data || [];
    } catch (err) {
      toast.error('Failed to load units of measure');
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
      const filteredCategories = selectedCategories.filter(c => c.toLowerCase() !== 'all');
      if (filteredCategories.length > 0) params.append('category', filteredCategories.join(','));
      if (lowStockOnly) params.append('maxStock', criticalThreshold.toString());
      const r = await apiClient.get(`/products?${params.toString()}`);
      products = r.data.data || [];
      total = r.data.total || 0;
    } catch (err) {
      toast.error('Failed to load products');
    } finally {
      loading = false;
      isSearching = false;
    }
  }

  const debouncedSearch = debounce(() => {
    offset = 0;
    fetchProducts(0, limit);
  }, 400);

  let isInitialMount = $state(true);
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


  $effect(() => {
    if (isInitialMount) return;
    const prevCatStr = previousCategories.slice().sort().join(',');
    const currCatStr = selectedCategories.slice().sort().join(',');
    if (prevCatStr === currCatStr) return;
    previousCategories = [...selectedCategories];
    offset = 0;
    fetchProducts(0, limit);
  });

  function toggleCategory(category) {
    if (category === 'All') {
      selectedCategories = ['All'];
    } else {
      if (selectedCategories.includes('All')) {
        selectedCategories = selectedCategories.filter(c => c !== 'All');
      }
      if (selectedCategories.includes(category)) {
        selectedCategories = selectedCategories.filter(c => c !== category);
        if (selectedCategories.length === 0) selectedCategories = ['All'];
      } else {
        selectedCategories = [...selectedCategories, category];
      }
    }
    offset = 0;
  }

  async function handleAdd() {
    if (!canManageInventory) {
      toast.error('Insufficient permission to add products');
      return;
    }
    if (!validateProductForm()) return;
    saving = true;
    try {
      const payload = {
        ...form,
        category_name: form.category,
        barcode: form.barcode?.trim() || undefined,
        description: form.description?.trim() || undefined,
        cost: form.cost >= 0 ? form.cost : undefined,
        weight_grams: form.weight_grams ?? undefined
      };
      await apiClient.post('/products', payload);
      toast.success('Product added');
      showModal = false;
      resetForm();
      await fetchProducts(offset, limit);
    } catch (err) {
      const errorMsg = err.response?.data?.error || err.message || 'Failed to add product';
      console.error('Add product error:', err, errorMsg);
      toast.error(errorMsg);
    } finally {
      saving = false;
    }
  }

  async function handleUpdate() {
    if (!canManageInventory) {
      toast.error('Insufficient permission to update products');
      return;
    }
    if (!validateProductForm()) return;
    saving = true;
    try {
      const payload = {
        ...form,
        category_name: form.category,
        barcode: form.barcode?.trim() || undefined,
        description: form.description?.trim() || undefined,
        cost: form.cost >= 0 ? form.cost : undefined,
        weight_grams: form.weight_grams ?? undefined
      };
      await apiClient.put(`/products/${selectedProduct.id}`, payload);
      toast.success('Product updated');
      showModal = false;
      resetForm();
      await fetchProducts(offset, limit);
    } catch (err) {
      const errorMsg = err.response?.data?.error || err.message || 'Failed to update product';
      console.error('Update product error:', err, errorMsg);
      toast.error(errorMsg);
    } finally {
      saving = false;
    }
  }

  async function handleDelete() {
    if (!selectedProduct) {
      toast.error('No product selected');
      return;
    }
    isDeleting = true;
    try {
      await apiClient.delete(`/products/${selectedProduct.id}`);
      toast.success('Product deleted successfully');
      showDeleteModal = false;
      selectedProduct = null;
      await fetchProducts(offset, limit);
    } catch (err) {
      console.error('Delete error:', err);
      const errorMessage = err.response?.data?.error || err.message || 'Failed to delete product';
      toast.error(errorMessage);
    } finally {
      isDeleting = false;
    }
  }

  function resetForm() {
    form = {
      name: '', sku: '', barcode: '', category: '', price: 0, cost: 0, stock: 0,
      brand_id: null, description: '', unit_of_measure_id: null, tax_class_id: null,
      weight_grams: null, status: 'draft'
    };
    modalCategorySearch = '';
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

  function validateProductForm() {
    if (!form.name.trim() || !form.sku.trim() || !form.category.trim()) {
      toast.error('Please complete all required fields');
      return false;
    }
    if (form.price <= 0) {
      toast.error('Price must be greater than zero');
      return false;
    }
    if (form.stock < 0) {
      toast.error('Stock must not be negative');
      return false;
    }
    return true;
  }

  let isSuperAdmin = $derived(() => getUserRoleName() === 'superadmin');
  let isAdmin = $derived(() => getUserRoleName() === 'admin');
  let isSensitive = $derived(() => ['superadmin', 'admin', 'manager'].includes(getUserRoleName()));
  let isFullAudit = $derived(() => ['superadmin', 'admin'].includes(getUserRoleName()));
  let canEdit = $derived(() => ['superadmin', 'admin', 'manager'].includes(getUserRoleName()));

  let stock_stk = $derived(selectedProduct?.stock ?? 0);

  function statusInfo(status?: string): { variant: 'success' | 'muted' | 'destructive'; label: string } {
    switch ((status || '').toLowerCase()) {
      case 'active': return { variant: 'success', label: 'Active' };
      case 'draft':
      case 'inactive':
        return { variant: 'muted', label: (status || 'Draft').charAt(0).toUpperCase() + (status || 'draft').slice(1) };
      case 'discontinued':
      case 'archived':
        return { variant: 'destructive', label: status!.charAt(0).toUpperCase() + status!.slice(1) };
      default: return { variant: 'muted', label: '- ' };
    }
  }

  let status_ = $derived(statusInfo(selectedProduct?.status || 'draft'));

  let margin = $derived(() => {
    const p = selectedProduct;
    if (!p) return null;
    return (p.price || 0) - (p.cost || 0);
  });

  let marginPct = $derived(() => {
    const p = selectedProduct;
    if (!p) return null;
    const price = p.price;
    const cost = p.cost;
    if (!price || !cost) return null;
    return ((price - cost) / price) * 100;
  });

  let margVal = $derived(margin());
  let margPctVal = $derived(marginPct());
  let margIsLoss = $derived(margVal !== null && margVal < 0);
  let uomLabel = $derived(selectedProduct?.unit_of_measure || selectedProduct?.unit || null);

  function copyToClipboard(value: string, field: string, ms = 2000): void {
    navigator.clipboard.writeText(value).then(() => {
      const base = showCopySuccess || new Set();
      const next = new Set(base);
      next.add(field);
      showCopySuccess = next;
      toast.success('Copied to clipboard');
      setTimeout(() => {
        const removed = new Set(next);
        removed.delete(field);
        showCopySuccess = removed;
      }, ms);
    });
  }

  function formatCurrency(value?: number): string {
    if (value == null || isNaN(value)) return '-';
    return 'Rp ' + value.toLocaleString('id-ID');
  }

  function formatDate(value?: string): string {
    if (!value) return '-';
    const d = new Date(value);
    if (isNaN(d.getTime())) return '-';
    return d.toLocaleDateString('id-ID', {
      year: 'numeric', month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
    });
  }

  $effect(() => {
    canManageInventory = allowedInventoryRoles.includes(getUserRoleName());
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
        case 'price': aVal = a.price || 0; bVal = b.price || 0; break;
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
    isInitialMount = true;
    await Promise.all([
      fetchCategories(), fetchBrands(), fetchTaxClasses(), fetchUnitsOfMeasure(), fetchThresholds()
    ]);
    await fetchProducts(0, limit);
    isInitialMount = false;

    ws.on('product_updated', (data) => {
      const product = products.find(p => p.id === data.id);
      if (product) {
        product.stock = data.stock;
        product.price = data.price;
        toast.info(`Product updated: ${product.name}`);
      }
    });

    ws.on('low_stock_alert', (data) => {
      toast.error(`Low stock alert: ${data.name} (stock: ${data.stock})`);
    });
  });

  function handleWindowKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      if (showDetailDrawer) showDetailDrawer = false;
      if (showDeleteModal) showDeleteModal = false;
      document.dispatchEvent(new CustomEvent('close-all-dropdowns'));
    }
  }
</script>

<svelte:window onkeydown={handleWindowKeydown} />

<CategoryFilterModal
  bind:open={showCategoryFilterModal}
  bind:selectedCategories
  {categories}
  {popularCategories}
  onApply={(cats) => { offset = 0; fetchProducts(0, limit); }}
/>

<div class="space-y-6">
  <div class="card p-4">
    <div class="flex items-center gap-4">
      <div class="flex-2">
        <SearchBar bind:value={searchQuery} placeholder="Search by name, SKU, or barcode..." oninput={handleSearchInput} loading={isSearching} inputClass="h-10" />
      </div>
      <button
        type="button"
        onclick={() => showCategoryFilterModal = true}
        class="flex items-center gap-[9px] h-10 px-[14px] rounded-xl shrink-0 transition-all duration-200"
        style={categoryBtnStyle}
      >
        <SlidersHorizontal size={15} style="color: {selectedCategories.length > 0 ? '#c4b5fd' : '#9ca3af'}" />
        <span class="text-[13px] font-medium whitespace-nowrap">
          {#if selectedCategories.length > 0 && !(selectedCategories.length === 1 && selectedCategories[0] === 'All')}
            {selectedCategories.length} Kategori Dipilih
          {:else}
            Kategori
          {/if}
        </span>
        <ChevronDown size={13} class="shrink-0 transition-opacity duration-150" style="color: {selectedCategories.length > 0 ? '#c4b5fd' : '#9ca3af'}; opacity: {selectedCategories.length > 0 ? 0.7 : 0.4}" />
      </button>
      <button
        type="button"
        role="switch"
        aria-checked={lowStockOnly}
        onclick={() => { lowStockOnly = !lowStockOnly; offset = 0; fetchProducts(0, limit); }}
        class="flex items-center gap-[9px] h-10 px-[14px] rounded-xl shrink-0 transition-all duration-200 border {lowStockOnly ? 'bg-warning/10 border-warning/30 text-warning-light' : 'bg-surface-default border-border-strong text-text-muted hover:text-text-secondary hover:border-border-strong'}"
      >
        <AlertTriangle size={14} class={lowStockOnly ? 'text-warning-light' : 'text-text-muted'} />
        <span class="text-[13px] font-medium whitespace-nowrap">Low Stock</span>
      </button>
      <button
        onclick={() => {
          if (!canManageInventory) return;
          modalMode = 'add';
          resetForm();
          showModal = true;
        }}
        disabled={!canManageInventory}
        class="btn btn-primary shrink-0 shadow-glow-primary-sm px-5 disabled:opacity-50 disabled:cursor-not-allowed"
        title={canManageInventory ? 'Add product' : 'Requires inventory role'}
      >
        <Plus size={18} />
        Add Product
      </button>
    </div>
  </div>

  <div class="card overflow-hidden">
    {#if loading}
      <table class="w-full table-fixed">
        <thead class="bg-muted/50">
          <tr>
            <th class="text-left p-4 font-semibold" style="width: 38%;">PRODUCT NAME</th>
            <th class="text-left p-4 font-semibold w-48">CATEGORY</th>
            <th class="text-right p-4 font-semibold w-32">PRICE</th>
            <th class="text-right p-4 font-semibold w-20">STOCK</th>
            <th class="text-left p-4 font-semibold w-28">STATUS</th>
            <th class="text-right p-4 font-semibold w-10"></th>
          </tr>
        </thead>
        <tbody>
          {#each Array(5) as _}
            <tr class="border-t border-border">
              <td class="p-4 min-w-0"><Skeleton class="h-4 w-full" /></td>
              <td class="p-4 w-60"><Skeleton class="h-4 w-3/4" /></td>
              <td class="p-4 text-right w-36"><Skeleton class="h-4 w-1/2 ml-auto" /></td>
              <td class="p-4 text-right w-24"><Skeleton class="h-4 w-1/3 ml-auto" /></td>
              <td class="p-4 w-28"><Skeleton class="h-6 w-20 rounded-full" /></td>
              <td class="p-4 w-20"><Skeleton class="h-4 w-8" /></td>
            </tr>
          {/each}
        </tbody>
      </table>
    {:else if products.length === 0}
      <div class="px-4 py-12 text-center">
        <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
          <Package size={32} class="text-text-muted" />
        </div>
        <p class="text-text-primary font-semibold mt-4">No products found</p>
        <p class="text-text-muted text-sm mt-1">
          {searchQuery || (selectedCategories.length > 0 && !selectedCategories.includes('All')) || lowStockOnly ? 'Try adjusting your filters' : 'Start by adding your first product'}
        </p>
      </div>
    {:else}
      <table class="w-full table-fixed">
        <thead class="bg-muted/50">
          <tr>
            <th class="text-left p-4 font-semibold" style="width: 44%;">
              <button class="flex items-center gap-1 hover:text-primary transition-colors" onclick={() => handleSort('name')}>
                PRODUCT NAME <ArrowUpDown size={14} class="text-text-muted" />
              </button>
            </th>
            <th class="text-left p-4 font-semibold w-44">
              <button class="flex items-center gap-1 hover:text-primary transition-colors" onclick={() => handleSort('category')}>
                CATEGORY <ArrowUpDown size={14} class="text-text-muted" />
              </button>
            </th>
            <th class="text-right p-4 font-semibold w-32">
              <button class="flex items-center gap-1 hover:text-primary transition-colors justify-end" onclick={() => handleSort('price')}>
                PRICE <ArrowUpDown size={14} class="text-text-muted" />
              </button>
            </th>
            <th class="text-right p-4 font-semibold w-24">
              <button class="flex items-center gap-1 hover:text-primary transition-colors justify-end" onclick={() => handleSort('stock')}>
                STOCK <ArrowUpDown size={14} class="text-text-muted" />
              </button>
            </th>
            <th class="text-left p-4 font-semibold w-24">STATUS</th>
            <th class="text-right p-4 font-semibold w-10"></th>
          </tr>
        </thead>
        <tbody>
          {#each products as product}
            <tr
              class="border-t border-border hover:bg-surface-hover/50 transition-colors cursor-pointer"
              onclick={() => openProductDetails(product)}
            >
              <td class="p-4 pr-6" style="width: 44%;">
                <div class="font-medium truncate" title={product.name}>{product.name}</div>
                <div class="flex items-baseline gap-2 mt-1 text-xs text-text-muted">
                  <span class="flex items-center gap-1">
                    <button class="text-left hover:text-primary transition-colors truncate max-w-[120px]" title="Salin SKU" onclick={(e) => { e.stopPropagation(); copyToClipboard(product.sku, `sku_${product.id}`); }}>{product.sku}</button>
                    <button class="p-0.5 hover:text-primary transition-colors w-5 h-5 flex items-center justify-center shrink-0" title="Salin SKU" onclick={(e) => { e.stopPropagation(); copyToClipboard(product.sku, `sku_${product.id}`); }}>
                      {#if showCopySuccess?.has(`sku_${product.id}`)}
                        <span class="text-xs text-primary font-bold leading-none">✓</span>
                      {:else}
                        <Copy size={14} class="text-text-muted hover:text-primary" />
                      {/if}
                    </button>
                  </span>
                  {#if product.barcode}
                    <span class="flex items-center gap-1 ml-4">
                      <button class="text-left hover:text-primary transition-colors truncate max-w-[140px]" title="Salin barcode" onclick={(e) => { e.stopPropagation(); copyToClipboard(product.barcode, `barcode_${product.id}`); }}>{product.barcode}</button>
                      <button class="p-0.5 hover:text-primary transition-colors w-5 h-5 flex items-center justify-center shrink-0" title="Salin barcode" onclick={(e) => { e.stopPropagation(); copyToClipboard(product.barcode, `barcode_${product.id}`); }}>
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
              <td class="p-4 w-60">{product.category_name || '-'}</td>
              <td class="p-4 text-right w-36">{product.price?.toLocaleString('id-ID')}</td>
              <td class="p-4 text-right w-24">
                {#if product.stock === 0}
                  <Badge variant="destructive" size="sm">Out of Stock</Badge>
                {:else if product.stock <= criticalThreshold}
                  <Badge variant="destructive" size="sm">{product.stock}</Badge>
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
                  canEdit={canEdit()}
                  canDelete={isSuperAdmin() || isAdmin()}
                  canAdjustStock={allowedStockRoles.includes(getUserRoleName())}
                  onView={() => { selectedProduct = product; showDetailDrawer = true; }}
                  onEdit={() => {
                    selectedProduct = product;
                    form = {
                      name: product.name || '', sku: product.sku || '', barcode: product.barcode || '',
                      category: product.category_name || '', brand_id: product.brand_id || null,
                      price: product.price || 0, cost: product.cost || 0, stock: product.stock || 0,
                      unit_of_measure_id: product.unit_of_measure_id || null,
                      tax_class_id: product.tax_class_id || null,
                      weight_grams: product.weight_grams || null,
                      description: product.description || '', status: product.status || 'draft'
                    };
                    modalCategorySearch = product.category_name || '';
                    modalMode = 'edit';
                    showModal = true;
                  }}
                  onDelete={() => { selectedProduct = product; showDeleteModal = true; }}
                  onAdjustStock={() => openAdjustStock(product)}
                />
              </td>
            </tr>
          {/each}
        </tbody>
       </table>
        {/if}

        {#if !loading && products.length > 0}
          <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50">
            <Pagination {total} {limit} {offset} onPageChange={fetchProducts} />
          </div>
        {/if}
      </div>
    </div>

    <ProductFormModal
  bind:open={showModal}
  bind:mode={modalMode}
  bind:form
  bind:modalCategorySearch
  {brands}
  {unitsOfMeasure}
  {taxClasses}
  {categories}
  {saving}
  isSuperAdmin={isSuperAdmin()}
  isAdmin={isAdmin()}
  onSubmit={() => { modalMode === 'add' ? handleAdd() : handleUpdate(); }}
  onCancel={() => { showModal = false; }}
/>

<Modal bind:open={showDeleteModal} title="Delete Product" size="sm">
  <div class="text-center py-2">
    <div class="w-14 h-14 rounded-2xl bg-danger-subtle flex items-center justify-center mx-auto mb-4">
      <Trash2 size={24} class="text-danger" />
    </div>
    <p class="text-text-primary font-semibold mb-1">Delete "{selectedProduct?.name}"?</p>
    <p class="text-text-muted text-sm">This action cannot be undone and will remove the product from the catalog.</p>
  </div>
  {#snippet footer()}
    <button class="btn btn-secondary px-5" disabled={isDeleting} onclick={() => showDeleteModal = false}>Cancel</button>
    <button class="btn btn-danger px-5" disabled={isDeleting} onclick={() => handleDelete()}>
      {isDeleting ? 'Deleting...' : 'Delete'}
    </button>
  {/snippet}
</Modal>

<StockAdjustModal
  bind:open={showAdjustStockModal}
  bind:stockAdjustProduct
  bind:stockAdjustForm
  {adjustingStock}
  onSubmit={handleAdjustStock}
  onCancel={() => { showAdjustStockModal = false; stockAdjustProduct = null; }}
/>

{#if showDetailDrawer && selectedProduct}
  <div class="fixed inset-0 bg-black/60 z-50" onclick={() => (showDetailDrawer = false)} aria-hidden="true"></div>
    <div
      class="fixed inset-y-0 right-0 w-[480px] max-w-full bg-surface-default border-l border-border shadow-2xl z-[55] flex flex-col transition-transform duration-300 ease-out"
      transition:fly={{ x: 480, duration: 300, easing: t => t * (2 - t) }}
      role="dialog" aria-modal="true" tabindex="-1"
      onkeydown={(e) => { if (e.key === 'Escape') { e.preventDefault(); showDetailDrawer = false; } }}
    >
    <div class="flex items-center justify-between px-6 py-5 border-b border-border shrink-0">
      <div class="flex items-center gap-3">
        <h2 class="text-lg font-bold text-text-primary">Detail Produk</h2>
        <Badge variant={status_.variant} size="sm">{status_.label}</Badge>
      </div>
      <button
        class="p-2 rounded-lg text-text-muted hover:bg-surface-hover hover:text-text-secondary transition-colors"
        onkeydown={(e) => { if (e.key === 'Enter' || e.key === 'Escape' || e.key === ' ') { e.preventDefault(); showDetailDrawer = false; } }}
        onclick={() => { showDetailDrawer = false; }}
        title="Close detail" aria-label="Close detail panel"
      >
        <X size={18} />
      </button>
    </div>

    <div class="flex-1 overflow-y-auto px-6 py-4 pb-28 space-y-3">
      <div class="flex flex-wrap items-center gap-x-4 gap-y-1 mt-0.5">
        <span class="flex items-center gap-1 min-w-0">
          <span class="text-[11px] font-semibold tracking-widest text-text-muted/60">SKU</span>
          <span class="text-text-secondary font-mono text-sm max-w-[130px] truncate">{selectedProduct.sku || '-'}</span>
          <button class="p-0.5 rounded transition-colors" title="Salin SKU" onclick={() => copyToClipboard(selectedProduct.sku, 'sku')}>
            {#if showCopySuccess?.has('sku')}
              <span class="text-sm text-primary font-semibold">✓</span>
            {:else}
              <Copy size={11} class="text-text-muted/70 hover:text-primary transition-colors"/>
            {/if}
          </button>
        </span>
        {#if selectedProduct.barcode}
          <span class="flex items-center gap-1 ml-1">
            <span class="text-[11px] font-semibold tracking-widest text-text-muted/60">Barcode</span>
            <span class="text-text-secondary font-mono text-sm max-w-[150px] truncate">{selectedProduct.barcode}</span>
            <button class="p-0.5 rounded transition-colors" title="Salin barcode" onclick={() => copyToClipboard(selectedProduct.barcode!, 'barcode')}>
              {#if showCopySuccess?.has('barcode')}
                <span class="text-sm text-primary font-semibold">✓</span>
              {:else}
                <Copy size={11} class="text-text-muted/70 hover:text-primary transition-colors"/>
              {/if}
            </button>
          </span>
        {/if}
      </div>

      <div>
        <h3 class="text-lg font-bold text-text-primary leading-tight">{selectedProduct.name || '—'}</h3>
        {#if selectedProduct.category_name || selectedProduct.brand_name}
          <span class="text-sm text-text-muted font-medium mt-1 block">
            {#if selectedProduct.category_name}<span>{selectedProduct.category_name}</span>{/if}
            {#if selectedProduct.category_name && selectedProduct.brand_name}<span class="text-text-muted/40 mx-1.5">•</span>{/if}
            {#if selectedProduct.brand_name}<span>{selectedProduct.brand_name}</span>{/if}
          </span>
        {/if}
      </div>

      <div class="rounded-2xl bg-surface-default border border-border space-y-0 overflow-hidden">
        <div class="px-3.5 py-2 border-b border-border/60 flex items-center gap-1.5">
          <span class="text-base leading-none">📦</span>
          <h4 class="text-xs font-semibold uppercase tracking-wide text-text-muted/80">Stok &amp; Logistik</h4>
        </div>
        <div class="px-3.5 py-2.5 grid grid-cols-2 gap-x-4 gap-y-3">
          <div class="flex items-center gap-2">
            {#if stock_stk <= criticalThreshold}
              <span class="inline-flex items-center justify-center h-8 w-8 rounded-lg shrink-0 font-mono text-primary-light text-sm font-bold leading-none" style="background: rgba(239,68,68,0.12);">{stock_stk}</span>
            {:else if stock_stk <= warningThreshold}
              <span class="inline-flex items-center justify-center h-8 w-8 rounded-lg shrink-0 font-mono text-warning-light text-sm font-bold leading-none" style="background: rgba(245,158,11,0.12);">{stock_stk}</span>
            {:else}
              <span class="inline-flex items-center justify-center h-8 w-8 rounded-lg shrink-0 font-mono text-success-light text-sm font-bold leading-none" style="background: rgba(16,185,129,0.12);">{stock_stk}</span>
            {/if}
            <span class="text-text-secondary text-xs">Unit {uomLabel ? `: ${uomLabel}` : ''}</span>
          </div>
          {#if selectedProduct.weight_grams != null}
            <div class="text-right">
              <span class="text-[10px] text-text-muted/60 font-medium uppercase tracking-wider">Berat Produk</span>
              <p class="text-text-secondary text-xs pt-0.5">
                {selectedProduct.weight_grams >= 1000 ? `${(selectedProduct.weight_grams / 1000).toFixed(1)} kg` : `${selectedProduct.weight_grams} gram`}
              </p>
            </div>
          {/if}
          {#if selectedProduct.store_id || selectedProduct.store_name}
            <div class="text-right col-span-2">
              <span class="text-text-secondary text-xs">{selectedProduct.store_name || `Store #${selectedProduct.store_id ?? '-'}`}</span>
            </div>
          {/if}
        </div>
      </div>

      <div class="rounded-2xl bg-surface-default border border-border space-y-0 overflow-hidden">
        <div class="px-3.5 py-2 border-b border-border/60 flex items-center gap-1.5">
          <span class="text-base leading-none">💰</span>
          <h4 class="text-xs font-semibold uppercase tracking-wide text-text-muted/80">Keuangan</h4>
        </div>
        <div class="p-4 grid grid-cols-2 gap-x-6 gap-y-5">
          <div class="flex flex-col gap-1 border-b border-border/60 pb-3">
            <span class="text-xs font-semibold tracking-wide text-text-muted/80">Harga Jual</span>
            <p class="text-primary-light text-base font-bold mt-0.5">{formatCurrency(selectedProduct.price)}</p>
          </div>
          <div class="flex flex-col gap-1 border-b border-border/60 pb-3">
            <span class="text-xs font-semibold tracking-wide text-text-muted/80">Diskon</span>
            <p class="text-text-secondary text-xs mt-0.5">
              {selectedProduct.default_discount_percent != null ? `${selectedProduct.default_discount_percent}%` : '0%'}
            </p>
          </div>
          {#if isSensitive()}
            <div class="flex flex-col gap-1 border-b border-border/60 pb-3">
              <span class="text-xs font-semibold tracking-wide text-text-muted/80">Harga Beli</span>
              <p class="text-danger-light text-sm font-semibold mt-0.5">{formatCurrency(selectedProduct.cost ?? 0)}</p>
            </div>
            <div class="flex flex-col gap-1 border-b border-border/60 pb-3">
              <span class="text-xs font-semibold tracking-wide text-text-muted/80">Margin</span>
              <p class="text-sm font-bold {margIsLoss ? 'text-danger-light' : 'text-emerald-400'} mt-0.5">
                {margVal !== null ? formatCurrency(margVal) : '—'}
                {#if margPctVal !== null}
                  <span class="{margIsLoss ? 'text-danger-light/70' : 'text-slate-400'} not-italic font-normal text-xs ml-0.5">
                    {margIsLoss ? '-' : ''}{margPctVal.toFixed(1)}%
                  </span>
                {/if}
              </p>
            </div>
          {:else}
            <div class="flex flex-col gap-1 border-b border-border/60 pb-3">
              <span class="text-[11px] text-text-muted/50 tracking-wide">Harga Beli &amp; Margin</span>
              <p class="text-text-muted/40 text-sm italic mt-0.5">(tersembunyi)</p>
            </div>
            <div class="flex flex-col gap-1 border-b border-border/60 pb-3">
              <span class="text-xs font-semibold tracking-wide text-text-muted/80">Margin</span>
              <p class="text-text-muted/40 text-[11px] italic mt-0.5">Hanya tampil untuk admin, manager, dan superadmin</p>
            </div>
          {/if}
          <div class="flex flex-col gap-1 col-span-2 pt-1">
            <span class="text-xs font-semibold tracking-wide text-text-muted/80">Pajak</span>
            <span class="text-text-secondary text-xs">
              {selectedProduct.tax_rate != null ? `${selectedProduct.tax_rate}%` : (selectedProduct.tax_class_id ? `Class #${selectedProduct.tax_class_id}` : '-')}
            </span>
          </div>
        </div>
      </div>

      {#if selectedProduct.description}
        <div class="rounded-2xl bg-surface-default border border-border space-y-0 overflow-hidden">
          <div class="px-3.5 py-2 border-b border-border/60 flex items-center gap-1.5">
            <span class="text-base leading-none">📝</span>
            <h4 class="text-xs font-semibold uppercase tracking-wide text-text-muted/80">Deskripsi</h4>
          </div>
          <div class="px-3.5 py-2.5">
            <p class="text-text-secondary text-xs leading-relaxed whitespace-pre-wrap break-words">{selectedProduct.description}</p>
          </div>
        </div>
      {/if}

      {#if isFullAudit()}
        <div class="rounded-2xl bg-surface-default border border-border space-y-0 overflow-hidden">
          <div class="px-3.5 py-2 border-b border-border/60 flex items-center gap-1.5">
            <span class="text-base leading-none">📅</span>
            <h4 class="text-xs font-semibold uppercase tracking-wide text-text-muted/80">Audit Trail</h4>
          </div>
          <div class="px-4 py-3 grid grid-cols-2 gap-x-6 gap-y-3">
            <div>
              <span class="text-xs font-semibold tracking-wide text-text-muted/80">Dibuat pada</span>
              <p class="text-text-secondary text-xs mt-1">{formatDate(selectedProduct.created_at)}</p>
            </div>
            <div>
              <span class="text-xs font-semibold tracking-wide text-text-muted/80">Diubah pada</span>
              <p class="text-text-secondary text-xs mt-1">{formatDate(selectedProduct.updated_at)}</p>
            </div>
          </div>
        </div>
      {/if}
    </div>

    {#if canEdit()}
      <div class="absolute bottom-0 left-0 right-0 p-4 bg-surface-default border-t border-border/50">
        <div class="flex items-center gap-3">
          {#if (isSuperAdmin() || isAdmin()) && selectedProduct?.stock === 0}
            <button
              class="flex-1 btn btn-secondary rounded-xl px-4 h-11 text-sm font-semibold text-text-secondary border border-border hover:border-danger hover:text-danger hover:bg-danger-subtle transition-all duration-200"
              onclick={() => { showDetailDrawer = false; showDeleteModal = true; }}
            >
              <Trash2 size={15} class="mr-1.5" />Hapus Produk
            </button>
          {/if}
          <button
            class="flex-1 btn btn-primary rounded-xl px-4 h-11 text-sm font-semibold text-white shadow-glow-primary-sm transition-all duration-200"
            onclick={() => {
              showDetailDrawer = false;
              modalMode = 'edit';
              const p = selectedProduct;
              form = {
                name: p.name || '', sku: p.sku || '', barcode: p.barcode || '',
                category: p.category_name || '', brand_id: p.brand_id || null,
                price: p.price || 0, cost: p.cost || 0, stock: p.stock || 0,
                unit_of_measure_id: p.unit_of_measure_id || null,
                tax_class_id: p.tax_class_id || null,
                weight_grams: p.weight_grams || null,
                description: p.description || '', status: p.status || 'draft'
              };
              modalCategorySearch = p.category_name || '';
              showModal = true;
            }}
          >
            <Pencil size={15} class="mr-1.5" />Edit Produk
          </button>
        </div>
      </div>
    {/if}
  </div>
{/if}
