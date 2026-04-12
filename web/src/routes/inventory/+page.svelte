<script>
  import { onMount, tick } from 'svelte';
  import { page } from '$app/stores';
  import { products } from '$lib/stores.js';
  import api from '$lib/api.js';
  import { 
    Plus, 
    Search, 
    Edit, 
    Trash2, 
    Package,
    AlertCircle
  } from 'lucide-svelte';
  import SearchableSelect from '$lib/components/SearchableSelect.svelte';
  import Pagination from '$lib/components/Pagination.svelte';

  let searchQuery = $state('');
  let showModal = $state(false);
  let editingProduct = $state(null);
  let loading = $state(false);
  let searchInput = $state(null);
  let hideEmptyStock = $state(true); // Default disembunyikan sesuai keluhan user
  let groups = $state([]);

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

  function autofocus(node) {
    requestAnimationFrame(() => {
      node.focus({ preventScroll: true });
    });
    return {
      destroy() {
        // no cleanup needed
      }
    };
  }

  // Form state
  let form = $state({
    name: '',
    sku: '',
    price: 0,
    stock: 0,
    group_id: null
  });

  onMount(() => {
    fetchGroups();
  });

  $effect(() => {
    if (!showModal && searchInput) {
      requestAnimationFrame(() => {
        searchInput.focus({ preventScroll: true });
      });
    }
  });

  async function fetchProducts() {
    loading = true;
    try {
      const q = searchQuery.trim();
      // Hanya kirim parameter search jika > 3 karakter
      const searchParam = q.length > 3 ? `&search=${q}` : '';
      const gParam = selectedGroupFilter === 'all' ? '' : `&group_id=${selectedGroupFilter}`;
      const url = `/products?limit=${limit}&offset=${offset}&sortBy=${sortField}&sortDir=${sortDir}${searchParam}${gParam}`;
      const resp = await api.get(url);
      const { data, total: totalCount } = resp.data;
      products.set(Array.isArray(data) ? data : []);
      total = totalCount;
    } catch (e) {
      console.error('Failed to fetch products:', e);
      products.set([]);
    } finally {
      loading = false;
    }
  }

  // Reload data when filters/paging changes
  $effect(() => {
    fetchProducts();
  });

  function handlePageChange(newOffset, newLimit) {
    if (newLimit !== undefined) limit = newLimit;
    offset = newOffset;
  }

  async function fetchGroups() {
    try {
      const resp = await api.get('/product-groups?limit=1000');
      groups = Array.isArray(resp.data.data) ? resp.data.data : [];
    } catch (e) {
      console.error('Failed to fetch groups:', e);
      groups = [];
    }
  }

  function openCreate() {
    editingProduct = null;
    form = { name: '', sku: '', price: 0, stock: 0, group_id: null };
    showModal = true;
  }

  /**
   * @param {{ id: number; name: string; sku: string; price: number; stock: number; group_id: number | null; }} p
   */
  function openEdit(p) {
    editingProduct = p;
    form = { ...p };
    showModal = true;
  }

  async function handleSubmit(event) {
    event.preventDefault();
    try {
      if (editingProduct) {
        await api.put(`/products/${editingProduct.id}`, form);
      } else {
        await api.post('/products', form);
      }
      showModal = false;
      fetchProducts();
    } catch (e) {
      alert(e.response?.data?.error || 'Operation failed');
    }
  }

  async function deleteProduct(id) {
    if (!confirm('Yakin ingin menghapus produk ini?')) return;
    try {
      await api.delete(`/products/${id}`);
      fetchProducts();
    } catch (e) {
      alert(e.response?.data?.error || 'Delete failed');
    }
  }

  function handleSort(field) {
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
</script>

<div class="inventory-container">
  <div class="header">
    <div class="title">
      <Package size={32} color="var(--primary)" />
      <h1>Manajemen Barang</h1>
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
          placeholder="Cari SKU atau nama barang..."
          bind:value={searchQuery}
          bind:this={searchInput}
          use:autofocus
        />
        {#if searchQuery.trim().length > 0 && searchQuery.trim().length <= 3}
          <div class="search-warning">Minimal 4 karakter</div>
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
  </div>

  <div class="table-container premium-card">
    <table>
      <thead>
        <tr>
          <th onclick={() => handleSort('sku')} class="sortable">
            SKU / Barcode {#if sortField === 'sku'}<span class="sort-icon">{sortDir === 'asc' ? '▲' : '▼'}</span>{/if}
          </th>
          <th onclick={() => handleSort('name')} class="sortable">
            Nama Produk {#if sortField === 'name'}<span class="sort-icon">{sortDir === 'asc' ? '▲' : '▼'}</span>{/if}
          </th>
          <th onclick={() => handleSort('group_id')} class="sortable">
            Kategori {#if sortField === 'group_id'}<span class="sort-icon">{sortDir === 'asc' ? '▲' : '▼'}</span>{/if}
          </th>
          <th onclick={() => handleSort('price')} class="sortable">
            Harga {#if sortField === 'price'}<span class="sort-icon">{sortDir === 'asc' ? '▲' : '▼'}</span>{/if}
          </th>
          <th onclick={() => handleSort('stock')} class="sortable">
            Stok {#if sortField === 'stock'}<span class="sort-icon">{sortDir === 'asc' ? '▲' : '▼'}</span>{/if}
          </th>
          <th>Aksi</th>
        </tr>
      </thead>
      <tbody>
        {#each displayProducts as p}
          <tr>
            <td><code>{p.sku}</code></td>
            <td><strong>{p.name}</strong></td>
            <td>
              {#if p.group_id}
                <span class="group-pill">
                  {groups.find(g => g.id === p.group_id)?.name || p.group_id}
                </span>
              {:else}
                <span class="text-dim">-</span>
              {/if}
            </td>
            <td>Rp {p.price.toLocaleString()}</td>
            <td>
              <span class="stock-pill" class:low={p.stock < 10}>
                {p.stock} units
              </span>
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
        {#if displayProducts.length === 0 && !loading}
          <tr>
            <td colspan="6" class="empty">Tidak ada produk ditemukan</td>
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
        <div class="form-group">
          <label for="product-sku">SKU / Barcode</label>
          <input id="product-sku" type="text" bind:value={form.sku} required use:autofocus />
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

  .search-warning {
    position: absolute;
    top: 100%;
    left: 40px;
    font-size: 0.75rem;
    color: var(--accent);
    margin-top: 4px;
    font-weight: 500;
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
  }

  th.sortable {
    cursor: pointer;
    user-select: none;
    transition: color 0.2s;
  }
  
  th.sortable:hover {
    color: white;
  }
  
  .sort-icon {
    font-size: 0.8em;
    margin-left: 4px;
  }

  .stock-pill {
    padding: 4px 10px;
    border-radius: 99px;
    background: rgba(16, 185, 129, 0.1);
    color: var(--success);
    font-size: 0.875rem;
  }

  .stock-pill.low {
    background: rgba(239, 68, 68, 0.1);
    color: var(--danger);
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
