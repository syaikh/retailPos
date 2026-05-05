<script>
  import { onMount } from 'svelte';
  import apiClient from '$lib/api/client';
  import { toast } from '$lib/stores/toast';
  import { debounce } from '$lib/utils/debounce';

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

      // GUNAKAN apiClient (Axios) agar Auto-Refresh Token bisa jalan
      const r = await apiClient.get(`/products?${params.toString()}`);
      // Axios otomatis parse JSON, akses via r.data
      products = r.data.data || [];
      total = r.data.total || 0;
    } catch (err) {
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

  async function handleAdd() {
    saving = true;
    try {
      await apiClient.post('/products', form);
      toast.success('Product added');
      showModal = false;
      resetForm();
      await fetchProducts();
    } catch (err) {
      toast.error('Failed to add product');
    } finally {
      saving = false;
    }
  }

  async function handleUpdate() {
    saving = true;
    try {
      await apiClient.put(`/products/${selectedProduct.id}`, form);
      toast.success('Product updated');
      showModal = false;
      resetForm();
      await fetchProducts();
    } catch (err) {
      toast.error('Failed to update product');
    } finally {
      saving = false;
    }
  }

  async function handleDelete() {
    try {
      await apiClient.delete(`/products/${selectedProduct.id}`);
      toast.success('Product deleted');
      showDeleteModal = false;
      selectedProduct = null;
      await fetchProducts();
    } catch (err) {
      toast.error('Failed to delete product');
    }
  }

  function resetForm() {
    form = { name: '', sku: '', category: 'Makanan', price: 0, stock: 0, stock_min: 5 };
  }

  $effect(() => {
    fetchProducts();
  });

  $inspect({ products, total });
</script>

<div class="space-y-6">
  <!-- Filters -->
  <div class="card p-4">
    <div class="flex items-center gap-3">
      <input
        type="text"
        placeholder="Search products..."
        bind:value={searchQuery}
        oninput={debouncedSearch}
        class="input max-w-xs"
      />
      <select bind:value={selectedCategory} onchange={fetchProducts} class="input max-w-xs">
        {#each categories as cat}
          <option value={cat}>{cat}</option>
        {/each}
      </select>
      <label class="flex items-center gap-2">
        <input type="checkbox" bind:checked={lowStockOnly} onchange={fetchProducts} />
        <span class="text-sm text-text-secondary">Low stock only</span>
      </label>
      <button
        onclick={() => {
          modalMode = 'add';
          resetForm();
          showModal = true;
        }}
        class="btn btn-primary"
      >
        <Plus size={18} />
        Add Product
      </button>
    </div>
  </div>

  <!-- Table -->
  <div class="card overflow-hidden">
    {#if loading}
      <div class="divide-y divide-border">
        {#each Array(5) as _}
          <div class="p-4">
            <Skeleton class="h-10 w-full" />
          </div>
        {/each}
      </div>
    {:else if products.length === 0}
      <div class="px-4 py-12 text-center">
        <div class="empty-state-icon bg-surface w-20 h-20 mx-auto">
          <Package size={32} class="text-text-muted" />
        </div>
        <p class="text-text-primary font-semibold mt-4">No products found</p>
        <p class="text-text-muted text-sm mt-1">
          {searchQuery || selectedCategory !== 'all' ? 'Try adjusting your filters' : 'Start by adding your first product'}
        </p>
      </div>
    {:else}
      <table class="w-full">
        <thead class="bg-muted/50">
          <tr>
            <th class="text-left p-4 font-semibold">SKU</th>
            <th class="text-left p-4 font-semibold">Name</th>
            <th class="text-left p-4 font-semibold">Category</th>
            <th class="text-left p-4 font-semibold">Price</th>
            <th class="text-left p-4 font-semibold">Stock</th>
            <th class="text-left p-4 font-semibold">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each products as product}
            <tr class="border-t border-border hover:bg-muted/30 transition-colors">
              <td class="p-4 font-medium">{product.sku}</td>
              <td class="p-4">{product.name}</td>
              <td class="p-4">{product.category?.name || '-'}</td>
              <td class="p-4">{product.price?.toLocaleString()}</td>
              <td class="p-4">
                <Badge variant={product.stock <= (product.stock_min || 5) ? 'destructive' : 'default'}>
                  {product.stock}
                </Badge>
              </td>
              <td class="p-4">
                <div class="flex items-center gap-2">
                  <button
                    onclick={() => {
                      selectedProduct = product;
                      form = { ...product };
                      modalMode = 'edit';
                      showModal = true;
                    }}
                    class="p-2 hover:bg-primary/10 rounded-lg transition-colors"
                    title="Edit"
                  >
                    <Pencil size={16} class="text-primary" />
                  </button>
                  <button
                    onclick={() => {
                      selectedProduct = product;
                      showDeleteModal = true;
                    }}
                    class="p-2 hover:bg-destructive/10 rounded-lg transition-colors"
                    title="Delete"
                  >
                    <Trash2 size={16} class="text-destructive" />
                  </button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>

  <!-- Pagination -->
  <Pagination {total} {limit} {offset} onPageChange={fetchProducts} />
</div>

<!-- Add/Edit Modal -->
<Modal bind:open={showModal} title={modalMode === 'add' ? 'Add Product' : 'Edit Product'}>
  <form
    onsubmit={(e) => {
      e.preventDefault();
      modalMode === 'add' ? handleAdd() : handleUpdate();
    }}
    class="space-y-4"
  >
    <div>
      <label for="prod-name" class="block text-sm font-medium text-text-secondary mb-2">Name</label>
      <input id="prod-name" bind:value={form.name} type="text" class="input" required />
    </div>
    <div>
      <label for="prod-sku" class="block text-sm font-medium text-text-secondary mb-2">SKU</label>
      <input id="prod-sku" bind:value={form.sku} type="text" class="input" required />
    </div>
    <div>
      <label for="prod-category" class="block text-sm font-medium text-text-secondary mb-2">Category</label>
      <select id="prod-category" bind:value={form.category} class="input">
        {#each categories as cat}
          <option value={cat}>{cat}</option>
        {/each}
      </select>
    </div>
    <div>
      <label for="prod-price" class="block text-sm font-medium text-text-secondary mb-2">Price (IDR)</label>
      <input id="prod-price" bind:value={form.price} type="number" class="input" required />
    </div>
    <div>
      <label for="prod-stock" class="block text-sm font-medium text-text-secondary mb-2">Stock</label>
      <input id="prod-stock" bind:value={form.stock} type="number" class="input" required />
    </div>
    <div class="flex justify-end gap-4 pt-4">
      <button type="button" onclick={() => (showModal = false)} class="btn-secondary">
        Cancel
      </button>
      <button type="submit" disabled={saving} class="btn-primary">
        {saving ? 'Saving...' : modalMode === 'add' ? 'Add' : 'Update'}
      </button>
    </div>
  </form>
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
    <button class="btn btn-danger" onclick={handleDelete}>Delete</button>
  {/snippet}
</Modal>