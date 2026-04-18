<script lang="ts">
  import { onMount } from 'svelte';
  import { getRoles, getPermissions, createRole, updateRolePermissions, deleteRole, type Role, type Permission } from '$lib/api/admin';
  import { Shield, Loader2, Plus, Trash2, Edit } from 'lucide-svelte';

  let roles = $state<Role[]>([]);
  let permissions = $state<Permission[]>([]);
  let loading = $state(true);
  let showModal = $state(false);
  let editingRole = $state<Role | null>(null);
  let saving = $state(false);
  let message = $state('');

  let formName = $state('');
  let formDescription = $state('');
  let formPermissions = $state<number[]>([]);

  let permCategories = $derived(
    permissions.reduce((acc, p) => {
      if (!acc[p.category]) acc[p.category] = [];
      acc[p.category].push(p);
      return acc;
    }, {} as Record<string, Permission[]>)
  );

  onMount(async () => {
    try {
      roles = await getRoles();
      permissions = await getPermissions();
    } catch (e) {
      message = 'Gagal memuat data';
    } finally {
      loading = false;
    }
  });

  function openCreateModal() {
    editingRole = null;
    formName = '';
    formDescription = '';
    formPermissions = [];
    message = '';
    showModal = true;
  }

  function openEditModal(role: Role) {
    editingRole = role;
    formName = role.name;
    formDescription = role.description;
    formPermissions = permissions
      .filter(p => role.permissions.includes(p.code))
      .map(p => p.id);
    message = '';
    showModal = true;
  }

  function closeModal() {
    showModal = false;
    editingRole = null;
  }

  function togglePermission(permId: number) {
    if (formPermissions.includes(permId)) {
      formPermissions = formPermissions.filter(id => id !== permId);
    } else {
      formPermissions = [...formPermissions, permId];
    }
  }

  async function save() {
    if (!formName || formPermissions.length === 0) {
      message = 'Nama dan minimal 1 permission wajib diisi';
      return;
    }
    saving = true;
    try {
      if (editingRole) {
        await updateRolePermissions(editingRole.id, formPermissions);
      } else {
        await createRole(formName, formDescription, formPermissions);
      }
      roles = await getRoles();
      message = 'Berhasil disimpan';
      setTimeout(closeModal, 1000);
    } catch (e) {
      message = 'Gagal menyimpan';
    } finally {
      saving = false;
    }
  }

  async function handleDelete(role: Role) {
    if (!confirm(`Hapus role "${role.name}"?`)) return;
    try {
      await deleteRole(role.id);
      roles = await getRoles();
    } catch (e) {
      alert('Gagal menghapus role');
    }
  }
</script>

