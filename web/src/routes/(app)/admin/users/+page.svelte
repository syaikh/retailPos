<script lang="ts">
  import { onMount } from 'svelte';
  import { getUsers, getRoles, updateUserRole, type User, type Role } from '$lib/api/admin';
  import { Users, Shield, Loader2 } from 'lucide-svelte';

  let users = $state<User[]>([]);
  let roles = $state<Role[]>([]);
  let loading = $state(true);
  let editingUser = $state<User | null>(null);
  let selectedRoleId = $state<number>(0);
  let saving = $state(false);
  let message = $state('');

  onMount(async () => {
    try {
      users = await getUsers();
      roles = await getRoles();
    } catch (e) {
      message = 'Gagal memuat data';
    } finally {
      loading = false;
    }
  });

  function openEditModal(user: User) {
    editingUser = user;
    selectedRoleId = user.role_id;
    message = '';
  }

  function closeModal() {
    editingUser = null;
    selectedRoleId = 0;
    message = '';
  }

  async function saveRole() {
    if (!editingUser) return;
    saving = true;
    try {
      await updateUserRole(editingUser.id, selectedRoleId);
      users = await getUsers();
      message = 'Role berhasil diperbarui';
      setTimeout(closeModal, 1000);
    } catch (e) {
      message = 'Gagal memperbarui role';
    } finally {
      saving = false;
    }
  }
</script>

<div class="page">
  <div class="header">
    <Users size={28} color="var(--primary)" />
    <h1>Kelola Pengguna</h1>
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
            <th>Username</th>
            <th>Role Saat Ini</th>
            <th>Aksi</th>
          </tr>
        </thead>
        <tbody>
          {#each users as user}
            <tr>
              <td>{user.username}</td>
              <td>
                <span class="badge" class:admin={user.role_name === 'admin'}>
                  {user.role_name}
                </span>
              </td>
              <td>
                {#if !user.is_system_role}
                  <button class="btn-edit" onclick={() => openEditModal(user)}>
                    Ubah Role
                  </button>
                {:else}
                  <span class="text-muted">System</span>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

{#if editingUser}
  <!-- svelte-ignore a11y_no_static_element_interactions a11y_click_events_have_key_events -->
  <div class="modal-overlay" onclick={closeModal} onkeydown={(e) => e.key === 'Escape' && closeModal()} role="dialog" aria-modal="true" tabindex="-1">
    <!-- svelte-ignore a11y_no_static_element_interactions a11y_click_events_have_key_events -->
    <div class="modal" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()}>
      
      <div class="form-group">
        <label for="role">Pilih Role</label>
        <select id="role" bind:value={selectedRoleId}>
          {#each roles as role}
            <option value={role.id}>{role.name} - {role.description}</option>
          {/each}
        </select>
      </div>

      {#if message}
        <div class="message">{message}</div>
      {/if}

      <div class="modal-actions">
        <button class="btn-cancel" onclick={closeModal}>Batal</button>
        <button class="btn-save" onclick={saveRole} disabled={saving}>
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

  .badge.admin {
    background: var(--primary);
    color: white;
  }

  .btn-edit {
    padding: 6px 12px;
    background: var(--primary);
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
  }

  .btn-edit:hover {
    background: var(--primary-hover);
  }

  .text-muted {
    color: var(--text-secondary);
    font-size: 0.875rem;
  }

  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
  }

  .modal {
    background: var(--bg-surface);
    padding: 24px;
    border-radius: 8px;
    width: 100%;
    max-width: 400px;
  }

  .form-group {
    margin-bottom: 16px;
  }

  .form-group label {
    display: block;
    margin-bottom: 8px;
    font-size: 0.875rem;
    color: var(--text-secondary);
  }

  select {
    width: 100%;
    padding: 10px;
    border: 1px solid var(--border);
    border-radius: 6px;
    background: var(--bg-primary);
    color: var(--text-primary);
  }

  .message {
    padding: 8px;
    background: rgba(34, 197, 94, 0.1);
    color: #22c55e;
    border-radius: 4px;
    margin-bottom: 16px;
  }

  .modal-actions {
    display: flex;
    gap: 12px;
    justify-content: flex-end;
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