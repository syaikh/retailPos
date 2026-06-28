<script lang="ts">
  import { onMount } from 'svelte';
  import apiClient from '$shared/api/http-client';
  import { useAuthStore } from '$modules/auth';
  import { debounce } from '$shared/utils/debounce';
  import { useWebSocket } from '$shared/api/websocket';

  import { Button, Modal, Pagination, ImportModal } from '$shared/ui';
  import CategoryFilterModal from '$modules/product/components/CategoryFilterModal.svelte';
  import ProductActionsDropdown from '$modules/product/components/ProductActionsDropdown.svelte';
  import ProductFormModal from '$modules/product/components/ProductFormModal.svelte';
  import ProductDetailDrawer from './ProductDetailDrawer.svelte';
  import StockAdjustModal from '$modules/inventory/components/StockAdjustModal.svelte';
  import ProductFiltersToolbar from './ProductFiltersToolbar.svelte';
  import ProductTable from './ProductTable.svelte';
  import ProductBulkActions from './ProductBulkActions.svelte';
  import { Plus, Pencil, Trash2, Package, Loader2 } from 'lucide-svelte';
  import { toast } from '$shared/stores/toast.svelte';
  import { exportProducts, importProducts } from '../services/product-service';

  const authStore = useAuthStore();

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
  let adjustProductId = $state(null);
  let adjustQuantityChange = $state(0);
  let adjustNotes = $state('');
  let lowStockOnly = $state(false);
  let filterStatus = $state('all');

  let previousCategories = ['All'];
  let sortBy = $state('name');
  let sortDir = $state('asc');
  let showCategoryFilterModal = $state(false);
  let modalCategorySearch = $state('');

  let selectedIds = $state(new Set());
  let showBulkStatusModal = $state(false);
  let bulkStatusTarget = $state('active');
  let isBulkUpdating = $state(false);
  let showImportModal = $state(false);

  function handleExport(format: 'csv' | 'xlsx') {
    exportProducts(format);
  }

  async function handleImport(file: File) {
    return await importProducts(file);
  }

  function clearSelection() {
    selectedIds = new Set();
  }

  async function handleBulkStatusUpdate() {
    isBulkUpdating = true;
    const eligibleIds = products.filter(p => selectedIds.has(p.id) && p.status !== bulkStatusTarget).map(p => p.id);
    const skippedCount = selectedIds.size - eligibleIds.length;
    if (eligibleIds.length === 0) {
      toast.warning(`All selected product(s) already ${bulkStatusTarget}`);
      isBulkUpdating = false;
      showBulkStatusModal = false;
      return;
    }
    try {
      await apiClient.post('/products/bulk/status', { ids: eligibleIds, status: bulkStatusTarget });
      toast.success(`Updated ${eligibleIds.length} product(s) to ${bulkStatusTarget}`);
      if (skippedCount > 0) {
        toast.warning(`${skippedCount} product(s) already ${bulkStatusTarget}`);
      }
      selectedIds = new Set();
      await fetchProducts(offset, limit);
    } catch (err) {
      toast.error(err.response?.data?.error || 'Failed to update product statuses');
    } finally {
      isBulkUpdating = false;
      showBulkStatusModal = false;
    }
  }

  function clearAllFilters() {
    filterStatus = 'all';
    selectedCategories = ['All'];
    lowStockOnly = false;
    offset = 0;
    fetchProducts(0, limit);
  }

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
    adjustProductId = product.id;
    adjustQuantityChange = 0;
    adjustNotes = '';
    showAdjustStockModal = true;
  }

  async function handleAdjustStock() {
    if (Number(adjustQuantityChange) === 0) {
      toast.error('Quantity change must be non-zero');
      return;
    }
    const trimmedNotes = adjustNotes?.trim();
    if (!trimmedNotes) {
      toast.error('Notes are required - please provide a reason for adjustment');
      return;
    }
    adjustingStock = true;
    try {
      await apiClient.post('/inventory/adjust', {
        product_id: adjustProductId,
        quantity_change: Number(adjustQuantityChange),
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
    selectedIds = new Set();
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
      if (filterStatus !== 'all') params.append('status', filterStatus);
      const r = await apiClient.get(`/products?${params.toString()}`);
      products = r.data.data || [];
      total = r.data.total || 0;
    } catch (err) {
      toast.error('Failed to load products');
    } finally {
      loading = false;
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
      previousSearchQuery = '';
      fetchProducts(0, limit);
      return;
    }
    if (searchQuery === previousSearchQuery) return;
    previousSearchQuery = searchQuery;
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
    const defaultTaxId = taxClasses.find(tc => tc.name === 'PPN 11%')?.id ?? null;
    form = {
      name: '', sku: '', barcode: '', category: '', price: 0, cost: 0, stock: 0,
      brand_id: null, description: '', unit_of_measure_id: null, tax_class_id: defaultTaxId,
      weight_grams: null, status: 'draft'
    };
    modalCategorySearch = '';
  }

  function getUserRoleName() {
    const user = authStore.user;
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

  let userRoleName = $derived(getUserRoleName());
  let isSuperAdmin = $derived(() => getUserRoleName() === 'superadmin');
  let isAdmin = $derived(() => getUserRoleName() === 'admin');
  let canCreate = $derived(['superadmin', 'admin'].includes(userRoleName));
  let isSensitive = $derived(() => ['superadmin', 'admin', 'manager'].includes(getUserRoleName()));
  let isFullAudit = $derived(() => ['superadmin', 'admin'].includes(getUserRoleName()));
  let canEdit = $derived(() => ['superadmin', 'admin', 'manager'].includes(getUserRoleName()));

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

  onMount(() => {
    isInitialMount = true;

    (async () => {
      try {
        await Promise.all([
          fetchCategories(), fetchBrands(), fetchTaxClasses(), fetchUnitsOfMeasure(), fetchThresholds()
        ]);
        await fetchProducts(0, limit);
      } catch {
        console.error('Failed to initialize product page data');
      }
      isInitialMount = false;
    })();

    const unsubProduct = ws.on('product_updated', (data) => {
      const product = products.find(p => p.id === data.id);
      if (product) {
        product.stock = data.stock;
        product.price = data.price;
      }
    });

    return () => {
      unsubProduct();
    };
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
  onApply={(cats) => { offset = 0; fetchProducts(0, limit); }}
/>

<div class="space-y-5">
  <ProductFiltersToolbar
    bind:searchQuery
    bind:selectedCategories
    {categories}
    bind:filterStatus
    bind:lowStockOnly
    {canManageInventory}
    {canCreate}
    onsearch={handleSearchInput}
    onfiltercategory={() => showCategoryFilterModal = true}
    onrefresh={() => { offset = 0; fetchProducts(0, limit); }}
    onclearall={clearAllFilters}
    onadd={() => {
      if (!canManageInventory) return;
      modalMode = 'add';
      resetForm();
      showModal = true;
    }}
    onExport={handleExport}
    onImport={() => showImportModal = true}
  />

  <div class="card overflow-hidden">
    <ProductTable
      {products}
      {loading}
      {searchQuery}
      bind:selectedIds
      bind:sortBy
      bind:sortDir
      bind:showCopySuccess
      onsort={handleSort}
      canEdit={canEdit()}
      canDelete={isSuperAdmin() || isAdmin()}
      canAdjustStock={allowedStockRoles.includes(getUserRoleName())}
      {warningThreshold}
      {criticalThreshold}
      onproductclick={openProductDetails}
      oncopy={(value, field) => copyToClipboard(value, field)}
      onedit={(product) => {
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
      ondelete={(product) => { selectedProduct = product; showDeleteModal = true; }}
      onadjuststock={openAdjustStock}
    />

    <ProductBulkActions
      selectedCount={selectedIds.size}
      onstatus={() => showBulkStatusModal = true}
      onclear={clearSelection}
    />

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
    <Button variant="secondary" class="px-5" disabled={isDeleting} onclick={() => showDeleteModal = false}>Cancel</Button>
    <Button variant="danger" class="px-5" disabled={isDeleting} onclick={() => handleDelete()}>
      {isDeleting ? 'Deleting...' : 'Delete'}
    </Button>
  {/snippet}
</Modal>

<StockAdjustModal
  bind:open={showAdjustStockModal}
  bind:stockAdjustProduct
  bind:productId={adjustProductId}
  bind:quantityChange={adjustQuantityChange}
  bind:notes={adjustNotes}
  {adjustingStock}
  onSubmit={handleAdjustStock}
  onCancel={() => { showAdjustStockModal = false; stockAdjustProduct = null; }}
/>

<Modal bind:open={showBulkStatusModal} title="Change Status" size="sm">
  <div class="py-2">
    <p class="text-text-primary font-semibold mb-3">Set status to <span class="text-primary-light">{bulkStatusTarget}</span> for {products.filter(p => selectedIds.has(p.id) && p.status !== bulkStatusTarget).length} of {selectedIds.size} product(s):</p>
    <div class="flex flex-wrap gap-2 justify-center">
      {#each ['active', 'inactive', 'archived'] as status}
        <button
          class="px-4 py-2 rounded-lg text-sm font-medium border transition-all {bulkStatusTarget === status ? 'bg-primary/10 border-primary/30 text-primary-light' : 'bg-surface-default border-border text-text-muted hover:border-border-strong hover:text-text-secondary'}"
          onclick={() => bulkStatusTarget = status}
        >
          {status.charAt(0).toUpperCase() + status.slice(1)}
        </button>
      {/each}
    </div>
  </div>
  {#snippet footer()}
    <Button variant="secondary" class="px-5" disabled={isBulkUpdating} onclick={() => showBulkStatusModal = false}>Cancel</Button>
    <Button variant="primary" class="px-5" disabled={isBulkUpdating} onclick={handleBulkStatusUpdate}>
      {isBulkUpdating ? 'Updating...' : 'Update'}
    </Button>
  {/snippet}
</Modal>

<ProductDetailDrawer
  bind:showDetailDrawer
  bind:showCopySuccess
  {selectedProduct}
  {warningThreshold}
  {criticalThreshold}
  canEdit={canEdit()}
  canDelete={isSuperAdmin() || isAdmin()}
  isSensitive={isSensitive()}
  isFullAudit={isFullAudit()}
  isSuperAdmin={isSuperAdmin()}
  isAdmin={isAdmin()}
  onclose={() => showDetailDrawer = false}
  onedit={() => {
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
  ondelete={() => { showDetailDrawer = false; showDeleteModal = true; }}
/>

<ImportModal
  bind:show={showImportModal}
  title="Import Products"
  templateHeaders={['SKU', 'Name', 'Barcode', 'Category', 'Brand', 'Price', 'Cost', 'Stock', 'Status', 'UnitOfMeasure', 'WeightGrams', 'Description']}
  onImport={handleImport}
/>

