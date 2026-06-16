<script>
  import { onMount } from 'svelte';
  import { fly } from 'svelte/transition';
  import { apiFetch } from '$lib/api/client';
  import { toast } from '$lib/stores/toast';
  import { auth } from '$lib/stores/auth';

  import SearchBar from '$lib/components/ui/SearchBar.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import Pagination from '$lib/components/ui/Pagination.svelte';
  import { Plus, Pencil, Trash2, Shield, Loader2, Search, X, ChevronRight, ChevronDown, ChevronLeft, ChevronsUpDown, Check, ChevronsLeft, ChevronsRight, Users, Package, Tag, ShoppingCart, Warehouse, UserPlus, BarChart3, LayoutDashboard, Settings, Store, Eye, RefreshCw, Copy, AlertTriangle, ArrowUpDown, MoreVertical } from 'lucide-svelte';

  // ── State ────────────────────────────────────────────────────────
  let loading = $state(true);
  let roles = $state([]);
  let permissions = $state([]);
  let showModal = $state(false);
  let showDeleteModal = $state(false);
  let showDiscardModal = $state(false);
  let selectedRole = $state(null);
  let modalMode = $state('add');
  let modalStep = $state(1);
  let saving = $state(false);
  let permissionSearch = $state('');
  let nameTouched = $state(false);
  let pendingClose = $state(false);
  let loadError = $state('');

  // ── Pagination ───────────────────────────────────────────────────
  let pageLimit = $state(20);
  let pageOffset = $state(0);

  // ── Roles list search / filter ───────────────────────────────────
  let roleSearch = $state('');
  let roleSearchDebounced = $state('');
  let filterType = $state('all');

  // ── Sorting (table header) ───────────────────────────────────────
  let sortField = $state('name');
  let sortDir = $state('asc');

  // ── Expanded detail row ──────────────────────────────────────────
  let expandedRoleId = $state(null);

  // ── Filter type dropdown ─────────────────────────────────────────
  let showFilterTypeDropdown = $state(false);

  // ── Action dropdown ──────────────────────────────────────────────
  let openActionRoleId = $state(null);

  // ── Role detail drawer ───────────────────────────────────────────
  let showRoleDrawer = $state(false);

  // Debounce role search
  let searchTimer = null;
  function handleRoleSearch(val) {
    pageOffset = 0;
    clearTimeout(searchTimer);
    searchTimer = setTimeout(() => {
      roleSearchDebounced = val.trim().toLowerCase();
    }, 300);
  }

  // ── Filtered + sorted roles ─────────────────────────────────────
  let filteredRoles = $derived(() => {
    let result = [...roles];
    if (roleSearchDebounced) {
      result = result.filter(r =>
        r.name.toLowerCase().includes(roleSearchDebounced) ||
        (r.description || '').toLowerCase().includes(roleSearchDebounced)
      );
    }
    if (filterType === 'system') result = result.filter(r => r.is_system);
    else if (filterType === 'custom') result = result.filter(r => !r.is_system);
    result.sort((a, b) => {
      let cmp = 0;
      if (sortField === 'name') cmp = a.name.localeCompare(b.name);
      else if (sortField === 'permissions') {
        const aLen = Array.isArray(a.permissions) ? a.permissions.length : 0;
        const bLen = Array.isArray(b.permissions) ? b.permissions.length : 0;
        cmp = aLen - bLen;
      }
      return sortDir === 'asc' ? cmp : -cmp;
    });
    return result;
  });

  // ── Paginated roles ──────────────────────────────────────────────
  let paginatedRoles = $derived(() => {
    const all = filteredRoles();
    return all.slice(pageOffset, pageOffset + pageLimit);
  });

  let totalFiltered = $derived(filteredRoles().length);

  let isFiltered = $derived(roleSearchDebounced !== '' || filterType !== 'all');

  function clearFilters() {
    roleSearch = '';
    roleSearchDebounced = '';
    filterType = 'all';
    pageOffset = 0;
  }

  function handlePageChange(newOffset, newLimit) {
    if (newLimit && newLimit !== pageLimit) pageLimit = newLimit;
    pageOffset = newOffset;
  }

  function toggleSort(field) {
    if (sortField === field) sortDir = sortDir === 'asc' ? 'desc' : 'asc';
    else { sortField = field; sortDir = 'asc'; }
  }

  function sortIcon(field) {
    if (sortField !== field) return 'none';
    return sortDir;
  }

  // ── Detail panel ────────────────────────────────────────────────
  function toggleDetail(roleId) {
    expandedRoleId = expandedRoleId === roleId ? null : roleId;
  }

  function getRolePermissions(role) {
    if (!role.permissions || !role.permissions.length) return [];
    const permCodes = new Set(role.permissions);
    return permissions.filter(p => permCodes.has(p.code));
  }

  function getGroupedPermissions(rolePerms) {
    const grouped = {};
    for (const perm of rolePerms) {
      let key = perm.code.split(':')[0];
      if (key === 'role') key = 'user';
      if (!grouped[key]) grouped[key] = [];
      grouped[key].push(perm);
    }
    return Object.entries(grouped);
  }

  // ── Permissions grouping ────────────────────────────────────────
  let collapsedGroups = $state(new Set());
  let initialPermissionIds = $state([]);
  let initialFormData = $state({ name: '', description: '' });

  const groupMeta = {
    'user': { label: 'User & Role', icon: Users },
    'product': { label: 'Product', icon: Package },
    'category': { label: 'Category', icon: Tag },
    'sale': { label: 'Sales', icon: ShoppingCart },
    'inventory': { label: 'Inventory', icon: Warehouse },
    'customer': { label: 'Customer', icon: UserPlus },
    'report': { label: 'Report', icon: BarChart3 },
    'dashboard': { label: 'Dashboard', icon: LayoutDashboard },
    'pos': { label: 'POS', icon: Store },
    'audit': { label: 'System', icon: Settings },
  };

  let groupedPermissions = $derived(() => {
    const filtered = permissionSearch.trim()
      ? permissions.filter(p =>
          p.name.toLowerCase().includes(permissionSearch.toLowerCase()) ||
          p.code.toLowerCase().includes(permissionSearch.toLowerCase())
        )
      : permissions;
    const groups = {};
    for (const p of filtered) {
      let key = p.code.split(':')[0];
      if (key === 'role') key = 'user';
      if (!groups[key]) groups[key] = [];
      groups[key].push(p);
    }
    const order = ['user', 'product', 'category', 'sale', 'inventory', 'customer', 'report', 'dashboard', 'pos', 'audit'];
    return order
      .filter(key => groups[key]?.length)
      .map(key => ({
        key,
        label: groupMeta[key]?.label || key,
        icon: groupMeta[key]?.icon || Shield,
        permissions: groups[key],
      }));
  });

  let selectedGroupCount = $derived(() => {
    const uniqueGroups = new Set();
    for (const gid of form.permission_ids) {
      const p = permissions.find(pp => pp.id === gid);
      if (p) { let key = p.code.split(':')[0]; if (key === 'role') key = 'user'; uniqueGroups.add(key); }
    }
    return uniqueGroups.size;
  });

  // ── Validation ──────────────────────────────────────────────────
  let nameError = $derived(() => {
    if (!nameTouched && modalStep === 1) return '';
    if (!form.name.trim()) return 'Role name is required';
    if (roles.some(r => r.name.toLowerCase() === form.name.trim().toLowerCase() && r.id !== selectedRole?.id)) {
      return 'Role name already exists';
    }
    return '';
  });
  let nameErrorText = $derived(nameError());

  let hasUnsavedChanges = $derived(() => {
    if (modalMode === 'add') return form.permission_ids.length > 0 || form.name.trim() !== '' || form.description.trim() !== '';
    const permsCurrent = JSON.stringify([...form.permission_ids].sort());
    const permsInitial = JSON.stringify([...initialPermissionIds].sort());
    if (permsCurrent !== permsInitial) return true;
    return form.name !== initialFormData.name || form.description !== initialFormData.description;
  });

  let isFormDirty = $derived(() => {
    if (modalMode === 'add') return form.name.trim() !== '' || form.permission_ids.length > 0;
    return hasUnsavedChanges();
  });

  let userRole = $derived($auth.user?.role?.name || ($auth.user?.role && typeof $auth.user?.role === 'object' ? $auth.user.role.name : $auth.user?.role) || '');
  let canView = $derived(userRole !== 'cashier' && userRole !== '');
  let canEdit = $derived(userRole === 'superadmin');
  let canDelete = $derived(userRole === 'superadmin');

  function allExpanded() { return groupedPermissions().every(g => !collapsedGroups.has(g.key)); }
  function toggleGroup(key) { const s = new Set(collapsedGroups); if (s.has(key)) s.delete(key); else s.add(key); collapsedGroups = s; }
  function setAllExpanded(expanded) { collapsedGroups = expanded ? new Set() : new Set(groupedPermissions().map(g => g.key)); }

  function toggleGroupAll(group) {
    const allSel = group.permissions.every(p => form.permission_ids.includes(p.id));
    if (allSel) form.permission_ids = form.permission_ids.filter(id => !group.permissions.some(p => p.id === id));
    else form.permission_ids = [...form.permission_ids, ...group.permissions.filter(p => !form.permission_ids.includes(p.id)).map(p => p.id)];
  }
  function isGroupAllSelected(group) { return group.permissions.every(p => form.permission_ids.includes(p.id)); }
  function isGroupPartialSelected(group) { return group.permissions.some(p => form.permission_ids.includes(p.id)) && !isGroupAllSelected(group); }

  function handleGroupKeydown(e, group) {
    if (e.key === 'ArrowDown') { e.preventDefault(); const groups = groupedPermissions(); const idx = groups.findIndex(g => g.key === group.key); if (idx < groups.length - 1) document.querySelector(`[data-group-toggle][data-group-key="${groups[idx + 1].key}"]`)?.focus(); }
    else if (e.key === 'ArrowUp') { e.preventDefault(); const groups = groupedPermissions(); const idx = groups.findIndex(g => g.key === group.key); if (idx > 0) document.querySelector(`[data-group-toggle][data-group-key="${groups[idx - 1].key}"]`)?.focus(); }
  }

  let form = $state({ name: '', description: '', permission_ids: [] });

  async function fetchData() {
    try {
      loading = true;
      loadError = '';
      const [rRes, pRes] = await Promise.all([apiFetch('/api/admin/roles'), apiFetch('/api/admin/permissions')]);
      if (rRes.ok) { const data = await rRes.json(); roles = data.data || []; }
      if (pRes.ok) { const data = await pRes.json(); permissions = data.data || []; }
    } catch { loadError = 'Failed to load roles and permissions'; toast.error(loadError); }
    finally { loading = false; }
  }

  function resetModal() {
    modalMode = 'add'; modalStep = 1; permissionSearch = '';
    collapsedGroups = new Set(permissions.map(p => { let k = p.code.split(':')[0]; return k === 'role' ? 'user' : k; }));
    nameTouched = false; initialPermissionIds = []; initialFormData = { name: '', description: '' };
    form = { name: '', description: '', permission_ids: [] };
  }
  function openAdd() { resetModal(); showModal = true; }
  function openEdit(role) {
    modalMode = 'edit'; modalStep = 1; permissionSearch = '';
    collapsedGroups = new Set(permissions.map(p => { let k = p.code.split(':')[0]; return k === 'role' ? 'user' : k; }));
    nameTouched = false; selectedRole = role;
    const currentPermIds = permissions.filter(p => role.permissions.includes(p.code)).map(p => p.id);
    initialPermissionIds = [...currentPermIds];
    initialFormData = { name: role.name, description: role.description };
    form = { name: role.name, description: role.description, permission_ids: currentPermIds };
    showModal = true;
  }
  function openDuplicate(role) {
    resetModal(); modalMode = 'add';
    form.name = `${role.name} (copy)`; form.description = role.description;
    form.permission_ids = [...permissions.filter(p => role.permissions.includes(p.code)).map(p => p.id)];
    showModal = true;
  }
  function proceedToPermissions() {
    nameTouched = true; if (nameErrorText) { toast.error(nameErrorText); return; }
    modalStep = 2;
    const allKeys = new Set(); for (const p of permissions) { let k = p.code.split(':')[0]; if (k === 'role') k = 'user'; allKeys.add(k); }
    collapsedGroups = allKeys;
  }

  function closeAll() { expandedRoleId = null; }

  function requestClose() { if (isFormDirty()) { pendingClose = true; showDiscardModal = true; } else showModal = false; }
  function confirmDiscard() { showDiscardModal = false; showModal = false; pendingClose = false; }
  function cancelDiscard() { showDiscardModal = false; pendingClose = false; }

  async function saveRole() {
    if (modalStep === 1) { proceedToPermissions(); return; }
    nameTouched = true; if (nameErrorText) { toast.error(nameErrorText); return; }
    try {
      saving = true;
      if (modalMode === 'add') {
        const r = await apiFetch('/api/admin/roles', { method: 'POST', body: JSON.stringify({ name: form.name, description: form.description }) });
        if (r.ok) { const newRole = await r.json(); await apiFetch(`/api/admin/roles/${newRole.data.id}/permissions`, { method: 'PUT', body: JSON.stringify({ permission_ids: form.permission_ids }) }); toast.success('Role created'); }
        else { const data = await r.json().catch(() => ({})); throw new Error(data.error || 'Failed to create role'); }
      } else {
        const r = await apiFetch(`/api/admin/roles/${selectedRole.id}`, { method: 'PUT', body: JSON.stringify({ name: form.name, description: form.description }) });
        if (!r.ok) { const data = await r.json().catch(() => ({})); throw new Error(data.error || 'Failed to update role'); }
        const p = await apiFetch(`/api/admin/roles/${selectedRole.id}/permissions`, { method: 'PUT', body: JSON.stringify({ permission_ids: form.permission_ids }) });
        if (p.ok) toast.success('Role updated');
        else { const data = await p.json().catch(() => ({})); throw new Error(data.error || 'Failed to update permissions'); }
      }
      showModal = false; closeAll(); await fetchData();
    } catch (err) { toast.error(err.message || 'Operation failed'); }
    finally { saving = false; }
  }

  async function confirmDelete() {
    if (!selectedRole) return;
    if (selectedRole.is_system) { toast.error('Cannot delete system roles'); return; }
    try {
      const r = await apiFetch(`/api/admin/roles/${selectedRole.id}`, { method: 'DELETE' });
      if (r.ok) { roles = roles.filter(r => r.id !== selectedRole.id); toast.success(`Role "${selectedRole.name}" removed`); if (expandedRoleId === selectedRole.id) expandedRoleId = null; }
      else { const data = await r.json().catch(() => ({})); toast.error(data.error || 'Failed to delete role'); }
    } catch { toast.error('Connection lost. Check your network.'); }
    finally { showDeleteModal = false; selectedRole = null; }
  }

  function togglePermission(id) { if (form.permission_ids.includes(id)) form.permission_ids = form.permission_ids.filter(pid => pid !== id); else form.permission_ids = [...form.permission_ids, id]; }
  function handleModalKeydown(e) { if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) { e.preventDefault(); saveRole(); } }

  function closeActionDropdown() { openActionRoleId = null; }

  function openRoleDrawer(role) {
    selectedRole = role;
    showRoleDrawer = true;
  }

  function closeRoleDrawer() {
    showRoleDrawer = false;
  }

  function handleWindowKeydown(e) {
    if (e.key === 'Escape') {
      showRoleDrawer = false;
      openActionRoleId = null;
      showFilterTypeDropdown = false;
      document.dispatchEvent(new CustomEvent('close-all-dropdowns'));
    }
  }

  function handleDocumentClick(e) {
    if (openActionRoleId && !e.target.closest('.role-action-dropdown')) {
      openActionRoleId = null;
    }
    if (showFilterTypeDropdown && !e.target.closest('.filter-type-container')) {
      showFilterTypeDropdown = false;
    }
  }

  onMount(() => {
    fetchData();
    document.addEventListener('click', handleDocumentClick);
    document.addEventListener('close-all-dropdowns', closeActionDropdown);
    return () => {
      document.removeEventListener('click', handleDocumentClick);
      document.removeEventListener('close-all-dropdowns', closeActionDropdown);
    };
  });
