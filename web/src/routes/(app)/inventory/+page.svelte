<script>
  import { onMount, onDestroy, tick } from 'svelte';
  import { page } from '$app/stores';
  import { products } from '$lib/stores/products';
  import api from '$lib/api.js';
  import { ui } from '$lib/stores/ui';
  import { 
    Plus, 
    Search, 
    Edit, 
    Trash2, 
    Package,
    AlertCircle,
    Download,
    Copy,
    Check,
    MoreVertical,
    Eye,
    X
  } from 'lucide-svelte';
  import SearchableSelect from '$lib/components/SearchableSelect.svelte';
  import Pagination from '$lib/components/Pagination.svelte';
  import StockBadge from '$lib/components/ui/StockBadge.svelte';

  let searchQuery = $state('');
  let showModal = $state(false);
  /** @type {any} */
  let editingProduct = $state(null);
  let loading = $state(false);
  /** @type {HTMLInputElement | null} */
  let searchInput = $state(null);
  let hideEmptyStock = $state(true);
  /** @type {any[]} */
  let groups = $state([]);
  /** @type {number | null} */
  let openMenuId = $state(null);
  let exportDropdownOpen = $state(false);

  let limit = $state(10);
  let offset = $state(0);
  let total = $state(0);
  let sortField = $state('id');
  let sortDir = $state('asc');
  
  /** @type {string | number} */
  let selectedGroupFilter = $state('all');

  let groupOptions = $derived([
    { value: 'all', label: 'Semua Kategori' },
    ...groups.map(g => ({ value: g.id, label: g.name }))
  ]);

  let formGroupOptions = $derived([
    { value: null, label: 'Tanpa Kategori' },
    ...groups.map(g => ({ value: g.id, label: g.name }))
  ]);

  $effect(() => {
    const groupParam = $page.url.searchParams.get('group');
    if (groupParam) {
      selectedGroupFilter = parseInt(groupParam, 10);
    } else {
      selectedGroupFilter = 'all';
    }
  });

  function autofocus(/** @type {HTMLElement} */ node) {
    requestAnimationFrame(() => {
      node.focus({ preventScroll: true });
    });
    return {
      destroy() {
        // no cleanup needed
      }
    };
  }

  function clickOutsideExport(/** @type {MouseEvent} */ e) {
    const dropdown = document.querySelector('.export-dropdown');
    if (exportDropdownOpen && dropdown && !dropdown.contains(/** @type {Node} */ (e.target))) {
      exportDropdownOpen = false;
    }
  }

  // Form state
  let form = $state({
    name: '',
    sku: '',
    barcode: '',
    price: 0,
    stock: 0,
    group_id: null
  });

  // Export state
  let exportFormat = $state('csv');
  let exportLoading = $state(false);

  // Debug: show export for all users for now
  let showExport = true;

  async function exportInventory() {
    exportLoading = true;
    try {
      const token = localStorage.getItem('token');
      const url = `/api/inventory/export?format=${exportFormat}`;
      const resp = await fetch(url, {
        headers: { Authorization: `Bearer ${token}` }
      });
      if (!resp.ok) throw new Error(await resp.text());
      const blob = await resp.blob();
      const downloadUrl = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = downloadUrl;
      a.download = `inventory_export.${exportFormat}`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      window.URL.revokeObjectURL(downloadUrl);
    } catch (/** @type {any} */ e) {
      alert('Export gagal: ' + e.message);
    } finally {
      exportLoading = false;
    }
  }

  onMount(() => {
    fetchGroups();
    document.addEventListener('click', clickOutsideExport);
  });

  onDestroy(() => {
    if (searchDebounceTimer) {
      clearTimeout(searchDebounceTimer);
    }
    document.removeEventListener('click', clickOutsideExport);
  });

  $effect(() => {
    if (!showModal && searchInput) {
      requestAnimationFrame(() => {
        if (searchInput) searchInput.focus({ preventScroll: true });
      });
    }
  });

  async function fetchProducts() {
    loading = true;
    try {
      const q = searchQuery.trim();
      // Hanya kirim parameter search jika > 3 karakter
      const searchParam = q.length >= 3 ? `&search=${q}` : '';
      const gParam = selectedGroupFilter === 'all' ? '' : `&group_id=${selectedGroupFilter}`;
      const url = `/products?limit=${limit}&offset=${offset}&sortBy=${sortField}&sortDir=${sortDir}${searchParam}${gParam}`;
      const resp = await api.get(url);
      const { data, total: totalCount } = resp.data;
      products.set(Array.isArray(data) ? data : []);
      total = totalCount;
    } catch (/** @type {any} */ e) {
      console.error('Failed to fetch products:', e);
      products.set([]);
    } finally {
      loading = false;
    }
  }

  // Debounce timer
  /** @type {ReturnType<typeof setTimeout> | null} */
  let searchDebounceTimer = null;

  // Reload data when filters/paging changes
  $effect(() => {
    const q = searchQuery;
    const off = offset;
    const lim = limit;
    const group = selectedGroupFilter;
    const sort = sortField;
    const dir = sortDir;
    
    // Clear previous timer
    if (searchDebounceTimer) {
      clearTimeout(searchDebounceTimer);
    }
    
    // Debounce search (faster for inventory since it's key-based)
    searchDebounceTimer = setTimeout(() => {
      fetchProducts();
    }, 150);
  });

  function handlePageChange(/** @type {number} */ newOffset, /** @type {number | undefined} */ newLimit) {
    if (newLimit !== undefined) limit = newLimit;
    offset = newOffset;
  }

  async function fetchGroups() {
    try {
      const resp = await api.get('/product-groups?limit=1000');
      groups = Array.isArray(resp.data.data) ? resp.data.data : [];
    } catch (/** @type {any} */ e) {
      console.error('Failed to fetch groups:', e);
      groups = [];
    }
  }

  function openCreate() {
    editingProduct = null;
    form = { name: '', sku: '', barcode: '', price: 0, stock: 0, group_id: null };
    showModal = true;
  }

  /**
   * @param {{ id: number; name: string; sku: string; barcode?: string | null; price: number; stock: number; group_id: number | null; }} p
   */
  function openEdit(/** @type {any} */ p) {
    editingProduct = p;
    form = { ...p, barcode: p.barcode || '' };
    showModal = true;
  }

  async function handleSubmit(/** @type {SubmitEvent} */ event) {
    event.preventDefault();
    try {
      if (editingProduct) {
        await api.put(`/products/${editingProduct.id}`, form);
      } else {
        await api.post('/products', form);
      }
      showModal = false;
      fetchProducts();
    } catch (/** @type {any} */ e) {
      alert(e.response?.data?.error || 'Operation failed');
    }
  }

  async function deleteProduct(/** @type {number} */ id) {
    if (!confirm('Yakin ingin menghapus produk ini?')) return;
    try {
      await api.delete(`/products/${id}`);
      fetchProducts();
    } catch (/** @type {any} */ e) {
      alert(e.response?.data?.error || 'Delete failed');
    }
  }

  function handleSort(/** @type {string} */ field) {
    if (sortField === field) {
      sortDir = sortDir === 'asc' ? 'desc' : 'asc';
    } else {
      sortField = field;
      sortDir = 'asc';
    }
    offset = 0; // Reset to page 1
  }

  // Logic filter stok kosong tetap di frontend atau bisa dipindah ke backend
  // Sesuai permintaan user, sisa data dari API difilter visual
  let displayProducts = $derived.by(() => {
    if (!$products) return [];
    if (hideEmptyStock) return $products.filter(p => p.stock > 0);
    return $products;
  });

  function getStockLevel(/** @type {number} */ stock) {
    if (stock < 5) return 'rendah';
    if (stock < 20) return 'sedang';
    return 'aman';
  }

  function truncateBarcode(/** @type {string} */ barcode) {
    if (!barcode || barcode.length < 9) return barcode || '-';
    return barcode.slice(0, 4) + '••••' + barcode.slice(-4);
  }

  async function copyToClipboard(/** @type {string} */ text, /** @type {string} */ label) {
    try {
      await navigator.clipboard.writeText(text);
      ui.success(`${label} disalin`);
    } catch (e) {
      ui.error(`Gagal menyalin ${label}`);
    }
  }

  function toggleMenu(/** @type {number} */ id) {
    openMenuId = openMenuId === id ? null : id;
  }
