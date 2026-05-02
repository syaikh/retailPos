<script>
  import { onMount } from 'svelte';
  import { apiFetch } from '$lib/api/client';
  import { toast } from '$lib/stores/toast';
  import { debounce } from '$lib/utils/debounce';
  import PageHeader from '$lib/components/ui/PageHeader.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import Pagination from '$lib/components/ui/Pagination.svelte';
  import {
    Search, Plus, Pencil, Trash2, Package,
    SlidersHorizontal, AlertTriangle, Loader2
  } from 'lucide-svelte';

  let loading = $state(true);
  let products = $state([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state('');
  let selectedCategory = $state('all');
  let lowStockOnly = $state(false);
  let showModal = $state(false);
  let showDeleteModal = $state(false);
  let selectedProduct = $state(null);
  let modalMode = $state('add'); // 'add' or 'edit'
  let saving = $state(false);

  // Form State
  let form = $state({
    name: '',
    sku: '',
    category: 'Makanan',
    price: 0,
    stock: 0,
    stock_min: 5
  });

  const categories = ['all', 'Makanan', 'Minuman', 'Snack', 'Lainnya'];

  async function fetchProducts() {
    try {
      loading = true;
      const params = new URLSearchParams({
        limit: limit.toString(),
        offset: offset.toString(),
        search: searchQuery
      });
      if (selectedCategory !== 'all') params.append('category', selectedCategory);
      if (lowStockOnly) params.append('maxStock', '5'); // Simplified logic

      const r = await apiFetch(`/api/products?${params.toString()}`);
      if (r.ok) {
        const data = await r.json();
        products = data.data || [];
        total = data.total || 0;
      }
    } catch {
      toast.error('Failed to load inventory');
    } finally {
      loading = false;
    }
  }

  // Debounced search
  const debouncedSearch = debounce(() => {
    offset = 0;
    fetchProducts();
  }, 400);

  $effect(() => {
    searchQuery; // Track search
    debouncedSearch();
  });

  $effect(() => {
    // Other filters (no debounce needed for dropdown/toggle)
    selectedCategory;
    lowStockOnly;
    offset;
    limit;
    // We wrap this in a non-tracked fetch to avoid infinite loops if fetch modifies dependencies
    // But here we are just calling it.
    untrackedFetch();
  });

  function untrackedFetch() {
    fetchProducts();
  }

  function handlePageChange(newOffset, newLimit) {
    offset = newOffset;
    limit = newLimit;
  }

  function openAdd() {
    modalMode = 'add';
    form = { name: '', sku: '', category: 'Makanan', price: 0, stock: 0, stock_min: 5 };
    showModal = true;
  }

  function openEdit(product) {
    modalMode = 'edit';
    selectedProduct = product;
    form = { ...product };
    showModal = true;
  }

  async function saveProduct() {
    if (!form.name || !form.sku) {
      toast.error('Name and SKU are required');
      return;
    }
    
    try {
      saving = true;
      const method = modalMode === 'add' ? 'POST' : 'PUT';
      const url = modalMode === 'add' ? '/api/products' : `/api/products/${selectedProduct.id}`;
      
      const r = await apiFetch(url, {
        method,
        body: JSON.stringify(form)
      });

      if (r.ok) {
        toast.success(modalMode === 'add' ? 'Product created' : 'Product updated');
        showModal = false;
        await fetchProducts();
      } else {
        const err = await r.json();
        toast.error(err.error || 'Failed to save product');
      }
    } catch {
      toast.error('Network error');
    } finally {
      saving = false;
    }
  }

  function openDelete(product) {
    selectedProduct = product;
    showDeleteModal = true;
  }

  async function confirmDelete() {
    if (!selectedProduct) return;
    try {
      const r = await apiFetch(`/api/products/${selectedProduct.id}`, { method: 'DELETE' });
      if (r.ok) {
        toast.success(`"${selectedProduct.name}" removed`);
        await fetchProducts();
      } else {
        toast.error('Failed to delete product');
      }
    } catch {
      toast.error('Failed to delete product');
    } finally {
      showDeleteModal = false;
      selectedProduct = null;
    }
  }

  onMount(fetchProducts);
</script>

<div class="space-y-5">
  <PageHeader title="Inventory" subtitle="Manage products, stock levels, and categories">
    {#snippet actions()}
      <button class="btn btn-primary" onclick={openAdd}>
        <Plus size={16} /> Add Product
      </button>
    {/snippet}
  </PageHeader>

  <!-- Filters -->
  <div class="card p-4 flex flex-col sm:flex-row gap-3">
    <div class="relative flex-1">
      <Search size={16} class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none" />
      <input type="text" placeholder="Search products by name or SKU…" class="input pl-9" bind:value={searchQuery} />
    </div>

    <select class="select sm:w-44" bind:value={selectedCategory}>
      {#each categories as cat}
        <option value={cat}>{cat === 'all' ? 'All Categories' : cat}</option>
      {/each}
    </select>

    <button
      class="btn {lowStockOnly ? 'btn-warning' : 'btn-secondary'} gap-2"
      onclick={() => lowStockOnly = !lowStockOnly}
    >
      <AlertTriangle size={14} />
      Low Stock
    </button>
  </div>

  <!-- Table -->
  <div class="card p-0 overflow-hidden">
    <div class="px-4 py-3 border-b border-border flex items-center justify-between">
      <p class="text-sm font-semibold text-text-primary">Product List</p>
      {#if !loading}
        <span class="badge badge-muted">{total} products</span>
      {/if}
    </div>

    {#if loading}
      <div class="divide-y divide-border">
        {#each { length: 6 } as _}
          <div class="flex items-center gap-4 px-4 py-3.5">
            <Skeleton width="w-44" height="h-4" />
            <Skeleton width="w-24" height="h-4" class="ml-auto" />
            <Skeleton width="w-16" height="h-6" rounded="rounded-full" />
            <Skeleton width="w-24" height="h-4" />
            <Skeleton width="w-16" height="h-8" rounded="rounded-xl" />
          </div>
        {/each}
      </div>
    {:else if products.length === 0}
      <div class="empty-state py-20">
        <div class="empty-state-icon bg-surface w-20 h-20">
          <Package size={32} class="text-text-muted" />
        </div>
        <p class="text-text-primary font-semibold">No products found</p>
        <p class="text-text-muted text-sm mt-1">
          {searchQuery ? `No results for "${searchQuery}"` : 'Start by adding a product'}
        </p>
        <button class="btn btn-primary mt-4" onclick={openAdd}>
          <Plus size={14} /> Add First Product
        </button>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table>
          <thead>
            <tr>
              <th>Product</th>
              <th>SKU</th>
              <th>Category</th>
              <th class="text-center">Stock</th>
              <th class="text-right">Price</th>
              <th class="text-center">Actions</th>
            </tr>
          </thead>
          <tbody>
            {#each products as product (product.id)}
              <tr>
                <td>
                  <div class="font-medium text-text-primary">{product.name}</div>
                </td>
                <td>
                  <span class="font-mono text-xs text-text-muted bg-surface px-2 py-0.5 rounded-md">{product.sku}</span>
                </td>
                <td class="text-text-secondary">{product.category || '—'}</td>
                <td class="text-center">
                  {#if product.stock === 0}
                    <Badge variant="danger">Out of stock</Badge>
                  {:else if product.stock <= (product.stock_min || 5)}
                    <Badge variant="warning">Low: {product.stock}</Badge>
                  {:else}
                    <Badge variant="success">{product.stock}</Badge>
                  {/if}
                </td>
                <td class="text-right font-semibold text-text-primary">
                  Rp {product.price?.toLocaleString('id-ID')}
                </td>
                <td>
                  <div class="flex items-center justify-center gap-2">
                    <button 
                      class="btn-icon btn-ghost text-text-muted hover:text-primary-light" 
                      title="Edit"
                      onclick={() => openEdit(product)}
                    >
                      <Pencil size={15} />
                    </button>
                    <button
                      class="btn-icon btn-ghost text-text-muted hover:text-danger hover:bg-danger-subtle"
                      onclick={() => openDelete(product)}
                      title="Delete"
                    >
                      <Trash2 size={15} />
                    </button>
                  </div>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

      <div class="p-4 bg-surface-subtle/30">
        <Pagination 
          {total} 
          {limit} 
          {offset} 
          onPageChange={handlePageChange} 
        />
      </div>
    {/if}
  </div>
</div>

<!-- Add/Edit Product Modal -->
<Modal bind:open={showModal} title={modalMode === 'add' ? 'Add New Product' : 'Edit Product'} size="md">
  <form onsubmit={(e) => { e.preventDefault(); saveProduct(); }} class="space-y-4">
    <div>
      <label for="inv-name" class="block text-sm font-medium text-text-secondary mb-2">Product Name</label>
      <input id="inv-name" type="text" placeholder="e.g. Indomie Goreng" class="input" bind:value={form.name} required />
    </div>
    <div class="grid grid-cols-2 gap-4">
      <div>
        <label for="inv-sku" class="block text-sm font-medium text-text-secondary mb-2">SKU</label>
        <input id="inv-sku" type="text" placeholder="SKU-001" class="input" bind:value={form.sku} required />
      </div>
      <div>
        <label for="inv-cat" class="block text-sm font-medium text-text-secondary mb-2">Category</label>
        <select id="inv-cat" class="select" bind:value={form.category}>
          {#each categories.slice(1) as cat}
            <option value={cat}>{cat}</option>
          {/each}
        </select>
      </div>
    </div>
    <div class="grid grid-cols-2 gap-4">
      <div>
        <label for="inv-price" class="block text-sm font-medium text-text-secondary mb-2">Price (Rp)</label>
        <input id="inv-price" type="number" placeholder="5000" class="input" bind:value={form.price} min="0" />
      </div>
      <div>
        <label for="inv-stock" class="block text-sm font-medium text-text-secondary mb-2">
          {modalMode === 'add' ? 'Initial Stock' : 'Current Stock'}
        </label>
        <input id="inv-stock" type="number" placeholder="100" class="input" bind:value={form.stock} min="0" />
      </div>
    </div>
    <div class="grid grid-cols-2 gap-4">
      <div>
        <label for="inv-stock-min" class="block text-sm font-medium text-text-secondary mb-2">Min Stock Alert</label>
        <input id="inv-stock-min" type="number" placeholder="5" class="input" bind:value={form.stock_min} min="0" />
      </div>
    </div>
  </form>
  {#snippet footer()}
    <button class="btn btn-secondary" onclick={() => showModal = false} disabled={saving}>Cancel</button>
    <button class="btn btn-primary min-w-32" onclick={saveProduct} disabled={saving}>
      {#if saving}
        <Loader2 size={16} class="animate-spin" /> Saving...
      {:else}
        {modalMode === 'add' ? 'Add Product' : 'Save Changes'}
      {/if}
    </button>
  {/snippet}
</Modal>

<!-- Delete Confirm Modal -->
<Modal bind:open={showDeleteModal} title="Delete Product" size="sm">
  <div class="text-center py-2">
    <div class="w-14 h-14 rounded-2xl bg-danger-subtle flex items-center justify-center mx-auto mb-4">
      <Trash2 size={24} class="text-danger" />
    </div>
    <p class="text-text-primary font-semibold mb-1">Delete "{selectedProduct?.name}"?</p>
    <p class="text-text-muted text-sm">This action cannot be undone and will remove the product from the catalog.</p>
  </div>
  {#snippet footer()}
    <button class="btn btn-secondary" onclick={() => showDeleteModal = false}>Cancel</button>
    <button class="btn btn-danger" onclick={confirmDelete}>Delete</button>
  {/snippet}
</Modal>