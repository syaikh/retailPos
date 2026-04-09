<script>
  import { onMount } from 'svelte';
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

  let searchQuery = $state('');
  let showModal = $state(false);
  let editingProduct = $state(null);
  let loading = $state(false);

  // Form state
  let form = $state({
    name: '',
    sku: '',
    price: 0,
    stock: 0
  });

  onMount(fetchProducts);

  async function fetchProducts() {
    loading = true;
    try {
      const resp = await api.get('/products');
      products.set(resp.data);
    } finally {
      loading = false;
    }
  }

  function openCreate() {
    editingProduct = null;
    form = { name: '', sku: '', price: 0, stock: 0 };
    showModal = true;
  }

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

  let filtered = $derived($products.filter(p => 
    p.name.toLowerCase().includes(searchQuery.toLowerCase()) || 
    p.sku.includes(searchQuery)
  ));
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
    <div class="search-wrapper">
      <span class="icon"><Search size={18} /></span>
      <input type="text" placeholder="Cari SKU atau nama barang..." bind:value={searchQuery} />
    </div>
  </div>

  <div class="table-container premium-card">
    <table>
      <thead>
        <tr>
          <th>SKU / Barcode</th>
          <th>Nama Produk</th>
          <th>Harga</th>
          <th>Stok</th>
          <th>Aksi</th>
        </tr>
      </thead>
      <tbody>
        {#each filtered as p}
          <tr>
            <td><code>{p.sku}</code></td>
            <td><strong>{p.name}</strong></td>
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
                <button class="del-btn" onclick={() => deleteProduct(p.id)} title="Hapus">
                  <Trash2 size={18} />
                </button>
              </div>
            </td>
          </tr>
        {/each}
        {#if filtered.length === 0 && !loading}
          <tr>
            <td colspan="5" class="empty">Tidak ada produk ditemukan</td>
          </tr>
        {/if}
      </tbody>
    </table>
  </div>
</div>

{#if showModal}
  <div class="modal-overlay">
    <div class="modal premium-card">
      <h2>{editingProduct ? 'Edit Produk' : 'Tambah Produk Baru'}</h2>
      <form onsubmit={handleSubmit}>
        <div class="form-group">
          <label for="product-name">Nama Produk</label>
          <input id="product-name" type="text" bind:value={form.name} required />
        </div>
        <div class="form-group">
          <label for="product-sku">SKU / Barcode</label>
          <input id="product-sku" type="text" bind:value={form.sku} required />
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

  .search-wrapper {
    position: relative;
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
    padding-left: 40px;
  }

  code {
    background: #1e293b;
    padding: 2px 6px;
    border-radius: 4px;
    color: var(--accent);
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
  .del-btn:hover { color: var(--danger); }

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