</script>

<div class="inventory-container">
  <div class="header">
    <div class="title">
      <Package size={32} color="var(--primary)" />
      <h1>Manajemen Produk</h1>
    </div>
    <button class="add-btn" onclick={openCreate}>
      <Plus size={20} />
      Tambah Produk
    </button>
  </div>

  <div class="actions premium-card glass">
    <div class="action-filters">
      <div class="search-wrapper">
        <span class="icon"><Search size={18} /></span>
        <input
          type="text"
          placeholder="Cari produk, SKU, atau barcode..."
          bind:value={searchQuery}
          bind:this={searchInput}
          use:autofocus
        />
        {#if searchQuery}
          <button class="clear-btn" onclick={() => { searchQuery = ''; searchInput?.focus(); }}>
            <X size={14} />
          </button>
        {/if}
        {#if searchQuery.trim().length > 0 && searchQuery.trim().length < 3}
          <div class="search-warning">Minimal 3 karakter</div>
        {/if}
      </div>
      <SearchableSelect 
        options={groupOptions} 
        bind:value={selectedGroupFilter} 
        width="220px" 
      />
    </div>
    <label class="stock-filter">
      <input type="checkbox" bind:checked={hideEmptyStock} />
      Sembunyikan stok kosong
    </label>
    {#if showExport}
      <div class="export-dropdown">
        <button 
          class="export-btn" 
          onclick={(e) => { e.stopPropagation(); exportDropdownOpen = !exportDropdownOpen; }}
          disabled={exportLoading}
        >
          {#if exportLoading}
            Exporting...
          {:else}
            <Download size={16} /> Export ▾
          {/if}
        </button>
        {#if exportDropdownOpen}
          <div class="export-menu">
            <button onclick={() => { exportFormat = 'csv'; exportInventory(); exportDropdownOpen = false; }}>
              <Download size={16} /> Export CSV
            </button>
            <button onclick={() => { exportFormat = 'xlsx'; exportInventory(); exportDropdownOpen = false; }}>
              <Download size={16} /> Export Excel
            </button>
          </div>
        {/if}
      </div>
    {/if}
  </div>

  {#if selectedGroupFilter !== 'all' || hideEmptyStock}
    <div class="filter-chips">
      {#if selectedGroupFilter !== 'all'}
        <span class="filter-chip">
          {groups.find(g => g.id === selectedGroupFilter)?.name || 'Kategori'}
          <button onclick={() => selectedGroupFilter = 'all'}><X size={12} /></button>
        </span>
      {/if}
      {#if hideEmptyStock}
        <span class="filter-chip">
          Stok Tersembunyi
          <button onclick={() => hideEmptyStock = false}><X size={12} /></button>
        </span>
      {/if}
    </div>
  {/if}

  <div class="table-container premium-card">
    {#if loading}
      <div class="loading-state">
        <div class="loading-spinner"></div>
        <p>Memuat produk...</p>
      </div>
    {:else}
      <table>
      <thead>
        <tr>
          <th class="col-name">Nama Produk</th>
          <th class="col-stock">Stok</th>
          <th class="col-price">Harga</th>
          <th>Kategori</th>
          <th>Diperbarui</th>
          <th class="col-actions">Aksi</th>
        </tr>
      </thead>
      <tbody>
        {#each displayProducts as p}
          <tr>
            <td>
              <div class="product-name">{p.name}</div>
              <div class="product-meta">
                <span class="meta-item">
                  <code>{p.sku}</code>
                  <button 
                    class="copy-btn" 
                    onclick={() => copyToClipboard(p.sku, 'SKU')}
                    title="Salin SKU"
                  >
                    <Copy size={12} />
                  </button>
                </span>
                {#if p.barcode}
                  <span class="meta-item">
                    <code>{truncateBarcode(p.barcode)}</code>
                    <button 
                      class="copy-btn" 
                      onclick={() => copyToClipboard(p.barcode || '', 'Barcode')}
                      title="Salin Barcode"
                    >
                      <Copy size={12} />
                    </button>
                  </span>
                {/if}
              </div>
            </td>
            <td>
              <StockBadge level={getStockLevel(p.stock)} value={p.stock} />
            </td>
            <td class="price-cell">Rp {p.price.toLocaleString()}</td>
            <td>
              {#if p.group_id}
                <span class="group-pill">
                  {groups.find(g => g.id === p.group_id)?.name || '-'}
                </span>
              {:else}
                <span class="text-dim">-</span>
              {/if}
            </td>
            <td class="text-dim">
              {p.updated_at ? new Date(p.updated_at).toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric' }) : '-'}
            </td>
            <td>
              <div class="row-actions">
                <button class="edit-btn" onclick={() => openEdit(p)} title="Edit">
                  <Edit size={18} />
                </button>
                <button 
                  class="del-btn" 
                  onclick={() => deleteProduct(p.id)} 
                  disabled={p.stock > 0}
                  title={p.stock > 0 ? 'Hapus dibatasi: Stok masih tersedia' : 'Hapus Produk'}
                >
                  <Trash2 size={18} />
                </button>
              </div>
            </td>
          </tr>
        {/each}
        {#if displayProducts.length === 0}
          <tr>
            <td colspan="7" class="empty">Tidak ada produk ditemukan</td>
          </tr>
        {/if}
      </tbody>
    </table>
    <Pagination 
      total={total} 
      limit={limit} 
      offset={offset} 
      onPageChange={handlePageChange} 
    />
    {/if}
  </div>
</div>

{#if showModal}
  <div 
    class="modal-overlay" 
    role="button" 
    tabindex="-1" 
    onclick={() => showModal = false}
    onkeydown={(e) => e.key === 'Escape' && (showModal = false)}
  >
    <div class="modal premium-card" role="presentation" onclick={(e) => e.stopPropagation()}>
      <h2>{editingProduct ? 'Edit Produk' : 'Tambah Produk Baru'}</h2>
      <form onsubmit={handleSubmit}>
        <div class="form-row">
          <div class="form-group">
            <label for="product-sku">SKU (Kode Internal)</label>
            <input id="product-sku" type="text" bind:value={form.sku} required use:autofocus />
          </div>
          <div class="form-group">
            <label for="product-barcode">Barcode (External)</label>
            <input id="product-barcode" type="text" bind:value={form.barcode} placeholder="Optional" />
          </div>
        </div>
        <div class="form-group">
          <label for="product-name">Nama Produk</label>
          <input id="product-name" type="text" bind:value={form.name} required />
        </div>
        <div class="form-group">
          <label for="product-group">Kategori</label>
          <SearchableSelect 
            options={formGroupOptions} 
            bind:value={form.group_id} 
            placeholder="Pilih Kategori" 
            width="100%"
          />
        </div>
        <div class="form-row">
          <div class="form-group">
            <label for="product-price">Harga Jual (Rp)</label>
            <input id="product-price" type="number" bind:value={form.price} required min="0" />
          </div>
          <div class="form-group">
            <label for="product-stock">Stok Awal</label>
            <input id="product-stock" type="number" bind:value={form.stock} required min="0" />
          </div>
        </div>
        <div class="modal-actions">
          <button type="button" class="cancel-btn" onclick={() => showModal = false}>Batal</button>
          <button type="submit" class="save-btn">Simpan</button>
        </div>
      </form>
    </div>
  </div>
{/if}

<style>
  .inventory-container {
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .title {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .add-btn {
    background: var(--primary);
    color: white;
    padding: 10px 20px;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .actions {
    display: flex; 
    justify-content: space-between; 
    align-items: center; 
    padding: 16px;
    gap: 16px;
    flex-wrap: wrap;
  }

  .action-filters {
    display: flex;
    gap: 16px;
    align-items: center;
    flex: 1;
  }

  .search-wrapper {
    position: relative;
    flex: 1;
    min-width: 250px;
    max-width: 400px;
  }

  .search-wrapper .icon {
    position: absolute;
    left: 12px;
    top: 50%;
    transform: translateY(-50%);
    color: var(--text-secondary);
  }

  .search-wrapper input {
    width: 100%;
    padding: 10px 10px 10px 40px;
    background: #0f172a;
    border: 1px solid var(--border);
    border-radius: 8px;
    color: white;
    font-size: 1rem;
    font-family: inherit;
    outline: none;
    transition: border-color 0.2s;
  }

  .search-wrapper input:focus {
    border-color: var(--primary);
  }

  .clear-btn {
    position: absolute;
    right: 10px;
    top: 50%;
    transform: translateY(-50%);
    background: transparent;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 4px;
    border-radius: 4px;
    display: flex;
    align-items: center;
  }

  .clear-btn:hover {
    color: white;
    background: rgba(255, 255, 255, 0.1);
  }

  .search-warning {
    position: absolute;
    top: calc(100% + 12px);
    left: 40px;
    font-size: 0.75rem;
    color: #000;
    background: #f59e0b;
    padding: 6px 12px;
    border-radius: 6px;
    pointer-events: none;
    z-index: 100;
    white-space: nowrap;
    font-weight: 700;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
    animation: slideDown 0.2s ease-out;
  }

  .search-warning::before {
    content: '';
    position: absolute;
    top: -6px;
    left: 12px;
    border-left: 6px solid transparent;
    border-right: 6px solid transparent;
    border-bottom: 6px solid #f59e0b;
  }

  @keyframes slideDown {
    from { opacity: 0; transform: translateY(-8px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .filter-chips {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }

  .filter-chip {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 4px 8px 4px 10px;
    background: rgba(99, 102, 241, 0.15);
    color: var(--primary);
    border-radius: 9999px;
    font-size: 0.8rem;
    font-weight: 500;
  }

  .filter-chip button {
    display: flex;
    background: transparent;
    border: none;
    color: var(--primary);
    cursor: pointer;
    padding: 2px;
    border-radius: 50%;
  }

  .filter-chip button:hover {
    background: rgba(99, 102, 241, 0.25);
  }

  .stock-filter {
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--text-secondary);
    cursor: pointer;
    font-size: 0.9rem;
  }

.stock-filter input {
    cursor: pointer;
  }

  code {
    background: #1e293b;
    padding: 2px 6px;
    border-radius: 4px;
    color: var(--accent);
    display: inline-block;
  }

  /* Table Redesign */
  .col-name { width: 30%; }
  .col-stock { width: 12%; text-align: right; }
  .col-price { width: 12%; text-align: right; }
  .col-actions { width: 10%; }

  .product-name {
    font-weight: 600;
    font-size: 1rem;
    color: white;
  }

  .product-meta {
    display: flex;
    gap: 12px;
    margin-top: 4px;
    font-size: 0.8rem;
    color: var(--text-secondary);
  }

  .meta-item {
    display: inline-flex;
    align-items: center;
    gap: 4px;
  }

  .copy-btn {
    background: transparent;
    color: var(--text-secondary);
    padding: 2px;
    border-radius: 4px;
    cursor: pointer;
  }

  .copy-btn:hover {
    color: var(--primary);
  }

  .price-cell {
    text-align: right;
    font-weight: 500;
  }

  .export-dropdown {
    position: relative;
  }

  .export-dropdown .export-btn {
    background: var(--success);
    color: white;
    padding: 8px 12px;
    border-radius: 6px;
    border: none;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .export-dropdown .export-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .export-menu {
    position: absolute;
    top: 100%;
    right: 0;
    margin-top: 4px;
    background: #1e293b;
    border: 1px solid var(--border);
    border-radius: 6px;
    overflow: hidden;
    z-index: 50;
  }

  .export-menu button {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 8px 16px;
    background: transparent;
    border: none;
    color: white;
    text-align: left;
    cursor: pointer;
    font-size: 0.875rem;
    white-space: nowrap;
  }

  .export-menu button:hover {
    background: rgba(99, 102, 241, 0.1);
  }

  .row-actions {
    display: flex;
    gap: 12px;
  }

  .edit-btn { background: transparent; color: var(--text-secondary); }
  .edit-btn:hover { color: var(--primary); }
  .del-btn { background: transparent; color: var(--text-secondary); }
  .del-btn:hover:not(:disabled) { color: var(--danger); }
  .del-btn:disabled {
    opacity: 0.3;
    cursor: not-allowed;
  }

  .empty {
    text-align: center;
    padding: 40px;
    color: var(--text-secondary);
  }

  .loading-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 300px;
    color: var(--text-secondary);
    gap: 16px;
  }

  .loading-spinner {
    width: 40px;
    height: 40px;
    border: 3px solid rgba(99, 102, 241, 0.2);
    border-top-color: var(--primary);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  /* Modal */
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .modal {
    width: 100%;
    max-width: 500px;
    padding: 32px;
  }

  .form-group {
    margin-bottom: 20px;
  }

  .form-group label {
    display: block;
    margin-bottom: 8px;
    color: var(--text-secondary);
    font-size: 0.875rem;
  }

  .form-group input {
    width: 100%;
    padding: 10px 12px;
    background: #0f172a;
    border: 1px solid var(--border);
    color: white;
    border-radius: 6px;
    font-family: inherit;
    font-size: 1rem;
  }
  
  .form-group input:focus {
    outline: none;
    border-color: var(--primary);
  }

  .group-pill {
    padding: 3px 8px;
    border-radius: 4px;
    background: rgba(99, 102, 241, 0.15);
    color: var(--primary);
    font-size: 0.8rem;
    font-weight: 600;
  }
  
  .text-dim {
    color: var(--text-secondary);
  }

  .form-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    margin-top: 32px;
  }

  .save-btn {
    background: var(--primary);
    color: white;
    padding: 10px 24px;
  }

  .cancel-btn {
    background: transparent;
    color: var(--text-secondary);
  }
</style>