</script>

<div class="space-y-4">
  {#if !canView}
    <div class="card px-4 py-12 text-center">
      <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center"><Shield size={32} class="text-text-muted" /></div>
      <p class="text-text-primary font-semibold mt-4">Access Denied</p>
      <p class="text-text-muted text-sm mt-1">You do not have permission to view roles</p>
    </div>
  {:else}
    <!-- Search & Filter Panel -->
    <div class="card p-4">
      <div class="flex items-center gap-4">
        <div class="flex-1">
          <SearchBar bind:value={roleSearch} placeholder="Search roles…" oninput={() => handleRoleSearch(roleSearch)} inputClass="h-10" />
        </div>
        <div class="relative shrink-0 filter-type-container" style="width: 140px; min-width: 140px; max-width: 140px;">
          <button
            class="flex items-center gap-2 px-3 h-10 w-full rounded-xl border border-border bg-surface-default text-text-secondary text-sm hover:border-border-strong hover:bg-surface-hover transition-colors"
            onclick={() => showFilterTypeDropdown = !showFilterTypeDropdown}
          >
            <span class="flex-1 text-left truncate">{filterType === 'all' ? 'All Roles' : filterType === 'system' ? 'System Only' : 'Custom Only'}</span>
            <ChevronDown size={14} class="text-text-muted shrink-0" />
          </button>
          {#if showFilterTypeDropdown}
            <div class="absolute left-0 top-full mt-2 z-50 bg-surface-default border border-border rounded-lg shadow-xl py-1 min-w-[160px]">
              <button
                class="w-full text-left px-4 py-2 text-sm transition-colors {filterType === 'all' ? 'text-primary-light bg-primary-subtle/30 font-medium' : 'text-text-secondary hover:bg-surface-hover'}"
                onclick={() => { filterType = 'all'; pageOffset = 0; showFilterTypeDropdown = false; }}
              >All Roles</button>
              <button
                class="w-full text-left px-4 py-2 text-sm transition-colors {filterType === 'system' ? 'text-primary-light bg-primary-subtle/30 font-medium' : 'text-text-secondary hover:bg-surface-hover'}"
                onclick={() => { filterType = 'system'; pageOffset = 0; showFilterTypeDropdown = false; }}
              >System Only</button>
              <button
                class="w-full text-left px-4 py-2 text-sm transition-colors {filterType === 'custom' ? 'text-primary-light bg-primary-subtle/30 font-medium' : 'text-text-secondary hover:bg-surface-hover'}"
                onclick={() => { filterType = 'custom'; pageOffset = 0; showFilterTypeDropdown = false; }}
              >Custom Only</button>
            </div>
          {/if}
        </div>
        {#if canEdit}
          <div class="flex items-center gap-2 shrink-0">
            <button title="Refresh" class="btn btn-secondary px-3 h-10" onclick={() => { fetchData(); closeAll(); }} disabled={loading}>
              <RefreshCw size={16} class="{loading ? 'animate-spin' : ''}" />
            </button>
            <button class="btn btn-primary shadow-glow-primary-sm px-5 h-10" onclick={openAdd}>
              <Plus size={16} class="mr-1.5" /> Create Role
            </button>
          </div>
        {/if}
      </div>
      {#if isFiltered}
        <div class="flex items-center gap-2 mt-3 pt-3 border-t border-border-subtle">
          <span class="text-xs text-text-muted">Filters:</span>
          {#if roleSearchDebounced}
            <span class="inline-flex items-center gap-1 rounded-md bg-primary-subtle text-primary-light border border-primary-default/20 px-2 py-0.5 text-xs font-medium">
              Search: "{roleSearchDebounced}"
              <button onclick={() => { roleSearch = ''; handleRoleSearch(''); }} class="hover:text-white transition-colors" aria-label="Clear search filter"><X size={10} /></button>
            </span>
          {/if}
          {#if filterType !== 'all'}
            <span class="inline-flex items-center gap-1 rounded-md bg-primary-subtle text-primary-light border border-primary-default/20 px-2 py-0.5 text-xs font-medium">
              {filterType === 'system' ? 'System' : 'Custom'}
              <button onclick={() => filterType = 'all'} class="hover:text-white transition-colors" aria-label="Clear type filter"><X size={10} /></button>
            </span>
          {/if}
          <button onclick={clearFilters} class="ml-auto text-xs font-medium text-text-muted hover:text-danger transition-colors">Clear all</button>
        </div>
      {/if}
    </div>

    <!-- Error state -->
    {#if loadError}
      <div class="card p-6 border-danger/30 bg-danger-subtle/20">
        <div class="flex items-center gap-3">
          <AlertTriangle size={20} class="text-danger-light shrink-0" />
          <div class="flex-1">
            <p class="text-sm font-medium text-danger-light">{loadError}</p>
            <p class="text-xs text-text-muted mt-0.5">Check your connection and try again.</p>
          </div>
          <button class="btn btn-secondary text-sm px-3 py-1.5" onclick={fetchData}>Retry</button>
        </div>
      </div>
    {/if}

    <!-- Table -->
    {#if loading}
      <div class="card overflow-hidden">
        <table class="w-full table-fixed">
          <thead class="bg-muted/50">
            <tr>
              <th class="text-left p-4 font-semibold w-8"></th>
              <th class="text-left p-4 font-semibold" style="width: 35%;">ROLE</th>
              <th class="text-left p-4 font-semibold w-20">TYPE</th>
              <th class="text-left p-4 font-semibold w-20">PERMISSIONS</th>
              <th class="text-left p-4 font-semibold" style="width: 20%;">DESCRIPTION</th>
              <th class="text-right p-4 font-semibold w-10"></th>
            </tr>
          </thead>
          <tbody>
            {#each Array(5) as _}
              <tr class="border-t border-border">
                <td class="p-4"><Skeleton width="w-4" height="h-4" /></td>
                <td class="p-4"><Skeleton width="w-32" height="h-4" /></td>
                <td class="p-4"><Skeleton width="w-16" height="h-6" rounded="rounded-full" /></td>
                <td class="p-4"><Skeleton width="w-20" height="h-4" /></td>
                <td class="p-4"><Skeleton width="w-40" height="h-4" /></td>
                <td class="p-4"><div class="flex justify-end gap-1"><Skeleton width="w-8" height="h-8" rounded="rounded-lg" /><Skeleton width="w-8" height="h-8" rounded="rounded-lg" /></div></td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {:else if totalFiltered === 0}
      <div class="card px-4 py-12 text-center">
        <div class="empty-state-icon bg-surface w-16 h-16 mx-auto flex justify-center"><Shield size={28} class="text-text-muted" /></div>
        <p class="text-text-primary font-semibold mt-3">{roleSearchDebounced || filterType !== 'all' ? 'No matching roles' : 'No roles defined'}</p>
        <p class="text-text-muted text-sm mt-1 max-w-sm mx-auto">
          {#if roleSearchDebounced}No roles match "<span class="text-text-secondary">{roleSearchDebounced}</span>". Try a different search term.
          {:else if filterType !== 'all'}No {filterType} roles found. Try a different filter.
          {:else}Create your first role to define access control for your team members.{/if}
        </p>
        {#if canEdit}
          {#if roleSearchDebounced || filterType !== 'all'}
            <button class="btn btn-secondary mt-4 text-sm" onclick={clearFilters}>Clear Filters</button>
          {:else}
            <button class="btn btn-primary mt-4 text-sm shadow-glow-primary-sm" onclick={openAdd}><Plus size={14} class="mr-1" /> Create First Role</button>
          {/if}
        {/if}
      </div>
    {:else}
      <div class="card overflow-hidden" role="list">
        <div class="overflow-x-auto">
          <table class="w-full table-fixed border-collapse" style="min-width: 760px;">
          <thead class="bg-muted/50">
            <tr>
              <th class="text-left p-4 font-semibold" style="width: 35%;">
                <button class="flex items-center gap-1 hover:text-primary transition-colors text-xs uppercase tracking-wider" onclick={() => toggleSort('name')}>
                  ROLE <ArrowUpDown size={14} class="text-text-muted" />
                </button>
              </th>
              <th class="text-left p-4 font-semibold w-20 text-xs uppercase tracking-wider">TYPE</th>
              <th class="text-left p-4 font-semibold w-20">
                <button class="flex items-center gap-1 hover:text-primary transition-colors text-xs uppercase tracking-wider" onclick={() => toggleSort('permissions')}>
                  PERMISSIONS <ArrowUpDown size={14} class="text-text-muted" />
                </button>
              </th>
              <th class="text-left p-4 font-semibold" style="width: 20%;">DESCRIPTION</th>
              <th class="text-right p-4 font-semibold w-10"></th>
            </tr>
          </thead>
            <tbody>
              {#each paginatedRoles() as role (role.id)}
                {@const rolePerms = getRolePermissions(role)}
                <tr class="border-t border-border hover:bg-surface-hover/50 transition-colors cursor-pointer" role="listitem" onclick={() => openRoleDrawer(role)} onkeydown={(e) => { if (e.key === 'Enter') openRoleDrawer(role); }} tabindex="0">
                  <td class="p-4">
                    <div class="flex items-center gap-2.5 min-w-0">
                      <div class="w-8 h-8 rounded-full bg-primary-subtle flex items-center justify-center shrink-0"><Shield size={14} class="text-primary-light" /></div>
                      <span class="text-sm font-semibold text-text-primary truncate">{role.name}</span>
                    </div>
                  </td>
                  <td class="p-4">{#if role.is_system}<Badge variant="primary" size="sm">System</Badge>{:else}<Badge variant="muted" size="sm">Custom</Badge>{/if}</td>
                  <td class="p-4"><span class="text-sm text-text-primary">{rolePerms.length} permissions</span></td>
                  <td class="p-4">{#if role.description}<span class="text-sm text-text-primary truncate block max-w-xs" title={role.description}>{role.description}</span>{:else}<span class="text-sm text-text-muted/50 italic">No description</span>{/if}</td>
                  <td class="p-4">
                    <div class="flex items-center justify-end">
                      <div class="relative" onclick={(e) => e.stopPropagation()}>
                        <button
                          onclick={() => { openActionRoleId = openActionRoleId === role.id ? null : role.id; }}
                          class="p-1.5 rounded-lg transition-colors hover:bg-surface-hover text-text-muted hover:text-text-primary"
                          title="Actions"
                          aria-label="Role actions"
                        >
                          <MoreVertical size={14} />
                        </button>
                        {#if openActionRoleId === role.id}
                          <div
                            class="role-action-dropdown absolute right-0 top-full mt-1 w-44 card-glass border border-border rounded-lg shadow-lg z-50 py-1"
                            role="menu"
                            aria-orientation="vertical"
                            tabindex="-1"
                          >
                            {#if canEdit}
                              <button onclick={() => { openActionRoleId = null; closeAll(); openEdit(role); }} class="w-full flex items-center gap-3 px-3 py-2 text-sm text-text-secondary hover:bg-surface-hover transition-colors" role="menuitem">
                                <Pencil size={14} /> Edit
                              </button>
                              <button onclick={() => { openActionRoleId = null; closeAll(); openDuplicate(role); }} class="w-full flex items-center gap-3 px-3 py-2 text-sm text-text-secondary hover:bg-surface-hover transition-colors" role="menuitem">
                                <Copy size={14} /> Duplicate
                              </button>
                            {/if}
                            {#if canDelete && !role.is_system}
                              <button onclick={() => { openActionRoleId = null; selectedRole = role; showDeleteModal = true; }} class="w-full flex items-center gap-3 px-3 py-2 text-sm text-danger hover:bg-danger-subtle transition-colors" role="menuitem">
                                <Trash2 size={14} /> Delete
                              </button>
                            {/if}
                          </div>
                        {/if}
                      </div>
                    </div>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
        {#if totalFiltered > 0}
          <div class="px-4 py-3 bg-surface-subtle/30 border-t border-border/50">
            <Pagination total={totalFiltered} limit={pageLimit} offset={pageOffset} onPageChange={handlePageChange} />
          </div>
        {/if}
      </div>
    {/if}
  {/if}
</div>

<svelte:window onkeydown={handleWindowKeydown} />

<!-- Create/Edit Modal -->
<Modal bind:open={showModal} title={modalStep === 1 ? (modalMode === 'add' ? 'Create Role — Step 1 of 2' : 'Edit Role — Step 1 of 2') : (modalMode === 'add' ? 'Create Role — Step 2 of 2' : 'Edit Role — Step 2 of 2')} size="lg" onkeydown={handleModalKeydown}>

<div class="space-y-4">
    {#if modalStep === 1}
      <div>
        <label for="role-name" class="block text-sm font-medium text-text-secondary mb-1.5">Role Name <span class="text-danger-light">*</span></label>
        <input id="role-name" type="text" placeholder="e.g. manager" class="input" class:border-danger={nameErrorText} bind:value={form.name} onblur={() => nameTouched = true} aria-invalid={!!nameErrorText} aria-describedby={nameErrorText ? 'role-name-error' : undefined} autofocus />
        {#if nameErrorText}<p id="role-name-error" class="text-xs text-danger mt-1.5" role="alert">{nameErrorText}</p>{/if}
      </div>
      <div>
        <label for="role-desc" class="block text-sm font-medium text-text-secondary mb-1.5">Description</label>
        <input id="role-desc" type="text" placeholder="Short description of this role" class="input" bind:value={form.description} />
        <p class="text-xs text-text-muted mt-1">Optional. Helps identify the role's purpose.</p>
      </div>
    {:else}
      {#if modalMode === 'edit'}
        <div class="flex items-center gap-3 p-3 bg-surface-subtle rounded-xl border border-border-subtle">
<div class="w-8 h-8 rounded-full bg-primary-subtle flex items-center justify-center shrink-0"><Shield size={16} class="text-primary-light" /></div>
          <div class="min-w-0"><p class="text-sm font-semibold text-text-primary truncate">{selectedRole?.name}</p><p class="text-xs text-text-muted truncate">{selectedRole?.description || 'No description'}</p></div>
        </div>
      {/if}
      {#if isFormDirty()}<div class="flex items-center gap-2 px-2.5 py-1.5 rounded-lg bg-warning-subtle/30 border border-warning/20"><div class="w-2 h-2 rounded-full bg-warning animate-pulse"></div><span class="text-xs text-warning-light">You have unsaved changes</span></div>{/if}
      <div class="border-t border-border pt-4 mt-1">
        <div class="flex items-center justify-between mb-2">
          <p class="text-sm font-medium text-text-secondary">Permissions</p>
          <div class="flex items-center gap-2">
            <span class="text-xs text-text-muted">{form.permission_ids.length} of {permissions.length} selected</span>
            {#if groupedPermissions().length > 1}<button type="button" class="text-xs text-primary hover:text-primary-light transition-colors" onclick={() => setAllExpanded(!allExpanded())}>{allExpanded() ? 'Collapse All' : 'Expand All'}</button>{/if}
          </div>
        </div>
        <div class="mb-3">
          <SearchBar bind:value={permissionSearch} placeholder="Search permissions…" inputClass="text-sm" />
        </div>
        <div class="space-y-2 max-h-80 overflow-y-auto">
          {#if groupedPermissions().length > 0}
            {#each groupedPermissions() as group (group.key)}
              {@const isCollapsed = collapsedGroups.has(group.key)} {@const Icon = group.icon} {@const allSel = isGroupAllSelected(group)} {@const partialSel = isGroupPartialSelected(group)}
              <div class="rounded-xl border border-border/50 overflow-hidden" data-group>
                <div class="flex items-center gap-2 px-3 py-2.5 bg-surface-subtle/40 hover:bg-surface-subtle/60 transition-colors">
                  <button type="button" class="flex items-center gap-2 flex-1 text-left min-w-0" aria-expanded={!isCollapsed} aria-controls="group-body-{group.key}" aria-label="Toggle {group.label} permissions" onclick={() => toggleGroup(group.key)} onkeydown={(e) => handleGroupKeydown(e, group)} data-group-toggle data-group-key={group.key}>
                    {#if isCollapsed}<ChevronRight size={14} class="text-text-muted shrink-0" />{:else}<ChevronDown size={14} class="text-text-muted shrink-0" />{/if}
                    <Icon size={14} class="text-primary-light shrink-0" /><span class="text-sm font-medium text-text-primary truncate">{group.label}</span><span class="text-xs text-text-muted shrink-0">({group.permissions.length})</span>
                  </button>
                  <button type="button" class="text-[10px] font-semibold shrink-0 px-2 py-1 rounded-md transition-colors {allSel ? 'bg-primary/15 text-primary border border-primary/20' : partialSel ? 'bg-warning/15 text-warning-light border border-warning/20' : 'bg-surface-default text-text-muted border border-border/50 hover:border-border-strong hover:text-text-primary'}" onclick={() => toggleGroupAll(group)} aria-label={allSel ? 'Deselect all ' + group.label + ' permissions' : 'Select all ' + group.label + ' permissions'} aria-pressed={allSel}>
                    {#if partialSel}<ChevronsUpDown size={10} />{:else if allSel}<Check size={10} />{:else}All{/if}
                  </button>
                  <span class="inline-flex items-center text-[10px] font-semibold px-1.5 py-0.5 rounded-md bg-primary/10 text-primary shrink-0 min-w-[2.5rem] justify-center">{group.permissions.filter(p => form.permission_ids.includes(p.id)).length}/{group.permissions.length}</span>
                </div>
                {#if !isCollapsed}
                  <div id="group-body-{group.key}" class="grid grid-cols-1 sm:grid-cols-2 gap-1.5 p-3 bg-surface/20" role="group" aria-label={group.label + ' permissions'}>
                    {#each group.permissions as perm (perm.id)}
                      <label class="flex items-center gap-2.5 px-3 py-2 rounded-lg border cursor-pointer transition-colors {form.permission_ids.includes(perm.id) ? 'border-primary/40 bg-primary-subtle/30' : 'border-transparent hover:border-border hover:bg-surface-default'}" title={perm.description || ''}>
                        <input type="checkbox" id="perm-{perm.id}" class="w-4 h-4 accent-primary rounded" checked={form.permission_ids.includes(perm.id)} onchange={() => togglePermission(perm.id)} aria-describedby="perm-desc-{perm.id}" />
                        <div class="min-w-0 flex-1"><p class="text-sm font-medium text-text-primary truncate">{perm.name}</p><p id="perm-desc-{perm.id}" class="text-[10px] text-text-muted font-mono truncate">{perm.code}</p></div>
                      </label>
                    {/each}
                  </div>
                {/if}
              </div>
            {/each}
          {:else}
            <div class="py-6 text-center"><p class="text-sm text-text-muted italic">No permissions match "{permissionSearch}"</p></div>
          {/if}
        </div>
        {#if form.permission_ids.length > 0}
          <div class="flex items-center justify-between px-3 py-2 mt-2 rounded-lg bg-surface-subtle/40 border border-border/30">
            <span class="text-xs text-text-secondary"><span class="font-semibold text-text-primary">{form.permission_ids.length}</span> permissions selected {#if selectedGroupCount() > 0}<span class="text-text-muted"> across {selectedGroupCount()} {selectedGroupCount() === 1 ? 'group' : 'groups'}</span>{/if}</span>
            <button type="button" class="text-xs text-danger hover:text-danger-light transition-colors" onclick={() => form.permission_ids = []}>Clear all</button>
          </div>
        {/if}
      </div>
    {/if}
  </div>
  {#snippet footer()}
    <div class="flex items-center justify-between w-full">
      <div>{#if modalStep === 2}<button class="btn btn-ghost text-sm" onclick={() => modalStep = 1} disabled={saving}>← Back</button>{/if}</div>
      <div class="flex items-center gap-2">
        <button class="btn btn-secondary" onclick={requestClose} disabled={saving}>Cancel</button>
        {#if modalStep === 1}
          <button class="btn btn-primary min-w-28" onclick={proceedToPermissions} disabled={!form.name.trim()}>Next →</button>
        {:else}
          <button class="btn btn-primary min-w-32" onclick={saveRole} disabled={saving || !!nameErrorText} aria-busy={saving}>
            {#if saving}<Loader2 size={16} class="animate-spin" /> Saving...{:else}{modalMode === 'add' ? 'Create Role' : 'Save Changes'}{/if}
          </button>
        {/if}
      </div>
    </div>
    {#if modalStep === 2}<div class="text-[10px] text-text-muted text-center mt-1 w-full">Ctrl+Enter to save</div>{/if}
  {/snippet}
</Modal>

<!-- Discard Confirmation -->
<Modal bind:open={showDiscardModal} title="Discard Changes?" size="sm">
  <div class="text-center py-2">
    <div class="w-14 h-14 rounded-2xl bg-warning-subtle flex items-center justify-center mx-auto mb-4"><Eye size={24} class="text-warning" /></div>
    <p class="text-text-primary font-semibold mb-1">You have unsaved changes</p>
    <p class="text-text-muted text-sm">Your permission selections will be lost if you close without saving.</p>
  </div>
  {#snippet footer()}<button class="btn btn-secondary" onclick={cancelDiscard}>Keep Editing</button><button class="btn btn-danger" onclick={confirmDiscard}>Discard</button>{/snippet}
</Modal>

<!-- Delete Confirmation -->
<Modal bind:open={showDeleteModal} title="Delete Role" size="sm">
  <div class="text-center py-2">
    <div class="w-14 h-14 rounded-2xl bg-danger-subtle flex items-center justify-center mx-auto mb-4"><Trash2 size={24} class="text-danger" /></div>
    <p class="text-text-primary font-semibold mb-1">Delete role "{selectedRole?.name}"?</p>
    <p class="text-text-muted text-sm">Users with this role will lose their assigned permissions. This action cannot be undone.</p>
  </div>
  {#snippet footer()}<button class="btn btn-secondary" onclick={() => { showDeleteModal = false; selectedRole = null; }}>Cancel</button><button class="btn btn-danger" onclick={confirmDelete}>Delete</button>{/snippet}
</Modal>

<!-- Role Detail Drawer -->
{#if showRoleDrawer && selectedRole}
  {@const rolePerms = getRolePermissions(selectedRole)}
  {@const grouped = getGroupedPermissions(rolePerms)}
  <div class="fixed inset-0 bg-black/60 z-50" onclick={closeRoleDrawer} aria-hidden="true"></div>
  <div
    class="fixed inset-y-0 right-0 w-[520px] max-w-full bg-surface-default border-l border-border shadow-2xl z-[55] flex flex-col transition-transform duration-300 ease-out"
    transition:fly={{ x: 520, duration: 300, easing: t => t * (2 - t) }}
    role="dialog" aria-modal="true" tabindex="-1"
    onkeydown={(e) => { if (e.key === 'Escape') { e.preventDefault(); closeRoleDrawer(); } }}
  >
    <div class="flex items-center justify-between px-6 py-5 border-b border-border shrink-0">
      <div class="flex items-center gap-3">
        <div class="w-9 h-9 rounded-lg bg-primary-subtle flex items-center justify-center shrink-0"><Shield size={16} class="text-primary-light" /></div>
        <h2 class="text-lg font-bold text-text-primary">{selectedRole.name}</h2>
        {#if selectedRole.is_system}<Badge variant="primary" size="sm">System</Badge>{:else}<Badge variant="muted" size="sm">Custom</Badge>{/if}
      </div>
      <button
        class="p-2 rounded-lg text-text-muted hover:bg-surface-hover hover:text-text-secondary transition-colors"
        onclick={closeRoleDrawer}
        onkeydown={(e) => { if (e.key === 'Enter' || e.key === 'Escape' || e.key === ' ') { e.preventDefault(); closeRoleDrawer(); } }}
        title="Close" aria-label="Close detail panel"
      >
        <X size={18} />
      </button>
    </div>

    <div class="flex-1 overflow-y-auto px-6 py-4 pb-28 space-y-4">
      {#if selectedRole.description}
        <div class="rounded-2xl bg-surface-default border border-border p-4">
          <p class="text-xs font-semibold text-text-muted uppercase tracking-wider mb-1.5">Description</p>
          <p class="text-sm text-text-secondary leading-relaxed">{selectedRole.description}</p>
        </div>
      {/if}

      <div class="rounded-2xl bg-surface-default border border-border overflow-hidden">
        <div class="px-4 py-2.5 border-b border-border/60 flex items-center gap-2">
          <Users size={14} class="text-text-muted" />
          <h3 class="text-xs font-semibold uppercase tracking-wide text-text-muted/80">Permissions ({rolePerms.length})</h3>
        </div>
        {#if rolePerms.length > 0}
          <div class="p-4 grid grid-cols-1 gap-3">
            {#each grouped as [key, perms]}
              <div class="rounded-lg border border-border/40 bg-surface-subtle/20 p-3">
                <p class="text-xs font-semibold text-primary-light uppercase tracking-wider mb-2">{groupMeta[key]?.label || key}</p>
                <div class="flex flex-wrap gap-1.5">
                  {#each perms as perm}
                    <span class="inline-flex items-center text-[11px] font-medium px-2 py-0.5 rounded-md bg-surface-default/80 text-text-secondary border border-border/30" title={perm.description || perm.code}>{perm.name}</span>
                  {/each}
                </div>
              </div>
            {/each}
          </div>
        {:else}
          <div class="p-4">
            <p class="text-sm text-text-muted italic">No permissions assigned</p>
          </div>
        {/if}
      </div>
    </div>

    {#if canEdit || canDelete}
      <div class="absolute bottom-0 left-0 right-0 p-4 bg-surface-default border-t border-border/50">
        <div class="flex items-center gap-3">
          {#if canEdit}
            <button
              class="flex-1 btn btn-secondary rounded-xl px-4 h-11 text-sm font-semibold text-text-secondary border border-border hover:border-primary hover:text-primary hover:bg-primary-subtle transition-all duration-200"
              onclick={() => { showRoleDrawer = false; closeAll(); openDuplicate(selectedRole); }}
            >
              <Copy size={15} class="mr-1.5" />Duplicate
            </button>
            <button
              class="flex-1 btn btn-primary rounded-xl px-4 h-11 text-sm font-semibold text-white shadow-glow-primary-sm transition-all duration-200"
              onclick={() => { showRoleDrawer = false; closeAll(); openEdit(selectedRole); }}
            >
              <Pencil size={15} class="mr-1.5" />Edit
            </button>
          {/if}
          {#if canDelete && !selectedRole.is_system}
            <button
              class="btn btn-danger rounded-xl px-4 h-11 text-sm font-semibold transition-all duration-200"
              onclick={() => { showRoleDrawer = false; showDeleteModal = true; }}
            >
              <Trash2 size={15} class="mr-1.5" />Delete
            </button>
          {/if}
        </div>
      </div>
    {/if}
  </div>
{/if}
