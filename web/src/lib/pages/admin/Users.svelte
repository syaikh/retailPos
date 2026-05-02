<script>
  import { onMount } from 'svelte';
  import { apiFetch } from '$lib/api/client';
  import { toast } from '$lib/stores/toast';
  import PageHeader from '$lib/components/ui/PageHeader.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import { Search, Plus, Pencil, Trash2, User, Users, Loader2 } from 'lucide-svelte';

  let loading = $state(true);
  let users = $state([]);
  let roles = $state([]);
  let searchQuery = $state('');
  let showModal = $state(false);
  let showDeleteModal = $state(false);
  let selectedUser = $state(null);
  let modalMode = $state('add');
  let saving = $state(false);

  // Form State
  let form = $state({
    username: '',
    email: '',
    password: '',
    role_id: 0,
    is_active: true
  });

  const filtered = $derived(
    users.filter(u =>
      !searchQuery ||
      (u.username || '').toLowerCase().includes(searchQuery.toLowerCase()) ||
      (u.email || '').toLowerCase().includes(searchQuery.toLowerCase())
    )
  );

  const roleVariant = (r) => {
    const roleName = typeof r === 'object' ? r.name : r;
    if (roleName === 'admin') return 'primary';
    if (roleName === 'manager') return 'warning';
    return 'muted';
  };

  async function fetchData() {
    try {
      loading = true;
      const [uRes, rRes] = await Promise.all([
        apiFetch('/api/admin/users'),
        apiFetch('/api/admin/roles')
      ]);

      if (uRes.ok) {
        const data = await uRes.json();
        users = data.data || [];
      }
      if (rRes.ok) {
        const data = await rRes.json();
        roles = data.data || [];
        if (roles.length > 0 && form.role_id === 0) {
          form.role_id = roles[0].id;
        }
      }
    } catch {
      toast.error('Failed to load user data');
    } finally {
      loading = false;
    }
  }

  function openAdd() {
    modalMode = 'add';
    form = { username: '', email: '', password: '', role_id: roles[0]?.id || 0, is_active: true };
    showModal = true;
  }

  function openEdit(user) {
    modalMode = 'edit';
    selectedUser = user;
    form = { 
      username: user.username, 
      email: user.email, 
      password: '', // Don't show password
      role_id: user.role_id, 
      is_active: user.is_active 
    };
    showModal = true;
  }

  async function saveUser() {
    if (!form.username || !form.email || (modalMode === 'add' && !form.password)) {
      toast.error('Please fill all required fields');
      return;
    }

    try {
      saving = true;
      const method = modalMode === 'add' ? 'POST' : 'PUT';
      const url = modalMode === 'add' ? '/api/admin/users' : `/api/admin/users/${selectedUser.id}`;
      
      const r = await apiFetch(url, {
        method,
        body: JSON.stringify(form)
      });

      if (r.ok) {
        toast.success(modalMode === 'add' ? 'User created' : 'User updated');
        showModal = false;
        await fetchData();
      } else {
        const err = await r.json();
        toast.error(err.error || 'Failed to save user');
      }
    } catch {
      toast.error('Network error');
    } finally {
      saving = false;
    }
  }

  async function confirmDelete() {
    if (!selectedUser) return;
    try {
      const r = await apiFetch(`/api/admin/users/${selectedUser.id}`, { method: 'DELETE' });
      if (r.ok) {
        users = users.filter(u => u.id !== selectedUser.id);
        toast.success(`User "${selectedUser.username}" removed`);
      } else {
        toast.error('Failed to delete user');
      }
    } catch {
      toast.error('Failed to delete user');
    } finally {
      showDeleteModal = false;
      selectedUser = null;
    }
  }

  onMount(fetchData);
</script>

