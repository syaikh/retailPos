<script lang="ts">
  import { onMount } from 'svelte';
  import apiClient from '$lib/api/client';
  import { auth } from '$lib/stores/auth';
  import { debounce } from '$lib/utils/debounce';
  import { useWebSocket } from '$lib/composables/useWebSocket';

import Badge from '$lib/components/ui/Badge.svelte';
   import Modal from '$lib/components/ui/Modal.svelte';
   import CategoryFilterModal from '$lib/components/ui/CategoryFilterModal.svelte';
   import Skeleton from '$lib/components/ui/Skeleton.svelte';
   import Pagination from '$lib/components/ui/Pagination.svelte';
  import {
     Search, Plus, Pencil, Trash2, Package,
     SlidersHorizontal, AlertTriangle, Loader2, Copy, ArrowUpDown, X, ChevronDown
    } from 'lucide-svelte';

  // Toast notifications
  const toast = {
    success: (message, timeout = 3000) => {
      console.log('✓', message);
      // Create a simple notification
      const notification = document.createElement('div');
      notification.textContent = message;
      notification.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        background: #10b981;
        color: white;
        padding: 12px 16px;
        border-radius: 8px;
        box-shadow: 0 4px 12px rgba(0,0,0,0.15);
        z-index: 9999;
        animation: slideIn 0.3s ease;
      `;
      document.body.appendChild(notification);
      setTimeout(() => notification.remove(), timeout);
    },
    error: (message, timeout = 3000) => {
      console.error('✗', message);
      // Create a simple notification
      const notification = document.createElement('div');
      notification.textContent = message;
      notification.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        background: #ef4444;
        color: white;
        padding: 12px 16px;
        border-radius: 8px;
        box-shadow: 0 4px 12px rgba(0,0,0,0.15);
        z-index: 9999;
        animation: slideIn 0.3s ease;
      `;
      document.body.appendChild(notification);
      setTimeout(() => notification.remove(), timeout);
    },
    info: (message, timeout = 2000) => {
      console.log('ℹ', message);
      // Create a simple notification
      const notification = document.createElement('div');
      notification.textContent = message;
      notification.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        background: #3b82f6;
        color: white;
        padding: 12px 16px;
        border-radius: 8px;
        box-shadow: 0 4px 12px rgba(0,0,0,0.15);
        z-index: 9999;
        animation: slideIn 0.3s ease;
      `;
      document.body.appendChild(notification);
      setTimeout(() => notification.remove(), timeout);
    }
  };

let loading = $state(true);
   let products = $state([]);
   let total = $state(0);
   let limit = $state(20);
   let offset = $state(0);
   let searchQuery = $state('');
   let selectedCategories = $state(['All']); // Support multiple categories
   let categories = $state(['All']); // Will be populated from API
   let lowStockOnly = $state(false);
  let showModal = $state(false);
let showDeleteModal = $state(false);
   let selectedProduct = $state(null);
   let modalMode = $state('add'); // 'add' or 'edit'
   let saving = $state(false);
   let isDeleting = $state(false);
   let isSearching = $state(false);
   let ws = useWebSocket();
   // Phase 1 Extension States
   let brands = $state([]);
   let unitsOfMeasure = $state([]);
   let taxClasses = $state([]);
   let canManageInventory = $state(false);
  const allowedInventoryRoles = ['superadmin', 'admin', 'inventory officer'];
  
// Track previous values to avoid duplicate fetches
    let previousSearchQuery = '';
    let previousCategories = ['All'];

  // Sorting state
    let sortBy = $state('name'); // 'name', 'category', 'price', 'stock'
    let sortDir = $state('asc'); // 'asc' or 'desc'
    let showCategoryFilterModal = $state(false);
    let modalCategorySearch = $state('');
    let showModalCategoryDropdown = $state(false);

    // Derived style for category filter button
    // Empty: dark surface + slate-700 border + muted text
    // Active: purple tint + purple border + bright purple text
    let categoryBtnStyle = $derived(selectedCategories.length > 0
      ? 'background: rgba(124,58,236,0.12); border-color: rgba(124,58,236,0.35); color: #c4b5fd;'
      : 'background: rgba(30,27,36,0.7); border-color: #374151; color: #9ca3af;'
    );

    // Derived style for low-stock toggle button — mirrors Category button when active
    let lowStockBtnStyle = $derived(lowStockOnly
      ? 'background: rgba(124,58,236,0.12); border-color: rgba(124,58,236,0.35); color: #c4b5fd;'
      : 'background: rgba(30,27,36,0.7); border-color: #374151; color: #9ca3af;'
    );
    
    // Popular categories (based on most commonly used)
    const popularCategories = $derived(
      ['Makanan', 'Minuman', 'Snack', 'Lainnya'].filter(cat => categories.includes(cat))
    );

// Form State
    let form = $state({
      name: '',
      sku: '',
      barcode: '',
      category: '',
      brand_id: null,
      price: 0,
      cost: 0,
      stock: 0,
      stock_min: 5,
      unit_of_measure_id: null,
      tax_class_id: null,
      weight_grams: null,
      description: '',
      status: 'draft'
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

async function fetchProducts(newOffset, newLimit) {
       if (newOffset !== undefined) offset = newOffset;
       if (newLimit !== undefined) limit = newLimit;
       try {
         loading = true;
         const params = new URLSearchParams({
           limit: limit.toString(),
           offset: offset.toString(),
           search: searchQuery
         });
         // Filter categories: exclude 'All' and send as comma-separated string
         const filteredCategories = selectedCategories.filter(c => c.toLowerCase() !== 'all');
         if (filteredCategories.length > 0) params.append('category', filteredCategories.join(','));
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
      fetchProducts(0, limit);
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
       fetchProducts(0, limit);
     } else {
       isSearching = true;
       debouncedSearch();
     }
  });

// Watch for category selection changes
    $effect(() => {
      // Skip the initial render
      if (isInitialMount) return;
      
      // Only proceed if selectedCategories actually changed
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

  // Modal category searchable dropdown
    let filteredModalCategories = $derived(
      categories.filter(cat =>
        cat !== 'All' && cat.toLowerCase().includes(modalCategorySearch.toLowerCase())
      )
    );

    function selectModalCategory(category) {
      form.category = category;
      modalCategorySearch = category;
      showModalCategoryDropdown = false;
    }

    function handleModalCategoryFocus() {
      showModalCategoryDropdown = true;
    }

    function handleModalCategoryBlur() {
      // Don't hide if clicking within dropdown
      setTimeout(() => {
        showModalCategoryDropdown = false;
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
       const payload = {
         ...form,
         category_name: form.category,
         barcode: form.barcode?.trim() || undefined,
         description: form.description?.trim() || undefined,
         cost: form.cost || undefined,
         weight_grams: form.weight_grams || undefined
       };
await apiClient.post('/products', payload);
        toast.success('Product added');
        showModal = false;
        resetForm();
        await fetchProducts(offset, limit);
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
       const payload = {
         ...form,
         category_name: form.category,
         barcode: form.barcode?.trim() || undefined,
         description: form.description?.trim() || undefined,
         cost: form.cost || undefined,
         weight_grams: form.weight_grams || undefined
       };
await apiClient.put(`/products/${selectedProduct.id}`, payload);
        toast.success('Product updated');
        showModal = false;
        resetForm();
        await fetchProducts(offset, limit);
      } catch (err) {
        toast.error('Failed to update product');
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
const response = await apiClient.delete(`/products/${selectedProduct.id}`);
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
        name: '',
        sku: '',
        barcode: '',
        category: '',
        price: 0,
        cost: 0,
        stock: 0,
        stock_min: 5,
        brand_id: null,
        description: '',
        unit_of_measure_id: null,
        tax_class_id: null,
        weight_grams: null,
        status: 'draft'
      };
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
    await Promise.all([
      fetchCategories(),
      fetchBrands(),
      fetchTaxClasses(),
      fetchUnitsOfMeasure()
    ]);
    await fetchProducts(0, limit);
    isInitialMount = false;

    // WebSocket event handlers for real-time updates
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
</script>


<!-- No global window handlers needed -->

<!-- Category Filter Modal -->
<CategoryFilterModal
  bind:open={showCategoryFilterModal}
  bind:selectedCategories
  categories={categories}
  popularCategories={popularCategories}
  onApply={(cats) => { offset = 0; fetchProducts(0, limit); }}
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
           class="input pl-10 pr-12 h-10"
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
<!-- Category Filter Button -->
       <button
         type="button"
         onclick={() => showCategoryFilterModal = true}
         class="flex items-center gap-[9px] h-10 px-[14px] rounded-lg shrink-0 transition-all duration-200"
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
       <!-- Low Stock Toggle Button -->
      <button
         type="button"
         role="switch"
         aria-checked={lowStockOnly}
         onclick={() => {
            lowStockOnly = !lowStockOnly;
            offset = 0;
            fetchProducts(0, limit);
          }}
         class="flex items-center gap-[9px] h-10 px-[14px] rounded-lg shrink-0 transition-all duration-200"
         style={lowStockBtnStyle}
       >
       <!-- Dot indicator -->
         <span
           class="block rounded-full shrink-0 transition-colors duration-200"
           style="width: 8px; height: 8px; background: {lowStockOnly ? '#c4b5fd' : '#6b7280'}; box-shadow: {lowStockOnly ? '0 0 6px rgba(196,181,253,0.7)' : 'none'};"
         ></span>
         <span class="text-[13px] font-medium whitespace-nowrap">
           Low Stock
         </span>
       </button>
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
           {searchQuery || selectedCategories.length > 0 && !selectedCategories.includes('All') ? 'Try adjusting your filters' : 'Start by adding your first product'}
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
                <Badge variant={product.stock <= product.stock_min ? 'destructive' : 'default'}>
                  {product.stock}
                </Badge>
              </td>
              <td class="p-4 w-20">
                <div class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button
onclick={() => {
                       if (!canManageInventory) return;
                       selectedProduct = product;
                       form = {
                         name: product.name || '',
                         sku: product.sku || '',
                         barcode: product.barcode || '',
                         category: product.category_name || '',
                         brand_id: product.brand_id || null,
                         price: product.price || 0,
                         cost: product.cost || 0,
                         stock: product.stock || 0,
                         stock_min: product.stock_min || 5,
                         unit_of_measure_id: product.unit_of_measure_id || null,
                         tax_class_id: product.tax_class_id || null,
                         weight_grams: product.weight_grams || null,
                         description: product.description || '',
                         status: product.status || 'draft'
                       };
                       modalCategorySearch = product.category_name || '';
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
     <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
       <div>
         <label for="prod-name" class="block text-sm font-medium text-text-secondary mb-2">Name <span class="text-destructive">*</span></label>
         <input id="prod-name" bind:value={form.name} type="text" class="input" required />
       </div>
       <div>
         <label for="prod-sku" class="block text-sm font-medium text-text-secondary mb-2">SKU <span class="text-destructive">*</span></label>
         <input id="prod-sku" bind:value={form.sku} type="text" class="input" required />
       </div>
     </div>

     <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div>
        <label for="prod-barcode" class="block text-sm font-medium text-text-secondary mb-2">Barcode <span class="text-text-muted text-xs">(optional)</span></label>
        <input id="prod-barcode" bind:value={form.barcode} type="text" class="input" placeholder="Optional barcode" />
      </div>
      <div>
        <label for="prod-category" class="block text-sm font-medium text-text-secondary mb-2">Category <span class="text-destructive">*</span></label>
        <div class="relative">
          <Search size={16} class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none" />
          <input
            type="text"
            id="prod-category"
            placeholder="Select a category"
            bind:value={modalCategorySearch}
            onfocus={handleModalCategoryFocus}
            onblur={handleModalCategoryBlur}
            class="input w-full pl-10 pr-10"
            required
          />
          {#if modalCategorySearch}
            <button
              type="button"
              onclick={() => { modalCategorySearch = ''; form.category = ''; }}
              class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-secondary transition-colors"
              title="Clear"
            >
              <X size={14} />
            </button>
          {:else}
            <ChevronDown size={16} class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none" />
          {/if}
          {#if showModalCategoryDropdown}
            <div class="absolute top-full mt-2 w-full card-glass p-1.5 z-50 min-w-0 flex flex-col gap-0.5 max-h-48 overflow-y-auto">
              {#if filteredModalCategories.length === 0}
                <div class="px-3 py-2 text-sm text-text-muted">No categories found</div>
              {:else}
{#each filteredModalCategories as cat}
                   <button
                     type="button"
                     onmousedown={() => selectModalCategory(cat)}
                     class="flex items-center gap-3 px-3 py-2 text-sm font-medium text-text-secondary hover:text-text-primary hover:bg-surface-hover rounded-xl transition-all duration-200 active:scale-[0.98] w-full text-left"
                     role="menuitem"
                   >
                     {cat}
                   </button>
                 {/each}
              {/if}
            </div>
          {/if}
        </div>
      </div>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <div>
        <label for="prod-brand" class="block text-sm font-medium text-text-secondary mb-2">Brand</label>
        <select id="prod-brand" bind:value={form.brand_id} class="input">
          <option value={null}>Select brand</option>
          {#each brands as brand}
            <option value={brand.id}>{brand.name}</option>
          {/each}
        </select>
      </div>
      <div>
        <label for="prod-uom" class="block text-sm font-medium text-text-secondary mb-2">Unit</label>
        <select id="prod-uom" bind:value={form.unit_of_measure_id} class="input">
          <option value={null}>Select unit</option>
          {#each unitsOfMeasure as uom}
            <option value={uom.id}>{uom.name} ({uom.code})</option>
          {/each}
        </select>
      </div>
      <div>
        <label for="prod-tax" class="block text-sm font-medium text-text-secondary mb-2">Tax Class</label>
        <select id="prod-tax" bind:value={form.tax_class_id} class="input">
          <option value={null}>Select tax</option>
          {#each taxClasses as tax}
            <option value={tax.id}>{tax.name} ({tax.rate_percent}%)</option>
          {/each}
        </select>
      </div>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <div>
        <label for="prod-price" class="block text-sm font-medium text-text-secondary mb-2">Price (IDR) <span class="text-destructive">*</span></label>
        <input id="prod-price" bind:value={form.price} type="number" class="input" required />
      </div>
      <div>
        <label for="prod-cost" class="block text-sm font-medium text-text-secondary mb-2">Cost (IDR)</label>
        <input id="prod-cost" bind:value={form.cost} type="number" class="input" />
      </div>
      <div>
        <label for="prod-stock" class="block text-sm font-medium text-text-secondary mb-2">Stock <span class="text-destructive">*</span></label>
        <input id="prod-stock" bind:value={form.stock} type="number" class="input" required />
      </div>
</div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label for="prod-stock-min" class="block text-sm font-medium text-text-secondary mb-2">Minimum Stock</label>
          <input id="prod-stock-min" bind:value={form.stock_min} type="number" class="input" placeholder="5" />
        </div>
      </div>

      <div>
        <label for="prod-description" class="block text-sm font-medium text-text-secondary mb-2">Description</label>
        <textarea id="prod-description" bind:value={form.description} class="input" rows="2" placeholder="Product description (optional)"></textarea>
      </div>

      <div>
        <label for="prod-status" class="block text-sm font-medium text-text-secondary mb-2">Status</label>
        <select id="prod-status" bind:value={form.status} class="input">
          <option value="draft">Draft</option>
          <option value="active">Active</option>
          <option value="inactive">Inactive</option>
          <option value="discontinued">Discontinued</option>
          <option value="archived">Archived</option>
        </select>
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
    <button class="btn btn-secondary rounded-full px-5" disabled={isDeleting} onclick={() => showDeleteModal = false}>Cancel</button>
    <button class="btn btn-danger rounded-full px-5" disabled={isDeleting} onclick={() => handleDelete()}>
      {isDeleting ? 'Deleting...' : 'Delete'}
    </button>
  {/snippet}
</Modal>