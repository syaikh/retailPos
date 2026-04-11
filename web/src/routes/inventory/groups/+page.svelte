<script>
  import { onMount } from 'svelte';
  import api from '$lib/api.js';
  import { Plus, Edit, Trash2, Tags } from 'lucide-svelte';

  let groups = $state([]);
  let showModal = $state(false);
  let editingGroup = $state(null);
  let loading = $state(false);

  let form = $state({
    name: '',
    description: ''
  });

  let sortField = $state('');
  let sortDir = $state('asc');

  onMount(fetchGroups);

  async function fetchGroups() {
    loading = true;
    try {
      const resp = await api.get('/product-groups');
      groups = Array.isArray(resp.data) ? resp.data : [];
    } catch (e) {
      console.error('Failed to fetch groups:', e);
      groups = [];
    } finally {
      loading = false;
    }
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
  }

  let sortedGroups = $derived.by(() => {
    let result = [...groups];
    if (sortField) {
      result.sort((a, b) => {
        let valA = a[sortField] || '';
        let valB = b[sortField] || '';
        if (typeof valA === 'string' && typeof valB === 'string') {
          return sortDir === 'asc' ? valA.localeCompare(valB) : valB.localeCompare(valA);
        } else {
          return sortDir === 'asc' ? ((valA||0) - (valB||0)) : ((valB||0) - (valA||0));
        }
      });
    }
    return result;
  });
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

  <div class="table-container premium-card">
    <table>
      <thead>
        <tr>
          <th onclick={() => handleSort('id')} class="sortable">
            ID {#if sortField === 'id'}<span class="sort-icon">{sortDir === 'asc' ? '▲' : '▼'}</span>{/if}
          </th>
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
        {#each sortedGroups as g}
          <tr>
            <td><code>{g.id}</code></td>
            <td><strong>{g.name}</strong></td>
            <td>{g.description || '-'}</td>
            <td>
              <div class="row-actions">
                <button class="edit-btn" onclick={() => openEdit(g)} title="Edit">
                  <Edit size={18} />
                </button>
                <button class="del-btn" onclick={() => deleteGroup(g.id)} title="Hapus">
                  <Trash2 size={18} />
                </button>
              </div>
            </td>
          </tr>
        {/each}
        {#if groups.length === 0 && !loading}
          <tr>
            <td colspan="4" class="empty">Belum ada kategori yang dibuat.</td>
          </tr>
        {/if}
      </tbody>
    </table>
  </div>
</div>

{#if showModal}
  <div class="modal-overlay">
    <div class="modal premium-card">
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
