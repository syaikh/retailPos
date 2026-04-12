<script>
  import { onMount } from 'svelte';
  import api from '$lib/api.js';
  import { Plus, Edit, Trash2, Tags, Search } from 'lucide-svelte';
  import Pagination from '$lib/components/Pagination.svelte';

  let groups = $state([]);
  let showModal = $state(false);
  let editingGroup = $state(null);
  let loading = $state(false);

  let limit = $state(10);
  let offset = $state(0);
  let total = $state(0);
  let searchQuery = $state('');
  let sortField = $state('id');
  let sortDir = $state('asc');

  let form = $state({
    name: '',
    description: ''
  });

  onMount(fetchGroups);

  async function fetchGroups() {
    loading = true;
    try {
      const q = searchQuery.trim();
      const searchParam = q.length > 3 ? `&search=${q}` : '';
      const url = `/product-groups?limit=${limit}&offset=${offset}&sortBy=${sortField}&sortDir=${sortDir}${searchParam}`;
      const resp = await api.get(url);
      const { data, total: totalCount } = resp.data;
      groups = Array.isArray(data) ? data : [];
      total = totalCount;
    } catch (e) {
      console.error('Failed to fetch groups:', e);
      groups = [];
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    fetchGroups();
  });

  function handlePageChange(newOffset, newLimit) {
    if (newLimit !== undefined) limit = newLimit;
    offset = newOffset;
  }

  function openCreate() {
    editingGroup = null;
    form = { name: '', description: '' };
    showModal = true;
  }

  function openEdit(g) {
    editingGroup = g;
    form = { ...g };
    showModal = true;
  }

  async function handleSubmit(event) {
    event.preventDefault();
    try {
      if (editingGroup) {
        await api.put(`/product-groups/${editingGroup.id}`, form);
      } else {
        await api.post('/product-groups', form);
      }
      showModal = false;
      fetchGroups();
    } catch (e) {
      alert(e.response?.data?.error || 'Operation failed');
    }
  }

  async function deleteGroup(id) {
    if (!confirm('Yakin ingin menghapus kategori ini?')) return;
    try {
      await api.delete(`/product-groups/${id}`);
      fetchGroups();
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
    offset = 0;
  }

  let displayGroups = $derived(groups);
</script>

<div class="groups-container">
  <div class="header">
    <div class="title">
      <Tags size={32} color="var(--primary)" />
      <h1>Manajemen Kategori Barang</h1>
    </div>
    <button class="add-btn" onclick={openCreate}>
      <Plus size={20} />
      Tambah Kategori
    </button>
  </div>

  <div class="actions premium-card glass">
    <div class="search-wrapper">
      <span class="icon"><Search size={18} /></span>
      <input
        type="text"
        placeholder="Cari nama kategori..."
        bind:value={searchQuery}
        oninput={() => offset = 0}
      />
      {#if searchQuery.trim().length > 0 && searchQuery.trim().length <= 3}
        <div class="search-warning">Minimal 4 karakter</div>
      {/if}
    </div>
  </div>

  <div class="table-container premium-card">
    <table>
      <thead>
        <tr>
          <th onclick={() => handleSort('name')} class="sortable">
            Nama Kategori {#if sortField === 'name'}<span class="sort-icon">{sortDir === 'asc' ? '▲' : '▼'}</span>{/if}
          </th>
          <th onclick={() => handleSort('description')} class="sortable">
            Deskripsi {#if sortField === 'description'}<span class="sort-icon">{sortDir === 'asc' ? '▲' : '▼'}</span>{/if}
          </th>
          <th>Aksi</th>
        </tr>
      </thead>
      <tbody>
        {#each displayGroups as g}
          <tr>
            <td>
              <a href="/inventory?group={g.id}" class="group-link">
                <strong>{g.name}</strong>
              </a>
            </td>
            <td>{g.description || '-'}</td>
            <td>
              <div class="row-actions">
                <button class="edit-btn" onclick={() => openEdit(g)} title="Edit">
                  <Edit size={18} />
                </button>
                <button 
                  class="del-btn" 
                  onclick={() => deleteGroup(g.id)} 
                  disabled={g.product_count > 0}
                  title={g.product_count > 0 ? 'Hapus dibatasi: Kategori masih memiliki produk' : 'Hapus Kategori'}
                >
                  <Trash2 size={18} />
                </button>
              </div>
            </td>
          </tr>
        {/each}
        {#if displayGroups.length === 0 && !loading}
          <tr>
            <td colspan="3" class="empty">Belum ada kategori yang ditemukan.</td>
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
    onkeydown={(e) => { if (e.key === 'Escape' || e.key === 'Enter') showModal = false; }}
  >
    <div class="modal premium-card" role="presentation" onclick={(e) => e.stopPropagation()}>
      <h2>{editingGroup ? 'Edit Kategori' : 'Tambah Kategori Baru'}</h2>
      <form onsubmit={handleSubmit}>
        <div class="form-group">
          <label for="group-name">Nama Kategori</label>
          <input id="group-name" type="text" bind:value={form.name} required />
        </div>
        <div class="form-group">
          <label for="group-desc">Deskripsi</label>
          <textarea id="group-desc" bind:value={form.description} rows="3"></textarea>
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
  .groups-container {
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .actions {
    padding: 16px;
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
    padding: 10px 10px 10px 40px;
    background: #0f172a;
    border: 1px solid var(--border);
    border-radius: 8px;
    color: white;
    font-size: 1rem;
    font-family: inherit;
    outline: none;
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
    border-radius: 8px;
    font-weight: 600;
  }
  
  .add-btn:hover { background: #4f46e5; }

  .group-link {
    color: var(--primary);
    text-decoration: none;
    transition: color 0.2s;
  }

  .group-link:hover {
    color: #818cf8;
    text-decoration: underline;
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

  .form-group input, .form-group textarea {
    width: 100%;
    background: #0f172a;
    border: 1px solid var(--border);
    color: white;
    padding: 10px 12px;
    border-radius: 6px;
    font-family: inherit;
  }
  
  .form-group textarea {
    resize: vertical;
  }

  .form-group input:focus, .form-group textarea:focus {
    outline: none;
    border-color: var(--primary);
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
    border-radius: 8px;
  }

  .cancel-btn {
    background: transparent;
    color: var(--text-secondary);
  }
  
  .cancel-btn:hover { color: white; }
</style>
