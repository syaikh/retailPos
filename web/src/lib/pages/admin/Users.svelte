<script>
  import { onMount } from 'svelte';
  import { apiFetch } from '$lib/api/client';
  import { toast } from '$lib/stores/toast';
  import { debounce } from '$lib/utils/debounce';

  import Badge from '$lib/components/ui/Badge.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import Pagination from '$lib/components/ui/Pagination.svelte';
  import { Search, Plus, Pencil, Trash2, User, Users, Loader2, X } from 'lucide-svelte';

  let loading = $state(true);
  let users = $state([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let roles = $state([]);
  let searchQuery = $state('');
  let showModal = $state(false);
  let showDeleteModal = $state(false);
  let selectedUser = $state(null);
  let modalMode = $state('add');
  let saving = $state(false);
  let isSearching = $state(false);
  let isInitialMount = $state(true);
  let prevSearchQuery = $state('');
  let prevOffset = $state(0);
  let prevLimit = $state(20);

  // Form State
  let form = $state({
    username: '',
    email: '',
    password: '',
    role_id: 0,
    is_active: true
  });

  const roleVariant = (r) => {
    const roleName = typeof r === 'object' ? r.name : r;
    if (roleName === 'superadmin') return 'primary';
    if (roleName === 'admin') return 'warning';
    if (roleName === 'manager') return 'info';
    if (roleName === 'cashier') return 'success';
    return 'muted';
  };

  async function fetchUsers(isSearch = false) {
    try {
      if (!isSearch) loading = true;
      const params = new URLSearchParams({
        limit: limit.toString(),
        offset: offset.toString(),
        search: searchQuery
      });
      const uRes = await apiFetch(`/api/admin/users?${params.toString()}`);
      if (uRes.ok) {
        const data = await uRes.json();
        users = data.data || [];
        total = data.total || 0;
      }
    } catch {
      toast.error('Failed to load users');
    } finally {
      if (!isSearch) loading = false;
      isSearching = false;
    }
  }

  async function fetchRoles() {
    try {
      const rRes = await apiFetch('/api/admin/roles');
      if (rRes.ok) {
        const data = await rRes.json();
        roles = data.data || [];
        if (roles.length > 0 && form.role_id === 0) {
          form.role_id = roles[0].id;
        }
      }
    } catch {
      toast.error('Failed to load roles');
    }
  }

  onMount(async () => {
    isInitialMount = true;
    await fetchRoles();
    await fetchUsers(false);
    isInitialMount = false;
  });

  // Debounced search
  const debouncedSearch = debounce(() => {
    offset = 0;
    prevOffset = 0;
    fetchUsers(true);
  }, 400);

  // Track search query changes with explicit tracking
  $effect(() => {
    // Access searchQuery to establish dependency
    const sq = searchQuery;
    
    if (isInitialMount) return;
    
    if (sq !== prevSearchQuery) {
      prevSearchQuery = sq;
      
      if (sq === '') {
        offset = 0;
        prevOffset = 0;
        isSearching = false;
        fetchUsers(false);
      } else {
        isSearching = true;
        debouncedSearch();
      }
    }
  });

  // Track pagination changes with explicit tracking
  $effect(() => {
    // Access offset and limit to establish dependency
    const off = offset;
    const lim = limit;
    
    if (isInitialMount) return;
    
    if (off !== prevOffset || lim !== prevLimit) {
      prevOffset = off;
      prevLimit = lim;
      fetchUsers(false);
    }
  });

  function handlePageChange(newOffset, newLimit) {
    offset = newOffset;
    limit = newLimit;
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
        await fetchUsers();
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
        toast.success(`User "${selectedUser.username}" removed`);
        await fetchUsers();
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

  onMount(() => {
    fetchUsers();
    fetchRoles();
  });
</script>

<div class="space-y-5">
  <!-- Header Section -->
  <div class="flex items-center justify-between mb-6">
    <div>
      <h2 class="text-2xl font-bold text-text-primary">User Management</h2>
      <p class="text-text-muted">Manage user accounts, roles, and access permissions</p>
    </div>
    <div class="flex items-center gap-2">
      <button class="btn btn-primary" onclick={openAdd}>
        <Plus size={16} /> Add User
      </button>
    </div>
  </div>

  <!-- Search -->
  <div class="card p-4">
    <div class="relative max-w-sm">
      <Search size={16} class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none" />
      <input type="text" placeholder="Search users by name or email…" class="input pl-9 pr-10" bind:value={searchQuery} />
      {#if searchQuery}
        <button
          onclick={() => searchQuery = ''}
          class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-secondary transition-colors"
          title="Clear search"
        >
          <X size={14} />
        </button>
      {/if}
    </div>
  </div>

  <!-- Table -->
  <div class="card p-0 overflow-hidden">
    <div class="px-4 py-3 border-b border-border flex items-center justify-between">
      <p class="text-sm font-semibold text-text-primary">User Accounts</p>
      {#if !loading}
        <span class="badge badge-muted">{total} users</span>
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
    {:else if users.length === 0}
      <div class="px-4 py-12 text-center">
        <div class="empty-state-icon bg-surface w-20 h-20 mx-auto">
          <Users size={32} class="text-text-muted" />
        </div>
        <p class="text-text-primary font-semibold mt-4">No users found</p>
        <p class="text-text-muted text-sm mt-1">
          {searchQuery ? `No match for "${searchQuery}"` : 'Start by adding a user'}
        </p>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table>
          <thead class="sticky top-0 bg-bg-secondary z-10 shadow-sm">
            <tr>
              <th>User</th>
              <th>Role</th>
              <th>Status</th>
              <th>Last Login</th>
              <th class="text-center">Actions</th>
            </tr>
          </thead>
          <tbody>
            {#each users as user (user.id)}
              <tr class="border-t border-border hover:bg-surface-hover/50 transition-colors">
                <td>
                  <div class="flex items-center gap-3">
                    <div class="w-9 h-9 rounded-full gradient-bg-primary flex items-center justify-center shrink-0">
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
                    {user.role?.name || (user.role_id === 1 ? 'superadmin' : user.role_id === 2 ? 'admin' : user.role_id === 3 ? 'cashier' : user.role_id === 4 ? 'manager' : 'cashier')}
                  </Badge>
                </td>
                <td>
                  <div class="flex items-center gap-2">
                    <span class="w-1.5 h-1.5 rounded-full {user.is_active !== false ? 'bg-success animate-pulse-dot' : 'bg-text-muted'}"></span>
                    <span class="text-sm text-text-secondary">{user.is_active !== false ? 'Active' : 'Inactive'}</span>
                  </div>
                </td>
                <td class="text-text-muted text-sm">
                  {user.last_login ? new Date(user.last_login).toLocaleString('en-US', { dateStyle: 'medium', timeStyle: 'medium' }) : 'Never'}
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

      <div class="p-4 bg-surface-subtle/30">
        <Pagination 
          {total} 
          {limit} 
          {offset} 
          onPageChange={handlePageChange} 
        />
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
        <label class="flex items-center gap-3 cursor-pointer select-none group">
          <div class="relative">
            <input type="checkbox" class="sr-only peer" bind:checked={form.is_active} />
            <div class="w-10 h-5 bg-surface-default border border-border rounded-full peer peer-checked:bg-primary-subtle peer-checked:border-primary/50 transition-colors"></div>
            <div class="absolute left-1 top-1 w-3 h-3 bg-text-muted rounded-full peer-checked:translate-x-5 peer-checked:bg-primary-light transition-transform shadow-sm"></div>
          </div>
          <span class="text-sm font-medium text-text-secondary group-hover:text-text-primary transition-colors">Active Account</span>
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