<div class="page">
  <div class="header">
    <Shield size={28} color="var(--primary)" />
    <h1>Kelola Role</h1>
    <button class="btn-add" onclick={openCreateModal}>
      <Plus size={18} />
      Tambah Role
    </button>
  </div>

  {#if loading}
    <div class="loading">
      <Loader2 size={32} class="spin" />
    </div>
  {:else}
    <div class="table-container">
      <table>
        <thead>
          <tr>
            <th>Nama Role</th>
            <th>Deskripsi</th>
            <th>Permissions</th>
            <th>Aksi</th>
          </tr>
        </thead>
        <tbody>
          {#each roles as role}
            <tr>
              <td>
                <span class="badge" class:system={role.is_system}>
                  {role.name}
                </span>
              </td>
              <td>{role.description || '-'}</td>
              <td>{role.permissions.length} permissions</td>
              <td>
                <div class="actions">
                  <button class="btn-icon" onclick={() => openEditModal(role)} disabled={role.is_system}>
                    <Edit size={16} />
                  </button>
                  <button class="btn-icon btn-delete" onclick={() => handleDelete(role)} disabled={role.is_system}>
                    <Trash2 size={16} />
                  </button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

{#if showModal}
  <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
  <div class="modal-overlay" onclick={closeModal} onkeydown={(e) => e.key === 'Escape' && closeModal()} role="dialog" aria-modal="true" tabindex="-1">
    <div class="modal">

      {#if !editingRole}
        <div class="form-group">
          <label for="name">Nama Role</label>
          <input type="text" id="name" bind:value={formName} placeholder="Contoh: manager" />
        </div>

        <div class="form-group">
          <label for="desc">Deskripsi</label>
          <textarea id="desc" bind:value={formDescription} placeholder="Deskripsi role..."></textarea>
        </div>
      {/if}

      <div class="form-group">
        <div class="perm-header">Permissions</div>
        <div class="perm-categories">
          {#each Object.entries(permCategories) as [category, perms]}
            <div class="perm-category">
              <h4>{category}</h4>
              <div class="perm-list">
                {#each perms as perm}
                  <label class="perm-checkbox">
                    <input
                      type="checkbox"
                      checked={formPermissions.includes(perm.id)}
                      onchange={() => togglePermission(perm.id)}
                    />
                    <span>{perm.code}</span>
                  </label>
                {/each}
              </div>
            </div>
          {/each}
        </div>
      </div>

      {#if message}
        <div class="message" class:error={!message.includes('Berhasil')}>{message}</div>
      {/if}

      <div class="modal-actions">
        <button class="btn-cancel" onclick={closeModal}>Batal</button>
        <button class="btn-save" onclick={save} disabled={saving}>
          {saving ? 'Menyimpan...' : 'Simpan'}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .page {
    padding: 24px;
  }

  .header {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 24px;
  }

  .header h1 {
    font-size: 1.5rem;
    font-weight: 600;
    flex: 1;
  }

  .btn-add {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 16px;
    background: var(--primary);
    color: white;
    border: none;
    border-radius: 6px;
    cursor: pointer;
  }

  .loading {
    display: flex;
    justify-content: center;
    padding: 48px;
  }

  .table-container {
    background: var(--bg-surface);
    border-radius: 8px;
    overflow: hidden;
  }

  table {
    width: 100%;
    border-collapse: collapse;
  }

  th, td {
    padding: 12px 16px;
    text-align: left;
    border-bottom: 1px solid var(--border);
  }

  th {
    background: var(--bg-hover);
    font-weight: 600;
  }

  .badge {
    padding: 4px 8px;
    border-radius: 4px;
    font-size: 0.75rem;
    background: var(--bg-hover);
  }

  .badge.system {
    background: var(--primary);
    color: white;
  }

  .actions {
    display: flex;
    gap: 8px;
  }

  .btn-icon {
    padding: 6px;
    background: transparent;
    border: 1px solid var(--border);
    border-radius: 4px;
    cursor: pointer;
    color: var(--text-primary);
  }

  .btn-icon:disabled {
    opacity: 0.3;
    cursor: not-allowed;
  }

  .btn-delete:hover {
    background: rgba(239, 68, 68, 0.1);
    color: var(--danger);
  }

  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
    overflow-y: auto;
    padding: 24px;
  }

  .modal {
    background: var(--bg-surface);
    padding: 24px;
    border-radius: 8px;
    width: 100%;
    max-width: 600px;
    max-height: 90vh;
    overflow-y: auto;
  }

  .form-group {
    margin-bottom: 16px;
  }

  .form-group label, .perm-header {
    display: block;
    margin-bottom: 8px;
    font-size: 0.875rem;
    color: var(--text-secondary);
  }

  input[type="text"], textarea {
    width: 100%;
    padding: 10px;
    border: 1px solid var(--border);
    border-radius: 6px;
    background: var(--bg-primary);
    color: var(--text-primary);
  }

  textarea {
    min-height: 60px;
  }

  .perm-categories {
    max-height: 300px;
    overflow-y: auto;
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 12px;
  }

  .perm-category {
    margin-bottom: 16px;
  }

  .perm-category:last-child {
    margin-bottom: 0;
  }

  .perm-category h4 {
    font-size: 0.875rem;
    color: var(--text-secondary);
    margin-bottom: 8px;
    text-transform: capitalize;
  }

  .perm-list {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .perm-checkbox {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 0.75rem;
    cursor: pointer;
  }

  .perm-checkbox input {
    width: auto;
  }

  .message {
    padding: 8px;
    background: rgba(34, 197, 94, 0.1);
    color: #22c55e;
    border-radius: 4px;
    margin-bottom: 16px;
  }

  .message.error {
    background: rgba(239, 68, 68, 0.1);
    color: var(--danger);
  }

  .modal-actions {
    display: flex;
    gap: 12px;
    justify-content: flex-end;
    margin-top: 16px;
  }

  .btn-cancel {
    padding: 10px 16px;
    background: transparent;
    color: var(--text-primary);
    border: 1px solid var(--border);
    border-radius: 6px;
    cursor: pointer;
  }

  .btn-save {
    padding: 10px 16px;
    background: var(--primary);
    color: white;
    border: none;
    border-radius: 6px;
    cursor: pointer;
  }

  .btn-save:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
