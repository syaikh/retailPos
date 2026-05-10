<script>
  import { onMount } from 'svelte';
  import apiClient from '$lib/api/client';
  import { auth } from '$lib/stores/auth';
  import { debounce } from '$lib/utils/debounce';

  import Badge from '$lib/components/ui/Badge.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import Pagination from '$lib/components/ui/Pagination.svelte';
  import {
    Search, Plus, Pencil, Trash2, Package,
    SlidersHorizontal, AlertTriangle, Loader2, Copy, ArrowUpDown, X
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
  let canManageInventory = $state(false);
  const allowedInventoryRoles = ['superadmin', 'admin', 'inventory officer'];
  
  // Track previous values to avoid duplicate fetches
  let previousSearchQuery = '';
  let previousCategory = 'All';

  // Sorting state
  let sortBy = $state('name'); // 'name', 'category', 'price', 'stock'
  let sortDir = $state('asc'); // 'asc' or 'desc'

  // Category search state
  let categorySearchQuery = $state('');
  let showCategoryDropdown = $state(false);

  // Form State
  let form = $state({
    name: '',
    sku: '',
    barcode: '',
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

  async function fetchProducts(isSearch = false) {
    try {
      if (!isSearch) loading = true;
      const params = new URLSearchParams({
        limit: limit.toString(),
        offset: offset.toString(),
        search: searchQuery
      });
      if (selectedCategory.toLowerCase() !== 'all') params.append('category', selectedCategory);
      if (lowStockOnly) params.append('maxStock', '5'); // Simplified logic

      // GUNAKAN apiClient (Axios) agar Auto-Refresh Token bisa jalan
      const r = await apiClient.get(`/products?${params.toString()}`);
      // Axios otomatis parse JSON, akses via r.data
      products = r.data.data || [];
      total = r.data.total || 0;
    } catch (err) {
      toast.error('Failed to load inventory');
    } finally {
      if (!isSearch) loading = false;
      isSearching = false;
    }
  }

  // Debounced search
  const debouncedSearch = debounce(() => {
    offset = 0;
    fetchProducts(true);
  }, 400);

  // Track if this is initial mount (to prevent double fetch)
  let isInitialMount = $state(true);

  // Watch for search query changes and trigger debounced search
  $effect(() => {
    // Skip the initial render to prevent double fetch
    if (isInitialMount) return;
    
    // Only proceed if searchQuery actually changed
    if (previousSearchQuery === searchQuery) return;
    
    previousSearchQuery = searchQuery;
    
    if (searchQuery === '') {
      // Immediate fetch when clearing search
      offset = 0;
      isSearching = false;
      fetchProducts(false);
    } else {
      isSearching = true;
      debouncedSearch();
    }
  });

  // Watch for category selection changes
  $effect(() => {
    // Skip the initial render
    if (isInitialMount) return;
    
    // Only proceed if selectedCategory actually changed
    if (previousCategory === selectedCategory) return;
    
    previousCategory = selectedCategory;
    offset = 0;
    fetchProducts(false);
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
    offset = 0;
    fetchProducts(false);
  }

  function handleCategoryInputFocus() {
    showCategoryDropdown = true;
  }

  function handleCategoryInputBlur() {
    // Delay hiding dropdown to allow for selection
    setTimeout(() => {
      showCategoryDropdown = false;
      categorySearchQuery = '';
    }, 150);
  }

   async function handleAdd() {
     if (!canManageInventory) {
       toast.error('Insufficient permission to add products');
       return;
     }
     if (!validateProductForm()) return;

     saving = true;
     try {
       const payload = { ...form, category_name: form.category, barcode: form.barcode?.trim() || undefined };
       await apiClient.post('/products', payload);
       toast.success('Product added');
       showModal = false;
       resetForm();
       await fetchProducts(false);
     } catch (err) {
       toast.error('Failed to add product');
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
       const payload = { ...form, category_name: form.category, barcode: form.barcode?.trim() || undefined };
       await apiClient.put(`/products/${selectedProduct.id}`, payload);
       toast.success('Product updated');
       showModal = false;
       resetForm();
       await fetchProducts(false);
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
      await fetchProducts(false);
    } catch (err) {
      toast.error('Failed to delete product');
    }
  }

  function resetForm() {
    form = { name: '', sku: '', barcode: '', category: categories.length > 1 ? categories[1] : '', price: 0, stock: 0, stock_min: 5 };
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

  $effect(() => {
    canManageInventory = allowedInventoryRoles.includes(getUserRoleName());
  });

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

   onMount(async () => {
     isInitialMount = true;
     await fetchCategories();
     await fetchProducts(false);
     isInitialMount = false;
   });
</script>



<svelte:window
  onclick={(e) => {
    if (showCategoryDropdown && !e.target.closest('.relative')) {
      showCategoryDropdown = false;
      categorySearchQuery = '';
    }
  }}
  onkeydown={(e) => {
    if (e.key === 'Escape' && showCategoryDropdown) {
      showCategoryDropdown = false;
      categorySearchQuery = '';
    }
  }}
/>

<div class="space-y-6">
  <!-- Filters -->
  <div class="card p-4">
    <div class="flex items-center gap-4">
      <div class="relative flex-2">
        <Search size={16} class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none" />
        <input
          type="text"
          placeholder="Search products..."
          bind:value={searchQuery}
          class="input pl-10 pr-12"
        />
        {#if isSearching}
          <Loader2 size={14} class="absolute right-4 top-1/2 -translate-y-1/2 text-primary-light animate-spin" />
        {:else if searchQuery}
          <button
            onclick={() => searchQuery = ''}
            class="absolute right-4 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-secondary transition-colors"
            title="Clear search"
          >
            <X size={14} />
          </button>
        {/if}
      </div>
      <!-- Category Search Input -->
      <div class="relative flex-1">
        <Search size={16} class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none" />
        <input
          type="text"
          placeholder="Filter categories"
          bind:value={categorySearchQuery}
          onfocus={handleCategoryInputFocus}
          onblur={handleCategoryInputBlur}
          class="input w-full pl-10 pr-10"
        />
        {#if categorySearchQuery}
          <button
            onclick={() => categorySearchQuery = ''}
            class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-secondary transition-colors"
            title="Clear filter"
          >
            <X size={14} />
          </button>
        {/if}

        {#if showCategoryDropdown && filteredCategories.length > 0}
          <div class="absolute top-full mt-2 w-full card-glass p-1.5 z-50 min-w-0 flex flex-col gap-0.5 max-h-48 overflow-y-auto">
            {#each filteredCategories as cat}
              <button
                onclick={() => selectCategory(cat)}
                class="flex items-center gap-3 px-3 py-2 text-sm font-medium text-text-secondary hover:text-text-primary hover:bg-surface-hover rounded-xl transition-all duration-200 active:scale-[0.98] w-full text-left"
                role="menuitem"
              >
                {cat}
              </button>
            {/each}
          </div>
        {/if}
      </div>
      <label class="flex items-center gap-2 px-4 py-2 rounded-full bg-surface-subtle hover:bg-surface-hover cursor-pointer transition-colors shrink-0">
        <input type="checkbox" bind:checked={lowStockOnly} onchange={() => { offset = 0; fetchProducts(false); }} class="rounded border-border bg-surface text-primary-light focus:ring-primary-light" />
        <span class="text-sm text-text-secondary font-medium">Low stock only</span>
      </label>
      <button
        onclick={() => {
          if (!canManageInventory) return;
          modalMode = 'add';
          resetForm();
          showModal = true;
        }}
        disabled={!canManageInventory}
        class="btn btn-primary rounded-full shrink-0 shadow-glow-primary-sm px-5 disabled:opacity-50 disabled:cursor-not-allowed"
        title={canManageInventory ? 'Add product' : 'Requires inventory role'}
      >
        <Plus size={18} />
        Add Product
      </button>
    </div>
  </div>

  <!-- Table -->
  <div class="card overflow-hidden">
    {#if loading}
      <table class="w-full table-fixed">
        <thead class="bg-muted/50">
          <tr>
            <th class="text-left p-4 font-semibold">PRODUCT NAME</th>
            <th class="text-left p-4 font-semibold w-60">CATEGORY</th>
            <th class="text-right p-4 font-semibold w-36">PRICE</th>
            <th class="text-left p-4 font-semibold w-28">STOCK</th>
            <th class="text-left p-4 font-semibold w-20">ACTIONS</th>
          </tr>
        </thead>
        <tbody>
          {#each Array(5) as _}
            <tr class="border-t border-border">
              <td class="p-4 min-w-0"><Skeleton class="h-4 w-full" /></td>
              <td class="p-4 w-60"><Skeleton class="h-4 w-3/4" /></td>
              <td class="p-4 text-right w-36"><Skeleton class="h-4 w-1/2 ml-auto" /></td>
              <td class="p-4 w-28 text-right"><Skeleton class="h-4 w-1/3 ml-auto" /></td>
              <td class="p-4 w-20"><Skeleton class="h-4 w-8" /></td>
            </tr>
          {/each}
        </tbody>
      </table>
    {:else if products.length === 0}
      <div class="px-4 py-12 text-center">
        <div class="empty-state-icon bg-surface w-20 h-20 mx-auto">
          <Package size={32} class="text-text-muted" />
        </div>
        <p class="text-text-primary font-semibold mt-4">No products found</p>
        <p class="text-text-muted text-sm mt-1">
          {searchQuery || selectedCategory.toLowerCase() !== 'all' ? 'Try adjusting your filters' : 'Start by adding your first product'}
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
                      if (!canManageInventory) return;
                      selectedProduct = product;
                      form = { ...product, barcode: product.barcode || '', category: product.category_name || categories.find(c => c !== 'All') || '' };
                      modalMode = 'edit';
                      showModal = true;
                    }}
                    disabled={!canManageInventory}
                    class="p-1.5 rounded-lg transition-colors hover:bg-primary/10 disabled:opacity-50 disabled:cursor-not-allowed"
                    title={canManageInventory ? 'Edit' : 'Requires inventory role'}
                  >
                    <Pencil size={14} class="text-primary" />
                  </button>
                  <button
                    onclick={() => {
                      selectedProduct = product;
                      showDeleteModal = true;
                    }}
                    disabled={!canManageInventory || product.stock > 0}
                    class="p-1.5 rounded-lg transition-colors hover:bg-destructive/10 disabled:opacity-50 disabled:cursor-not-allowed"
                    title={product.stock > 0 ? 'Cannot delete products with stock remaining' : canManageInventory ? 'Delete' : 'Requires inventory role'}
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
      <label for="prod-barcode" class="block text-sm font-medium text-text-secondary mb-2">Barcode <span class="text-text-muted text-xs">(optional)</span></label>
      <input id="prod-barcode" bind:value={form.barcode} type="text" class="input" placeholder="Optional barcode" />
    </div>
    <div>
      <label for="prod-category" class="block text-sm font-medium text-text-secondary mb-2">Category</label>
       <select id="prod-category" bind:value={form.category} class="input" required>
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
      <button 
        type="button" 
        onclick={() => (showModal = false)} 
        class="btn btn-secondary rounded-full px-5 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        Cancel
      </button>
      <button 
        type="submit" 
        disabled={saving} 
        class="btn btn-primary rounded-full shadow-glow-primary-sm px-5 disabled:opacity-50 disabled:cursor-not-allowed"
      >
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
    <button class="btn btn-secondary rounded-full px-5" onclick={() => showDeleteModal = false}>Cancel</button>
    <button class="btn btn-danger rounded-full px-5" onclick={handleDelete}>Delete</button>
  {/snippet}
</Modal>