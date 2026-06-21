<script>
  import { onMount } from 'svelte';
  import apiClient from '$lib/api/client';
  import { toast } from '$lib/stores/toast';
  import { debounce } from '$lib/utils/debounce';
  import { auth } from '$lib/stores/auth';
  import { formatDateInJakarta, formatTimeInJakarta } from '$lib/utils/jakartaTime';

  import Badge from '$lib/components/ui/Badge.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import Pagination from '$lib/components/ui/Pagination.svelte';
  import SearchBar from '$lib/components/ui/SearchBar.svelte';
  import { Search, Plus, Pencil, Trash2, User, Users, Loader2, X, Shield, ChevronDown, SlidersHorizontal } from 'lucide-svelte';

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
  let showRoleDropdown = $state(false);
  let showStatusDropdown = $state(false);
  let showFormRoleDropdown = $state(false);

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

  let roleLabel = $derived(filterRole === 'all' ? 'All Roles' : roles.find(r => String(r.id) === filterRole)?.name || filterRole);
  let statusLabel = $derived(filterStatus === 'all' ? 'All Status' : filterStatus === 'true' ? 'Active' : 'Inactive');
  let selectedRoleName = $derived(roles.find(r => r.id === form.role_id)?.name || 'Select Role');

  let dropdownStyle = $state('');

  function toggleFormRoleDropdown(e) {
    if (showFormRoleDropdown) {
      showFormRoleDropdown = false;
      dropdownStyle = '';
      return;
    }
    const btn = e.target.closest('button');
    if (btn) {
      const rect = btn.getBoundingClientRect();
      dropdownStyle = `position:fixed; top:${rect.bottom + 4}px; left:${rect.left}px; width:${rect.width}px;`;
    }
    showFormRoleDropdown = true;
  }

  function handleFormKeydown(e) {
    if (e.key === 'Escape' && showFormRoleDropdown) {
      showFormRoleDropdown = false;
      dropdownStyle = '';
      e.stopPropagation();
    }
  }

  function handleFormClick(e) {
    if (showFormRoleDropdown && !e.target.closest('.form-role-dropdown-container')) {
      showFormRoleDropdown = false;
      dropdownStyle = '';
    }
  }

  let isFiltered = $derived(filterRole !== 'all' || filterStatus !== 'all' || sortBy !== 'username' || sortDir !== 'asc');

  let activeChips = $derived.by(() => {
    const chips = [];
    if (filterRole !== 'all') {
      const r = roles.find(role => String(role.id) === filterRole);
      chips.push({ type: 'role', label: r ? r.name : filterRole });
    }
    if (filterStatus !== 'all') {
      chips.push({ type: 'status', label: filterStatus === 'true' ? 'Active' : 'Inactive' });
    }
    return chips;
  });

  function clearFilter(type) {
    if (type === 'role') filterRole = 'all';
    if (type === 'status') filterStatus = 'all';
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

  const handleClickOutside = (e) => {
    if (showRoleDropdown && !e.target.closest('.role-filter-container')) showRoleDropdown = false;
    if (showStatusDropdown && !e.target.closest('.status-filter-container')) showStatusDropdown = false;
    if (showFormRoleDropdown && !e.target.closest('.form-role-dropdown-container')) { showFormRoleDropdown = false; dropdownStyle = ''; }
  };

  const handleEsc = (e) => {
    if (e.key === 'Escape') {
      if (showFormRoleDropdown) {
        showFormRoleDropdown = false;
        dropdownStyle = '';
        e.stopPropagation();
        return;
      }
      showRoleDropdown = false;
      showStatusDropdown = false;
    }
  };

  onMount(async () => {
    isInitialMount = true;
    await fetchRoles();
    await fetchUsers(false);
    isInitialMount = false;
    document.addEventListener('click', handleClickOutside);
    document.addEventListener('keydown', handleEsc);
    return () => {
      document.removeEventListener('click', handleClickOutside);
      document.removeEventListener('keydown', handleEsc);
    };
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

  function clearAllFilters() {
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
        <div class="flex-1">
          <SearchBar bind:value={searchQuery} placeholder="Search by username or email…" />
        </div>
        <div class="relative shrink-0 role-filter-container">
          <button
            class="flex items-center gap-2 px-3 h-10 rounded-xl border border-border bg-surface-default text-sm hover:border-border-strong hover:bg-surface-hover transition-colors {filterRole !== 'all' ? 'text-text-primary' : 'text-text-secondary'}"
            style="min-width: 140px;"
            onclick={() => showRoleDropdown = !showRoleDropdown}
          >
            <span class="flex-1 text-left truncate">{roleLabel}</span>
            <ChevronDown size={14} class="text-text-muted shrink-0" />
          </button>
          {#if showRoleDropdown}
            <div class="absolute left-0 top-full mt-2 z-50 bg-surface-default border border-border rounded-lg shadow-xl p-2 min-w-[360px] max-h-64 overflow-y-auto">
              <button
                class="w-full text-left px-3 py-2 text-sm rounded-lg transition-colors truncate {filterRole === 'all' ? 'text-primary-light bg-primary-subtle/30 font-medium' : 'text-text-secondary hover:bg-surface-hover'}"
                onclick={() => { filterRole = 'all'; showRoleDropdown = false; }}
              >All Roles</button>
              <div class="grid grid-cols-2 gap-1 mt-1">
                {#each roles as role}
                  <button
                    class="text-left px-3 py-2 text-sm rounded-lg transition-colors truncate {filterRole === String(role.id) ? 'text-primary-light bg-primary-subtle/30 font-medium' : 'text-text-secondary hover:bg-surface-hover'}"
                    onclick={() => { filterRole = String(role.id); showRoleDropdown = false; }}
                    title={role.name}
                  >{role.name}</button>
                {/each}
              </div>
            </div>
          {/if}
        </div>
        <div class="relative shrink-0 status-filter-container" style="width: 128px; min-width: 128px; max-width: 128px;">
          <button
            class="flex items-center gap-2 px-3 h-10 w-full rounded-xl border border-border bg-surface-default text-sm hover:border-border-strong hover:bg-surface-hover transition-colors {filterStatus !== 'all' ? 'text-text-primary' : 'text-text-secondary'}"
            onclick={() => showStatusDropdown = !showStatusDropdown}
          >
            <span class="flex-1 text-left truncate">{statusLabel}</span>
            <ChevronDown size={14} class="text-text-muted shrink-0" />
          </button>
          {#if showStatusDropdown}
            <div class="absolute left-0 top-full mt-2 z-50 bg-surface-default border border-border rounded-lg shadow-xl py-1 min-w-[160px]">
              <button
                class="w-full text-left px-4 py-2 text-sm transition-colors {filterStatus === 'all' ? 'text-primary-light bg-primary-subtle/30 font-medium' : 'text-text-secondary hover:bg-surface-hover'}"
                onclick={() => { filterStatus = 'all'; showStatusDropdown = false; }}
              >All Status</button>
              <button
                class="w-full text-left px-4 py-2 text-sm transition-colors {filterStatus === 'true' ? 'text-primary-light bg-primary-subtle/30 font-medium' : 'text-text-secondary hover:bg-surface-hover'}"
                onclick={() => { filterStatus = 'true'; showStatusDropdown = false; }}
              >Active</button>
              <button
                class="w-full text-left px-4 py-2 text-sm transition-colors {filterStatus === 'false' ? 'text-primary-light bg-primary-subtle/30 font-medium' : 'text-text-secondary hover:bg-surface-hover'}"
                onclick={() => { filterStatus = 'false'; showStatusDropdown = false; }}
              >Inactive</button>
            </div>
          {/if}
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
      <div class="filter-chips-wrapper" class:is-open={activeChips.length > 0}>
        <div class="filter-chips-inner">
          <div class="flex flex-wrap items-center gap-2 pt-3 mt-3 border-t border-border/50">
            {#each activeChips as chip}
              <div class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-primary-subtle/20 border border-primary-subtle/30 rounded-full text-sm text-text-secondary">
                <SlidersHorizontal size={13} class="text-primary-light shrink-0" />
                <span class="font-medium truncate max-w-[180px]">{chip.label}</span>
                <button
                  class="w-4 h-4 rounded-full flex items-center justify-center text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors"
                  onclick={() => clearFilter(chip.type)}
                >
                  <X size={12} />
                </button>
              </div>
            {/each}
            <button
              class="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-text-muted hover:text-text-primary bg-surface-default/50 border border-border/50 rounded-full transition-colors"
              onclick={clearAllFilters}
            >
              Clear all
              <X size={12} />
            </button>
          </div>
        </div>
      </div>
    </div>

    <div class="card p-0 overflow-hidden">
      <div class="overflow-x-auto" style="min-width: 0;">
        <table class="w-full table-fixed border-collapse" style="min-width: 680px;">
          <thead class="bg-muted/50">
             <tr>
                 <th class="text-left p-4 font-semibold" style="width: 30%;">
                  <button class="flex items-center gap-1 hover:text-primary transition-colors" onclick={() => toggleSort('username')}>
                    USER {#if sortBy === 'username'}<span>{sortDir === 'asc' ? '▲' : '▼'}</span>{/if}
                  </button>
                </th>
                <th class="text-left p-4 font-semibold w-40">
                  <button class="flex items-center gap-1 hover:text-primary transition-colors" onclick={() => toggleSort('role_id')}>
                    ROLE {#if sortBy === 'role_id'}<span>{sortDir === 'asc' ? '▲' : '▼'}</span>{/if}
                  </button>
                </th>
                <th class="text-left p-4 font-semibold w-28">STATUS</th>
                <th class="text-left p-4 font-semibold w-44">
                  <button class="flex items-center gap-1 hover:text-primary transition-colors" onclick={() => toggleSort('last_login')}>
                    LAST LOGIN {#if sortBy === 'last_login'}<span>{sortDir === 'asc' ? '▲' : '▼'}</span>{/if}
                  </button>
                </th>
                <th class="text-center p-4 font-semibold w-20">ACTIONS</th>
              </tr>
          </thead>
          <tbody>
            {#if loading}
              {#each { length: 5 } as _}
                <tr class="border-t border-border">
                  <td class="p-4">
                    <div class="flex items-center gap-3">
                      <Skeleton width="w-8" height="h-8" rounded="rounded-full" />
                      <div class="flex flex-col gap-1.5">
                        <Skeleton width="w-32" height="h-3.5" />
                        <Skeleton width="w-44" height="h-3" />
                      </div>
                    </div>
                  </td>
                  <td class="p-4">
                    <Skeleton width="w-16" height="h-6" rounded="rounded-full" />
                  </td>
                  <td class="p-4">
                    <div class="flex items-center gap-2">
                      <Skeleton width="w-1.5" height="h-1.5" rounded="rounded-full" />
                      <Skeleton width="w-12" height="h-3.5" />
                    </div>
                  </td>
                  <td class="p-4">
                    <Skeleton width="w-36" height="h-3.5" />
                  </td>
                  <td class="p-4">
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
                      <div class="w-8 h-8 rounded-full gradient-bg-primary flex items-center justify-center shrink-0">
                        <User size={14} class="text-white" />
                      </div>
                      <div>
                        <p class="font-medium text-text-primary">{user.username}</p>
                        <p class="text-xs text-text-muted">{user.email || '—'}</p>
                      </div>
                    </div>
                  </td>
                  <td class="p-4">
                     <Badge variant={roleVariant(user.role)}>
                       {user.role?.name || (user.role_id === 1 ? 'superadmin' : user.role_id === 2 ? 'admin' : user.role_id === 3 ? 'cashier' : user.role_id === 4 ? 'manager' : user.role_id === 5 ? 'staff' : 'unknown')}
                     </Badge>
                   </td>
                   <td class="p-4">
                     <div class="flex items-center gap-2">
                       <span class="w-1.5 h-1.5 rounded-full {user.is_active !== false ? 'bg-success animate-pulse-dot' : 'bg-text-muted'}"></span>
                       <span class="text-sm text-text-secondary">{user.is_active !== false ? 'Active' : 'Inactive'}</span>
                     </div>
                   </td>
                   <td class="p-4 text-text-muted text-sm leading-relaxed">
                       {#if user.last_login}
                         <span class="block">{formatDateInJakarta(user.last_login)}</span>
                         <span class="block text-[10px] text-text-muted">{formatTimeInJakarta(user.last_login)}</span>
                       {:else}
                         Never
                       {/if}
                    </td>
                   <td class="p-4 text-center">
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
        <div class="p-4 bg-surface-subtle/30 border-t border-border/50">
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
  <form onsubmit={(e) => { e.preventDefault(); saveUser(); }} class="space-y-6" onkeydown={handleFormKeydown} onclick={handleFormClick}>
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
        <div class="relative form-role-dropdown-container">
          <label class="flex items-center gap-2 text-sm font-medium text-text-secondary mb-2">
            <Shield size={14} class="text-text-muted" />
            Role
          </label>
          <div class="form-role-dropdown-container">
            <button
              type="button"
              class="flex items-center gap-2 px-3 h-10 w-full rounded-xl border border-border bg-surface-default text-sm hover:border-border-strong hover:bg-surface-hover transition-colors {form.role_id ? 'text-text-primary' : 'text-text-muted'}"
              onclick={toggleFormRoleDropdown}
            >
              <span class="flex-1 text-left truncate">{selectedRoleName}</span>
              <ChevronDown size={14} class="text-text-muted shrink-0" />
            </button>
            {#if showFormRoleDropdown}
              <div style={dropdownStyle} class="z-[100] bg-surface-default border border-border rounded-lg shadow-xl">
                <div class="p-1.5 max-h-64 overflow-y-auto">
                  <div class="grid grid-cols-2 gap-1">
                    {#each roles as role}
                      <button
                        type="button"
                        class="w-full text-left px-3 py-2 text-sm rounded-lg transition-colors truncate {form.role_id === role.id ? 'text-primary-light bg-primary-subtle/30 font-medium' : 'text-text-secondary hover:bg-surface-hover'}"
                        onclick={() => { form.role_id = role.id; showFormRoleDropdown = false; }}
                      >{role.name}</button>
                    {/each}
                  </div>
                </div>
              </div>
            {/if}
          </div>
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

<style>
  .filter-chips-wrapper {
    display: grid;
    grid-template-rows: 0fr;
    opacity: 0;
    transition: grid-template-rows 0.2s ease-out, opacity 0.2s ease-out;
  }

  .filter-chips-wrapper.is-open {
    grid-template-rows: 1fr;
    opacity: 1;
  }

  .filter-chips-inner {
    overflow: hidden;
  }
</style>
