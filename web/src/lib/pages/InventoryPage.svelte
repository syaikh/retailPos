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
    SlidersHorizontal, AlertTriangle, Loader2, Copy, ArrowUpDown
  } from 'lucide-svelte';

  let loading = $state(true);
  let products = $state([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state('');
  let selectedCategory = $state('All');
  let categories = $state(['All']); // Will be populated from API
  let lowStockOnly = $state(false);
  let showModal = $state(false);
  let showDeleteModal = $state(false);
  let selectedProduct = $state(null);
  let modalMode = $state('add'); // 'add' or 'edit'
  let saving = $state(false);
  let isSearching = $state(false);

  // Sorting state
  let sortBy = $state('name'); // 'name', 'category', 'price', 'stock'
  let sortDir = $state('asc'); // 'asc' or 'desc'

  // Searchable category dropdown state
  let categorySearchQuery = $state('');
  let showCategoryDropdown = $state(false);
  let categoryInputRef = $state(null);

  // Form State
  let form = $state({
    name: '',
    sku: '',
    category: '',
    price: 0,
    stock: 0,
    stock_min: 5
  });

  async function fetchCategories() {
    try {
      const r = await apiClient.get('/categories');
      const catList = r.data.data || [];
      categories = ['All', ...catList.map(c => c.name)];
      // Set default category if not set
      if (!form.category && catList.length > 0) {
        form.category = catList[0].name;
      }
    } catch (err) {
      toast.error('Failed to load categories');
    }
  }

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
      isSearching = false;
    }
  }

  // Debounced search
  const debouncedSearch = debounce(() => {
    offset = 0;
    fetchProducts();
  }, 400);

  // Watch for search query changes and trigger debounced search
  $effect(() => {
    if (searchQuery !== undefined) {
      isSearching = true;
      debouncedSearch();
    }
  });

  // Filtered categories for searchable dropdown
  let filteredCategories = $derived(
    categories.filter(cat =>
      cat.toLowerCase().includes(categorySearchQuery.toLowerCase())
    )
  );

  function selectCategory(category) {
    selectedCategory = category;
    categorySearchQuery = '';
    showCategoryDropdown = false;
    fetchProducts();
  }

  function handleCategoryInputFocus() {
    showCategoryDropdown = true;
  }

  function handleCategoryInputBlur() {
    // Delay hiding dropdown to allow for selection
    setTimeout(() => {
      showCategoryDropdown = false;
    }, 150);
  }

   async function handleAdd() {
     saving = true;
     try {
       const payload = { ...form, category_name: form.category };
       await apiClient.post('/products', payload);
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
       const payload = { ...form, category_name: form.category };
       await apiClient.put(`/products/${selectedProduct.id}`, payload);
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
    form = { name: '', sku: '', category: categories.length > 1 ? categories[1] : '', price: 0, stock: 0, stock_min: 5 };
  }

  function handleSort(column) {
    if (sortBy === column) {
      // Toggle direction
      sortDir = sortDir === 'asc' ? 'desc' : 'asc';
    } else {
      // New column, default to asc
      sortBy = column;
      sortDir = 'asc';
    }
    // Client-side sorting
    sortProducts();
  }

  function sortProducts() {
    products.sort((a, b) => {
      let aVal, bVal;
      switch (sortBy) {
        case 'name':
          aVal = a.name.toLowerCase();
          bVal = b.name.toLowerCase();
          break;
        case 'category':
          aVal = (a.category_name || '').toLowerCase();
          bVal = (b.category_name || '').toLowerCase();
          break;
        case 'price':
          aVal = a.price || 0;
          bVal = b.price || 0;
          break;
        case 'stock':
          aVal = a.stock || 0;
          bVal = b.stock || 0;
          break;
        default:
          return 0;
      }

      if (sortDir === 'asc') {
        return aVal < bVal ? -1 : aVal > bVal ? 1 : 0;
      } else {
        return aVal > bVal ? -1 : aVal < bVal ? 1 : 0;
      }
    });
  }

   $effect(() => {
     fetchProducts();
     fetchCategories();
   });
</script>

<div class="space-y-6">
  <!-- Filters -->
  <div class="card p-4">
    <div class="flex items-center gap-4">
      <div class="relative flex-2">
        <Search size={16} class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" />
        <input
          type="text"
          placeholder="Search products..."
          bind:value={searchQuery}
          class="input pl-10 pr-10"
        />
        {#if isSearching}
          <Loader2 size={14} class="absolute right-4 top-1/2 -translate-y-1/2 text-primary-light animate-spin" />
        {/if}
      </div>
      <!-- Searchable Category Dropdown -->
      <div class="relative flex-1">
        <div class="relative">
          <input
            bind:value={categorySearchQuery}
            placeholder={categorySearchQuery || showCategoryDropdown ? "Search categories..." : selectedCategory}
            onfocus={handleCategoryInputFocus}
            onblur={handleCategoryInputBlur}
            class="input w-full pr-10"
          />
          <div class="absolute right-3 top-1/2 -translate-y-1/2 pointer-events-none">
            <svg class="w-4 h-4 text-text-muted transition-transform {showCategoryDropdown ? 'rotate-180' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
            </svg>
          </div>
        </div>
        {#if showCategoryDropdown && filteredCategories.length > 0}
          <div class="absolute top-full mt-1 w-full bg-surface border border-border rounded-lg shadow-lg z-10 max-h-48 overflow-y-auto">
            {#each filteredCategories as cat}
              <button
                onclick={() => selectCategory(cat)}
                class="w-full text-left px-4 py-2 hover:bg-surface-hover transition-colors first:rounded-t-lg last:rounded-b-lg {selectedCategory === cat ? 'bg-primary-subtle text-primary' : 'text-text-primary'}"
              >
                {cat}
              </button>
            {/each}
          </div>
        {/if}
      </div>
      <label class="flex items-center gap-2 px-4 py-2 rounded-full bg-surface-subtle hover:bg-surface-hover cursor-pointer transition-colors shrink-0">
        <input type="checkbox" bind:checked={lowStockOnly} onchange={fetchProducts} class="rounded border-border bg-surface text-primary-light focus:ring-primary-light" />
        <span class="text-sm text-text-secondary font-medium">Low stock only</span>
      </label>
      <button
        onclick={() => {
          modalMode = 'add';
          resetForm();
          showModal = true;
        }}
        class="btn btn-primary rounded-full shrink-0 shadow-glow-primary-sm px-5"
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
          <table class="w-full table-fixed">
        <thead class="bg-muted/50">
          <tr>
            <th class="text-left p-4 font-semibold">
              <button
                class="flex items-center gap-1 hover:text-primary transition-colors"
                onclick={() => handleSort('name')}
              >
                PRODUCT NAME
                <ArrowUpDown size={14} class="text-text-muted" />
              </button>
            </th>
            <th class="text-left p-4 font-semibold w-60">
              <button
                class="flex items-center gap-1 hover:text-primary transition-colors"
                onclick={() => handleSort('category')}
              >
                CATEGORY
                <ArrowUpDown size={14} class="text-text-muted" />
              </button>
            </th>
            <th class="text-right p-4 font-semibold w-36">
              <button
                class="flex items-center gap-1 hover:text-primary transition-colors justify-end"
                onclick={() => handleSort('price')}
              >
                PRICE
                <ArrowUpDown size={14} class="text-text-muted" />
              </button>
            </th>
            <th class="text-left p-4 font-semibold w-28">
              <button
                class="flex items-center gap-1 hover:text-primary transition-colors"
                onclick={() => handleSort('stock')}
              >
                STOCK
                <ArrowUpDown size={14} class="text-text-muted" />
              </button>
            </th>
            <th class="text-left p-4 font-semibold w-20">ACTIONS</th>
          </tr>
        </thead>
        <tbody>
          {#each products as product}
            <tr class="border-t border-border hover:bg-surface-hover/50 transition-colors group">
              <td class="p-4 min-w-0">
                <!-- Product name (normal size) -->
                <div class="font-medium truncate w-full" title={product.name}>
                  {product.name}
                </div>

                <!-- SKU and Barcode details (smaller font) -->
                <div class="flex items-baseline gap-2 mt-1 text-xs text-text-muted">
                  <!-- SKU with copy button -->
                  <span class="flex items-center gap-1">
                    {product.sku}
                    <button
                      class="p-0.5 hover:text-primary transition-colors"
                      title="Salin SKU"
                      onclick={() => {
                        navigator.clipboard.writeText(product.sku).then(() => {
                          toast.info(`SKU copied: ${product.sku}`, 2000);
                        });
                      }}
                    >
                      <Copy size={14} class="text-text-muted hover:text-primary" />
                    </button>
                  </span>

                  <!-- Barcode with copy button (only if barcode exists) -->
                  {#if product.barcode}
                    <span class="flex items-center gap-1 ml-4">
                      {product.barcode}
                      <button
                        class="p-0.5 hover:text-primary transition-colors"
                        title="Salin barcode"
                        onclick={() => {
                          navigator.clipboard.writeText(product.barcode).then(() => {
                            toast.info(`Barcode copied: ${product.barcode}`, 2000);
                          });
                        }}
                      >
                        <Copy size={14} class="text-text-muted hover:text-primary" />
                      </button>
                    </span>
                  {/if}
                </div>
              </td>
               <td class="p-4 w-60">{product.category_name || '-'}</td>
              <td class="p-4 text-right w-36">{product.price?.toLocaleString('id-ID')}</td>
              <td class="p-4 w-28 text-right">
                <Badge variant={product.stock <= (product.stock_min || 5) ? 'destructive' : 'default'}>
                  {product.stock}
                </Badge>
              </td>
              <td class="p-4 w-20">
                <div class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button
                    onclick={() => {
                      selectedProduct = product;
                      form = { ...product, category: product.category_name || categories.find(c => c !== 'All') || '' };
                      modalMode = 'edit';
                      showModal = true;
                    }}
                    class="p-1.5 hover:bg-primary/10 rounded-lg transition-colors"
                    title="Edit"
                  >
                    <Pencil size={14} class="text-primary" />
                  </button>
                  <button
                    onclick={() => {
                      selectedProduct = product;
                      showDeleteModal = true;
                    }}
                    class="p-1.5 hover:bg-destructive/10 rounded-lg transition-colors"
                    title="Delete"
                  >
                    <Trash2 size={14} class="text-destructive" />
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
         {#each categories.filter(c => c !== 'all') as cat}
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