<div class="space-y-5">
  <PageHeader title="Users" subtitle="Manage user accounts and access">
    {#snippet actions()}
      <button class="btn btn-primary" onclick={openAdd}>
        <Plus size={15} /> Add User
      </button>
    {/snippet}
  </PageHeader>

  <!-- Search -->
  <div class="card p-4">
    <div class="relative max-w-sm">
      <Search size={16} class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none" />
      <input type="text" placeholder="Search users by name or email…" class="input pl-9" bind:value={searchQuery} />
    </div>
  </div>

  <!-- Table -->
  <div class="card p-0 overflow-hidden">
    <div class="px-4 py-3 border-b border-border flex items-center justify-between">
      <p class="text-sm font-semibold text-text-primary">User Accounts</p>
      {#if !loading}
        <span class="badge badge-muted">{filtered.length} users</span>
      {/if}
    </div>

    {#if loading}
      <div class="divide-y divide-border">
        {#each { length: 5 } as _}
          <div class="flex items-center gap-4 px-4 py-3.5">
            <Skeleton width="w-9" height="h-9" rounded="rounded-full" />
            <div class="flex flex-col gap-1.5">
              <Skeleton width="w-32" height="h-3.5" />
              <Skeleton width="w-44" height="h-3" />
            </div>
            <Skeleton width="w-16" height="h-6" rounded="rounded-full" class="ml-auto" />
            <Skeleton width="w-20" height="h-6" rounded="rounded-full" />
            <Skeleton width="w-16" height="h-8" rounded="rounded-xl" />
          </div>
        {/each}
      </div>
    {:else if filtered.length === 0}
      <div class="empty-state py-16">
        <div class="empty-state-icon bg-surface w-20 h-20">
          <Users size={32} class="text-text-muted" />
        </div>
        <p class="text-text-primary font-semibold">No users found</p>
        <p class="text-text-muted text-sm mt-1">
          {searchQuery ? `No match for "${searchQuery}"` : 'Start by adding a user'}
        </p>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table>
          <thead>
            <tr>
              <th>User</th>
              <th>Role</th>
              <th>Status</th>
              <th>Last Login</th>
              <th class="text-center">Actions</th>
            </tr>
          </thead>
          <tbody>
            {#each filtered as user (user.id)}
              <tr>
                <td>
                  <div class="flex items-center gap-3">
                    <div class="w-9 h-9 rounded-full gradient-bg-primary flex items-center justify-center flex-shrink-0">
                      <User size={14} class="text-white" />
                    </div>
                    <div>
                      <p class="font-medium text-text-primary">{user.username}</p>
                      <p class="text-xs text-text-muted">{user.email || '—'}</p>
                    </div>
                  </div>
                </td>
                <td>
                  <Badge variant={roleVariant(user.role)}>
                    {user.role?.name || user.role || 'cashier'}
                  </Badge>
                </td>
                <td>
                  <div class="flex items-center gap-2">
                    <span class="w-1.5 h-1.5 rounded-full {user.is_active !== false ? 'bg-success animate-pulse-dot' : 'bg-text-muted'}"></span>
                    <span class="text-sm text-text-secondary">{user.is_active !== false ? 'Active' : 'Inactive'}</span>
                  </div>
                </td>
                <td class="text-text-muted text-sm">
                  {user.last_login ? new Date(user.last_login).toLocaleString('id-ID') : 'Never'}
                </td>
                <td>
                  <div class="flex items-center justify-center gap-2">
                    <button 
                      class="btn-icon btn-ghost text-text-muted hover:text-primary-light" 
                      title="Edit"
                      onclick={() => openEdit(user)}
                    >
                      <Pencil size={14} />
                    </button>
                    <button
                      class="btn-icon btn-ghost text-text-muted hover:text-danger hover:bg-danger-subtle"
                      onclick={() => { selectedUser = user; showDeleteModal = true; }}
                      title="Delete"
                    >
                      <Trash2 size={14} />
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
</div>

<!-- Add/Edit User Modal -->
<Modal bind:open={showModal} title={modalMode === 'add' ? 'Add New User' : 'Edit User'} size="md">
  <form onsubmit={(e) => { e.preventDefault(); saveUser(); }} class="space-y-4">
    <div>
      <label for="usr-username" class="block text-sm font-medium text-text-secondary mb-2">Username</label>
      <input id="usr-username" type="text" placeholder="johndoe" class="input" bind:value={form.username} required />
    </div>
    <div>
      <label for="usr-email" class="block text-sm font-medium text-text-secondary mb-2">Email</label>
      <input id="usr-email" type="email" placeholder="john@example.com" class="input" bind:value={form.email} required />
    </div>
    <div class="grid grid-cols-2 gap-4">
      <div>
        <label for="usr-role" class="block text-sm font-medium text-text-secondary mb-2">Role</label>
        <select id="usr-role" class="select" bind:value={form.role_id}>
          {#each roles as role}
            <option value={role.id}>{role.name}</option>
          {/each}
        </select>
      </div>
      <div class="flex items-end pb-2">
        <label class="flex items-center gap-3 cursor-pointer select-none">
          <input type="checkbox" class="w-4 h-4 accent-primary rounded" bind:checked={form.is_active} />
          <span class="text-sm font-medium text-text-secondary">Active Account</span>
        </label>
      </div>
    </div>
    <div>
      <label for="usr-password" class="block text-sm font-medium text-text-secondary mb-2">
        {modalMode === 'add' ? 'Password' : 'New Password (optional)'}
      </label>
      <input id="usr-password" type="password" placeholder="••••••••" class="input" bind:value={form.password} required={modalMode === 'add'} minlength="6" />
    </div>
  </form>
  {#snippet footer()}
    <button class="btn btn-secondary" onclick={() => showModal = false} disabled={saving}>Cancel</button>
    <button class="btn btn-primary min-w-32" onclick={saveUser} disabled={saving}>
      {#if saving}
        <Loader2 size={16} class="animate-spin" /> Saving...
      {:else}
        {modalMode === 'add' ? 'Create User' : 'Save Changes'}
      {/if}
    </button>
  {/snippet}
</Modal>

<!-- Delete Confirm Modal -->
<Modal bind:open={showDeleteModal} title="Delete User" size="sm">
  <div class="text-center py-2">
    <div class="w-14 h-14 rounded-2xl bg-danger-subtle flex items-center justify-center mx-auto mb-4">
      <Trash2 size={24} class="text-danger" />
    </div>
    <p class="text-text-primary font-semibold mb-1">Delete "{selectedUser?.username}"?</p>
    <p class="text-text-muted text-sm">This will permanently remove the account and all associated access.</p>
  </div>
  {#snippet footer()}
    <button class="btn btn-secondary" onclick={() => showDeleteModal = false}>Cancel</button>
    <button class="btn btn-danger" onclick={confirmDelete}>Delete</button>
  {/snippet}
</Modal>