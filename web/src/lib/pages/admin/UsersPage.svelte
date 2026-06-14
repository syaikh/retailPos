<script>
  import { onMount } from 'svelte';
  import apiClient from '$lib/api/client';
  import { toast } from '$lib/stores/toast';
  import { debounce } from '$lib/utils/debounce';
  import { auth } from '$lib/stores/auth';

  import Badge from '$lib/components/ui/Badge.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import Pagination from '$lib/components/ui/Pagination.svelte';
  import { Search, Plus, Pencil, Trash2, User, Users, Loader2, X, Shield, ArrowUpDown, ChevronDown, SlidersHorizontal } from 'lucide-svelte';

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

  let sortBy = $state('username');
  let sortDir = $state('asc');
  let filterRole = $state('all');
  let filterStatus = $state('all');

  let userRole = $derived(
    $auth.user?.role?.name ||
    ($auth.user?.role && typeof $auth.user?.role === 'object' ? $auth.user.role.name : $auth.user?.role) ||
    ''
  );
  let canCreate = $derived(['superadmin', 'admin'].includes(userRole));
  let canEdit = $derived(['superadmin', 'admin'].includes(userRole));
  let canDelete = $derived(userRole === 'superadmin');
  let canView = $derived(userRole !== 'cashier' && userRole !== '');
  let canEditSuperadmin = $derived(userRole === 'superadmin');
  let usernameHasInvalidChars = $derived(form.username.length > 0 && !/^[a-zA-Z0-9]+$/.test(form.username));

  let currentUserID = $derived($auth.user?.id || 0);

  let prevSearchQuery = '';
  let prevOffset = 0;
  let prevLimit = 20;
  let prevSortBy = 'username';
  let prevSortDir = 'asc';
  let prevFilterRole = 'all';
  let prevFilterStatus = 'all';

  let form = $state({
    username: '',
    email: '',
    password: '',
    role_id: 0,
    is_active: true
  });

  let isFiltered = $derived(filterRole !== 'all' || filterStatus !== 'all' || sortBy !== 'username' || sortDir !== 'asc');

  let pills = $derived(() => {
    const result = [];
    if (filterRole !== 'all') {
      const r = roles.find(role => String(role.id) === filterRole);
      result.push({ key: 'role', label: r ? r.name : filterRole });
    }
    if (filterStatus !== 'all') {
      result.push({ key: 'status', label: filterStatus === 'true' ? 'Active' : 'Inactive' });
    }
    return result;
  });

  function removePill(key) {
    if (key === 'role') filterRole = 'all';
    if (key === 'status') filterStatus = 'all';
  }

  const roleVariant = (r) => {
    const roleName = typeof r === 'object' ? r.name : r;
    if (roleName === 'superadmin') return 'primary';
    if (roleName === 'admin') return 'warning';
    return 'muted';
  };

  async function fetchUsers(isSearch = false) {
    try {
      if (!isSearch) loading = true;
      const params = { limit, offset, search: searchQuery, sort: sortBy, dir: sortDir };
      if (filterRole !== 'all') params.role = filterRole;
      if (filterStatus !== 'all') params.status = filterStatus;
      const uRes = await apiClient.get('/admin/users', { params });
      users = uRes.data?.data || [];
      total = uRes.data?.total || 0;
    } catch {
      toast.error('Failed to load users');
    } finally {
      if (!isSearch) loading = false;
      isSearching = false;
    }
  }

  async function fetchRoles() {
    try {
      const rRes = await apiClient.get('/admin/roles');
      roles = rRes.data?.data || [];
      if (roles.length > 0 && form.role_id === 0) {
        form.role_id = roles[0].id;
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

  const debouncedSearch = debounce(() => {
    offset = 0;
    prevOffset = 0;
    fetchUsers(true);
  }, 400);

  $effect(() => {
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

  $effect(() => {
    const off = offset;
    const lim = limit;
    if (isInitialMount) return;
    if (off !== prevOffset || lim !== prevLimit) {
      prevOffset = off;
      prevLimit = lim;
      fetchUsers(false);
    }
  });

  $effect(() => {
    const sb = sortBy;
    const sd = sortDir;
    if (isInitialMount) return;
    if (sb !== prevSortBy || sd !== prevSortDir) {
      prevSortBy = sb;
      prevSortDir = sd;
      offset = 0;
      prevOffset = 0;
      fetchUsers(false);
    }
  });

  $effect(() => {
    const fr = filterRole;
    const fs = filterStatus;
    if (isInitialMount) return;
    if (fr !== prevFilterRole || fs !== prevFilterStatus) {
      prevFilterRole = fr;
      prevFilterStatus = fs;
      offset = 0;
      prevOffset = 0;
      fetchUsers(false);
    }
  });

  function handlePageChange(newOffset, newLimit) {
    offset = newOffset;
    limit = newLimit;
  }

  function toggleSort(key) {
    if (sortBy === key) {
      sortDir = sortDir === 'asc' ? 'desc' : 'asc';
    } else {
      sortBy = key;
      sortDir = 'asc';
    }
  }

  function clearFilters() {
    filterRole = 'all';
    filterStatus = 'all';
    sortBy = 'username';
    sortDir = 'asc';
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
      password: '',
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
      const url = modalMode === 'add' ? '/admin/users' : `/admin/users/${selectedUser.id}`;

      const r = await apiClient({ url, method, data: form });

      if (r.status >= 200 && r.status < 300) {
        toast.success(modalMode === 'add' ? 'User created' : 'User updated');
        showModal = false;
        await fetchUsers();
      } else {
        toast.error(r.data?.error || 'Failed to save user');
      }
    } catch (error) {
      const message = error?.response?.data?.error || error?.message || 'Network error';
      toast.error(message);
    } finally {
      saving = false;
    }
  }

  async function confirmDelete() {
    if (!selectedUser) return;
    if (selectedUser.id === currentUserID) {
      toast.error('You cannot delete your own account');
      return;
    }
    try {
      await apiClient.delete(`/admin/users/${selectedUser.id}`);
      toast.success(`User "${selectedUser.username}" removed`);
      await fetchUsers();
    } catch {
      toast.error('Failed to delete user');
    } finally {
      showDeleteModal = false;
      selectedUser = null;
    }
  }
</script>

<div class="space-y-5">
  {#if !canView}
    <div class="card px-4 py-12 text-center">
      <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
        <Users size={32} class="text-text-muted" />
      </div>
      <p class="text-text-primary font-semibold mt-4">Access Denied</p>
      <p class="text-text-muted text-sm mt-1">You do not have permission to view users</p>
    </div>
  {:else}
    <div class="card p-4">
      <div class="flex items-center gap-3">
        <div class="relative flex-1">
          <Search size={16} class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none" />
          <input type="text" placeholder="Search users by name or email…" class="input pl-9 pr-10 w-full" bind:value={searchQuery} />
          <button
            onclick={() => searchQuery = ''}
            class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-secondary transition-all duration-150 {searchQuery ? 'opacity-100' : 'opacity-0 pointer-events-none'}"
            title="Clear search"
          >
            <X size={14} />
          </button>
        </div>
        <div class="relative shrink-0" style="width: 140px; min-width: 140px; max-width: 140px;">
          <select
            class="appearance-none bg-surface-default border border-border rounded-xl py-2.5 pl-3 pr-8 text-sm text-text-secondary hover:border-border-strong hover:text-text-primary focus:text-text-primary focus:outline-none focus:border-primary-default focus:ring-2 focus:ring-primary-default/20 transition-colors cursor-pointer {filterRole !== 'all' ? 'text-text-primary' : ''}"
            style="width: 140px; min-width: 140px; max-width: 140px;"
            bind:value={filterRole}
          >
            <option value="all">All Roles</option>
            {#each roles as role}
              <option value={String(role.id)}>{role.name}</option>
            {/each}
          </select>
          <ChevronDown size={14} class="absolute right-2.5 top-1/2 -translate-y-1/2 pointer-events-none text-text-muted" />
        </div>
        <div class="relative shrink-0" style="width: 128px; min-width: 128px; max-width: 128px;">
          <select
            class="appearance-none bg-surface-default border border-border rounded-xl py-2.5 pl-3 pr-8 text-sm text-text-secondary hover:border-border-strong hover:text-text-primary focus:text-text-primary focus:outline-none focus:border-primary-default focus:ring-2 focus:ring-primary-default/20 transition-colors cursor-pointer {filterStatus !== 'all' ? 'text-text-primary' : ''}"
            style="width: 128px; min-width: 128px; max-width: 128px;"
            bind:value={filterStatus}
          >
            <option value="all">All Status</option>
            <option value="true">Active</option>
            <option value="false">Inactive</option>
          </select>
          <ChevronDown size={14} class="absolute right-2.5 top-1/2 -translate-y-1/2 pointer-events-none text-text-muted" />
        </div>
        {#if canCreate}
        <button
          onclick={openAdd}
          class="btn btn-primary shrink-0 shadow-glow-primary-sm px-5"
        >
          <Plus size={18} />
          Add User
        </button>
        {/if}
      </div>
      <div class="flex items-center gap-2 mt-2.5 pt-2.5 border-t border-border-subtle h-7 transition-opacity duration-150 {pills().length > 0 ? 'opacity-100' : 'opacity-0 pointer-events-none'}">
        <SlidersHorizontal size={12} class="text-text-muted shrink-0" />
        {#each pills() as pill}
          <span class="inline-flex items-center gap-1 rounded-md bg-primary-subtle text-primary-light border border-primary-default/20 px-2 py-0.5 text-xs font-medium">
            {pill.label}
            <button onclick={() => removePill(pill.key)} class="hover:text-white transition-colors">
              <X size={10} />
            </button>
          </span>
        {/each}
        <button
          onclick={clearFilters}
          class="ml-auto text-xs font-medium text-text-muted hover:text-danger transition-colors shrink-0"
        >
          Clear all
        </button>
      </div>
    </div>

    <div class="card p-0 overflow-hidden">
      <div class="overflow-x-auto" style="min-width: 0;">
        <table class="w-full table-fixed border-collapse" style="min-width: 680px;">
          <thead class="bg-muted/50">
             <tr>
                <th class="text-left p-4 font-semibold">
                  <button class="flex items-center gap-1 hover:text-primary transition-colors" onclick={() => toggleSort('username')}>
                    USER <ArrowUpDown size={14} class="text-text-muted" />
                  </button>
                </th>
                <th class="text-left p-4 font-semibold w-40">
                  <button class="flex items-center gap-1 hover:text-primary transition-colors" onclick={() => toggleSort('role_id')}>
                    ROLE <ArrowUpDown size={14} class="text-text-muted" />
                  </button>
                </th>
                <th class="text-left p-4 font-semibold w-28">STATUS</th>
                <th class="text-left p-4 font-semibold w-56">
                  <button class="flex items-center gap-1 hover:text-primary transition-colors" onclick={() => toggleSort('last_login')}>
                    LAST LOGIN <ArrowUpDown size={14} class="text-text-muted" />
                  </button>
                </th>
                <th class="text-center p-4 font-semibold w-28">ACTIONS</th>
              </tr>
          </thead>
          <tbody>
            {#if loading}
              {#each { length: 5 } as _}
                <tr class="border-t border-border">
                  <td>
                    <div class="flex items-center gap-3">
                      <Skeleton width="w-9" height="h-9" rounded="rounded-full" />
                      <div class="flex flex-col gap-1.5">
                        <Skeleton width="w-32" height="h-3.5" />
                        <Skeleton width="w-44" height="h-3" />
                      </div>
                    </div>
                  </td>
                  <td>
                    <Skeleton width="w-16" height="h-6" rounded="rounded-full" />
                  </td>
                  <td>
                    <div class="flex items-center gap-2">
                      <Skeleton width="w-1.5" height="h-1.5" rounded="rounded-full" />
                      <Skeleton width="w-12" height="h-3.5" />
                    </div>
                  </td>
                  <td>
                    <Skeleton width="w-36" height="h-3.5" />
                  </td>
                  <td>
                    <div class="flex items-center justify-center gap-2">
                      <Skeleton width="w-8" height="h-8" rounded="rounded-xl" />
                      <Skeleton width="w-8" height="h-8" rounded="rounded-xl" />
                    </div>
                  </td>
                </tr>
              {/each}
            {:else if users.length === 0}
              <tr>
                <td colspan="5" class="px-4 py-12 text-center">
                  <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
                    <Users size={32} class="text-text-muted" />
                  </div>
                  <p class="text-text-primary font-semibold mt-4">No users found</p>
                  <p class="text-text-muted text-sm mt-1">
                    {searchQuery ? `No match for "${searchQuery}"` : 'Start by adding a user'}
                  </p>
                </td>
              </tr>
            {:else}
              {#each users as user (user.id)}
                <tr class="border-t border-border hover:bg-surface-hover/50 transition-colors">
                  <td class="p-4 min-w-0 overflow-hidden text-ellipsis whitespace-nowrap">
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
                       {user.role?.name || (user.role_id === 1 ? 'superadmin' : user.role_id === 2 ? 'admin' : user.role_id === 3 ? 'cashier' : user.role_id === 4 ? 'manager' : user.role_id === 5 ? 'staff' : 'unknown')}
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
                   <td class="text-center">
                     <div class="flex items-center justify-center gap-2">
                      <button
                        class="btn-icon btn-ghost text-text-muted hover:text-primary-light"
                        title="Edit"
                        onclick={() => openEdit(user)}
                        disabled={user.role_id === 1 && !canEditSuperadmin}
                      >
                        <Pencil size={14} />
                      </button>
                      <button
                        class="btn-icon btn-ghost text-text-muted hover:text-danger hover:bg-danger-subtle"
                        onclick={() => { selectedUser = user; showDeleteModal = true; }}
                        title="Delete"
                        disabled={user.id === currentUserID || user.role_id === 1}
                      >
                        <Trash2 size={14} />
                      </button>
                    </div>
                  </td>
                </tr>
              {/each}
            {/if}
          </tbody>
        </table>
      </div>

      {#if !loading && users.length > 0}
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
  {/if}
</div>

<Modal bind:open={showModal} title={modalMode === 'add' ? 'Add New User' : 'Edit User'} size="md">
  <form onsubmit={(e) => { e.preventDefault(); saveUser(); }} class="space-y-6">
    <div class="space-y-4">
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label for="usr-username" class="flex items-center gap-2 text-sm font-medium text-text-secondary mb-2">
            <User size={14} class="text-text-muted" />
            Username
          </label>
          <input id="usr-username" type="text" placeholder="johndoe" class="input" bind:value={form.username} required minlength="3" maxlength="50" pattern="[a-zA-Z0-9]+" title="3-50 alphanumeric characters only (will be converted to lowercase)" />
        </div>
        <div>
          <label for="usr-email" class="flex items-center gap-2 text-sm font-medium text-text-secondary mb-2">
            Email Address
          </label>
          <input id="usr-email" type="email" placeholder="john@example.com" class="input" bind:value={form.email} required />
        </div>
      </div>

      <div class="grid grid-cols-2 gap-4">
        <div>
          <label for="usr-role" class="flex items-center gap-2 text-sm font-medium text-text-secondary mb-2">
            <Shield size={14} class="text-text-muted" />
            Role
          </label>
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

      <div class="pt-2">
        <label for="usr-password" class="flex items-center gap-2 text-sm font-medium text-text-secondary mb-2">
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-text-muted"><rect width="18" height="11" x="3" y="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
          {modalMode === 'add' ? 'Password' : 'New Password (optional)'}
        </label>
        <input id="usr-password" type="password" placeholder="••••••••" class="input" bind:value={form.password} required={modalMode === 'add'} minlength="6" />
        {#if modalMode === 'edit'}
          <p class="text-xs text-text-muted mt-1.5">Leave blank to keep current password</p>
        {/if}
      </div>
    </div>
  </form>
  {#snippet footer()}
    <button class="btn btn-secondary" onclick={() => showModal = false} disabled={saving}>Cancel</button>
    <button class="btn btn-primary min-w-32" onclick={saveUser} disabled={saving || (modalMode === 'add' && usernameHasInvalidChars)}>
      {#if saving}
        <Loader2 size={16} class="animate-spin" /> Saving...
      {:else}
        {modalMode === 'add' ? 'Create User' : 'Save Changes'}
      {/if}
    </button>
  {/snippet}
</Modal>

<Modal bind:open={showDeleteModal} title="Delete User" size="sm">
  <div class="text-center py-3">
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
