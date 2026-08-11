<script>
  import { onMount } from 'svelte';
  import { apiFetch } from '$shared/api/http-client';
  import RoleDetailDrawer from './RoleDetailDrawer.svelte';
  import { toast } from '$shared/stores/toast.svelte';
  import { useAuthStore } from '$modules/auth';
  import { useRBAC } from '$shared/composables/useRBAC.svelte';
  import { Permissions } from '$shared/constants/permissions';
  import { labels } from '$shared/i18n';
  import { groupPermissions, permissionGroupKey } from '$shared/utils/permissionGroups';

  import { Badge, Button, Dropdown, Input, Modal, Pagination, SearchBar, Skeleton, ConfirmDeleteModal, SortableHeader } from '$shared/ui';
  import { Plus, Pencil, Trash2, Shield, Loader2, ChevronRight, ChevronDown, ChevronsUpDown, Check, Eye, RefreshCw, Copy, AlertTriangle, MoreVertical } from 'lucide-svelte';

  const authStore = useAuthStore();
  const rbac = useRBAC();

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
  let filteredRoles = $derived.by(() => {
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
  let paginatedRoles = $derived.by(() => {
    const all = filteredRoles;
    return all.slice(pageOffset, pageOffset + pageLimit);
  });

  let totalFiltered = $derived(filteredRoles.length);

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

  // ── Detail panel ────────────────────────────────────────────────
  function toggleDetail(roleId) {
    expandedRoleId = expandedRoleId === roleId ? null : roleId;
  }

  function getRolePermissions(role) {
    if (!role.permissions || !role.permissions.length) return [];
    const permCodes = new Set(role.permissions);
    return permissions.filter(p => permCodes.has(p.code));
  }

  // ── Permissions grouping ────────────────────────────────────────
  let collapsedGroups = $state(new Set());
  let initialPermissionIds = $state([]);
  let initialFormData = $state({ name: '', description: '' });

  let groupedPermissions = $derived.by(() => {
    const filtered = permissionSearch.trim()
      ? permissions.filter(p =>
          p.name.toLowerCase().includes(permissionSearch.toLowerCase()) ||
          p.code.toLowerCase().includes(permissionSearch.toLowerCase())
        )
      : permissions;
    return groupPermissions(filtered);
  });

  let selectedGroupCount = $derived.by(() => {
    const uniqueGroups = new Set();
    for (const gid of form.permission_ids) {
      const p = permissions.find(pp => pp.id === gid);
      if (p) uniqueGroups.add(permissionGroupKey(p.code));
    }
    return uniqueGroups.size;
  });

  // ── Validation ──────────────────────────────────────────────────
  let nameError = $derived.by(() => {
    if (!nameTouched && modalStep === 1) return '';
    if (!form.name.trim()) return 'Role name is required';
    if (roles.some(r => r.name.toLowerCase() === form.name.trim().toLowerCase() && r.id !== selectedRole?.id)) {
      return 'Role name already exists';
    }
    return '';
  });
  let nameErrorText = $derived(nameError);

  let hasUnsavedChanges = $derived.by(() => {
    if (modalMode === 'add') return form.permission_ids.length > 0 || form.name.trim() !== '' || form.description.trim() !== '';
    const permsCurrent = JSON.stringify([...form.permission_ids].sort());
    const permsInitial = JSON.stringify([...initialPermissionIds].sort());
    if (permsCurrent !== permsInitial) return true;
    return form.name !== initialFormData.name || form.description !== initialFormData.description;
  });

  let isFormDirty = $derived.by(() => {
    if (modalMode === 'add') return form.name.trim() !== '' || form.permission_ids.length > 0;
    return hasUnsavedChanges;
  });

  let canView = $derived(rbac.can(Permissions.role.view));
  let canEdit = $derived(rbac.can(Permissions.role.update));
  let canDelete = $derived(rbac.can(Permissions.role.delete));

  function allExpanded() { return groupedPermissions.every(g => !collapsedGroups.has(g.key)); }
  function toggleGroup(key) { const s = new Set(collapsedGroups); if (s.has(key)) s.delete(key); else s.add(key); collapsedGroups = s; }
  function setAllExpanded(expanded) { collapsedGroups = expanded ? new Set() : new Set(groupedPermissions.map(g => g.key)); }

  function toggleGroupAll(group) {
    const allSel = group.permissions.every(p => form.permission_ids.includes(p.id));
    if (allSel) form.permission_ids = form.permission_ids.filter(id => !group.permissions.some(p => p.id === id));
    else form.permission_ids = [...form.permission_ids, ...group.permissions.filter(p => !form.permission_ids.includes(p.id)).map(p => p.id)];
  }
  function isGroupAllSelected(group) { return group.permissions.every(p => form.permission_ids.includes(p.id)); }
  function isGroupPartialSelected(group) { return group.permissions.some(p => form.permission_ids.includes(p.id)) && !isGroupAllSelected(group); }

  function handleGroupKeydown(e, group) {
    if (e.key === 'ArrowDown') { e.preventDefault(); const groups = groupedPermissions; const idx = groups.findIndex(g => g.key === group.key); if (idx < groups.length - 1) document.querySelector(`[data-group-toggle][data-group-key="${groups[idx + 1].key}"]`)?.focus(); }
    else if (e.key === 'ArrowUp') { e.preventDefault(); const groups = groupedPermissions; const idx = groups.findIndex(g => g.key === group.key); if (idx > 0) document.querySelector(`[data-group-toggle][data-group-key="${groups[idx - 1].key}"]`)?.focus(); }
  }

  let form = $state({ name: '', description: '', permission_ids: [] });

  async function fetchData() {
    try {
      loading = true;
      loadError = '';
      const [rRes, pRes] = await Promise.all([apiFetch('/api/admin/roles'), apiFetch('/api/admin/permissions')]);
      if (rRes.ok) { const data = await rRes.json(); roles = data.data || []; }
      if (pRes.ok) { const data = await pRes.json(); permissions = data.data || []; }
    } catch { loadError = labels.failedToLoad; toast.error(loadError); }
    finally { loading = false; }
  }

  function resetModal() {
    modalMode = 'add'; modalStep = 1; permissionSearch = '';
    collapsedGroups = new Set(permissions.map(p => permissionGroupKey(p.code)));
    nameTouched = false; initialPermissionIds = []; initialFormData = { name: '', description: '' };
    form = { name: '', description: '', permission_ids: [] };
  }
  function openAdd() { resetModal(); showModal = true; }
  function openEdit(role) {
    modalMode = 'edit'; modalStep = 1; permissionSearch = '';
    collapsedGroups = new Set(permissions.map(p => permissionGroupKey(p.code)));
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
    collapsedGroups = new Set(permissions.map(p => permissionGroupKey(p.code)));
  }

  function closeAll() { expandedRoleId = null; }

  function requestClose() { if (isFormDirty) { pendingClose = true; showDiscardModal = true; } else showModal = false; }
  function confirmDiscard() { showDiscardModal = false; showModal = false; pendingClose = false; }
  function cancelDiscard() { showDiscardModal = false; pendingClose = false; }

  async function saveRole() {
    if (modalStep === 1) { proceedToPermissions(); return; }
    nameTouched = true; if (nameErrorText) { toast.error(nameErrorText); return; }
    try {
      saving = true;
      if (modalMode === 'add') {
        const r = await apiFetch('/api/admin/roles', { method: 'POST', body: JSON.stringify({ name: form.name, description: form.description }) });
        if (r.ok) { const newRole = await r.json(); await apiFetch(`/api/admin/roles/${newRole.data.id}/permissions`, { method: 'PUT', body: JSON.stringify({ permission_ids: form.permission_ids }) }); toast.success(`${labels.role} ${labels.actionCreated}`); }
        else { const data = await r.json().catch(() => ({})); throw new Error(data.error || labels.errorOccurred); }
      } else {
        const r = await apiFetch(`/api/admin/roles/${selectedRole.id}`, { method: 'PUT', body: JSON.stringify({ name: form.name, description: form.description }) });
        if (!r.ok) { const data = await r.json().catch(() => ({})); throw new Error(data.error || labels.errorOccurred); }
        const p = await apiFetch(`/api/admin/roles/${selectedRole.id}/permissions`, { method: 'PUT', body: JSON.stringify({ permission_ids: form.permission_ids }) });
        if (p.ok) toast.success(`${labels.role} ${labels.actionUpdated}`);
        else { const data = await p.json().catch(() => ({})); throw new Error(data.error || labels.errorOccurred); }
      }
      showModal = false; closeAll(); await fetchData();
    } catch (err) { toast.error(err.message || labels.errorOccurred); }
    finally { saving = false; }
  }

  async function confirmDelete() {
    if (!selectedRole) return;
    if (selectedRole.is_system) { toast.error('Cannot delete system roles'); return; }
    try {
      const r = await apiFetch(`/api/admin/roles/${selectedRole.id}`, { method: 'DELETE' });
      if (r.ok) { roles = roles.filter(r => r.id !== selectedRole.id); toast.success(`${labels.role} "${selectedRole.name}" ${labels.actionDeleted}`); if (expandedRoleId === selectedRole.id) expandedRoleId = null; }
      else { const data = await r.json().catch(() => ({})); toast.error(data.error || labels.errorOccurred); }
    } catch { toast.error(labels.networkError); }
    finally { showDeleteModal = false; selectedRole = null; }
  }

  function togglePermission(id) { if (form.permission_ids.includes(id)) form.permission_ids = form.permission_ids.filter(pid => pid !== id); else form.permission_ids = [...form.permission_ids, id]; }
  function handleModalKeydown(e) { if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) { e.preventDefault(); saveRole(); } }

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
    }
  }

  onMount(() => {
    fetchData();
  });
</script>

<div class="space-y-4">
  {#if !canView}
    <div class="card px-4 py-12 text-center">
      <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center"><Shield size={32} class="text-text-muted" /></div>
      <p class="text-text-primary font-semibold mt-4">{labels.accessDenied}</p>
      <p class="text-text-muted text-sm mt-1">{labels.youDoNotHavePermissionToViewRoles}</p>
    </div>
  {:else}
    <!-- Search & Filter Panel -->
    <div class="card p-4">
      <div class="flex items-center gap-4">
        <div class="flex-1">
          <SearchBar bind:value={roleSearch} placeholder={labels.searchRoles} oninput={() => handleRoleSearch(roleSearch)} inputClass="h-10" />
        </div>
        <div class="flex items-center p-1 gap-1 bg-bg-secondary rounded-xl border border-border-default shrink-0">
          <button
            class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {filterType === 'all' ? 'bg-primary-subtle text-primary-light border border-primary-default/20' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
            onclick={() => { filterType = 'all'; pageOffset = 0; }}
          >{labels.allRoles}</button>
          <button
            class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {filterType === 'system' ? 'bg-primary-subtle text-primary-light border border-primary-default/20' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
            onclick={() => { filterType = 'system'; pageOffset = 0; }}
          >{labels.system}</button>
          <button
            class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {filterType === 'custom' ? 'bg-primary-subtle text-primary-light border border-primary-default/20' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
            onclick={() => { filterType = 'custom'; pageOffset = 0; }}
          >{labels.custom}</button>
        </div>
        {#if canEdit}
          <div class="flex items-center gap-2 shrink-0">
            <Button title={labels.refresh} variant="secondary" class="px-3 h-10" onclick={() => { fetchData(); closeAll(); }} disabled={loading}>
              <RefreshCw size={16} class={loading ? 'animate-spin' : ''} />
            </Button>
            <Button variant="primary" class="shadow-glow-primary-sm px-5 h-10" onclick={openAdd}>
              <Plus size={16} class="mr-1.5" /> {labels.create} {labels.role}
            </Button>
          </div>
        {/if}
      </div>

    </div>

    <!-- Error state -->
    {#if loadError}
      <div class="card p-6 border-danger/30 bg-danger-subtle/20">
        <div class="flex items-center gap-3">
          <AlertTriangle size={20} class="text-danger-light shrink-0" />
          <div class="flex-1">
            <p class="text-sm font-medium text-danger-light">{loadError}</p>
            <p class="text-xs text-text-muted mt-0.5">{labels.networkError}</p>
          </div>
          <Button variant="secondary" class="text-sm px-3 py-1.5" onclick={fetchData}>{labels.retry}</Button>
        </div>
      </div>
    {/if}

    <!-- Table -->
    {#if loading}
      <div class="card overflow-hidden" aria-busy="true" aria-label={labels.loading}>
        <table class="w-full table-fixed">
          <thead class="bg-muted/50">
            <tr>
              <th class="text-left p-4 font-semibold w-8"></th>
              <th class="text-left p-4 font-semibold" style="width: 35%;">{labels.roleLabel}</th>
              <th class="text-left p-4 font-semibold w-20">{labels.type}</th>
              <th class="text-left p-4 font-semibold w-20 text-xs uppercase tracking-wider">{labels.permissions}</th>
              <th class="text-left p-4 font-semibold" style="width: 20%;">{labels.descriptionLabel}</th>
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
      <div class="card px-4 py-12 text-center" role="status">
        <div class="empty-state-icon bg-surface w-16 h-16 mx-auto flex justify-center"><Shield size={28} class="text-text-muted" /></div>
        <p class="text-text-primary font-semibold mt-3">{labels.noRolesFound}</p>
        <p class="text-text-muted text-sm mt-1 max-w-sm mx-auto">
          {#if roleSearchDebounced}No roles match "<span class="text-text-secondary">{roleSearchDebounced}</span>". Try a different search term.
          {:else if filterType !== 'all'}No {filterType} roles found. Try a different filter.
          {:else}Create your first role to define access control for your team members.{/if}
        </p>
        {#if canEdit}
          {#if roleSearchDebounced || filterType !== 'all'}
            <Button variant="secondary" class="mt-4 text-sm" onclick={clearFilters}>{labels.clearFilters}</Button>
          {:else}
            <Button variant="primary" class="mt-4 text-sm shadow-glow-primary-sm" onclick={openAdd}><Plus size={14} class="mr-1" /> {labels.create} {labels.role}</Button>
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
                <SortableHeader label={labels.roleLabel} column="name" sortColumn={sortField} sortDirection={sortDir} onsort={toggleSort} />
              </th>
              <th class="text-left p-4 font-semibold w-20 text-xs uppercase tracking-wider">{labels.type}</th>
              <th class="text-left p-4 font-semibold w-20 text-xs uppercase tracking-wider">
                <SortableHeader label={labels.permissions} column="permissions" sortColumn={sortField} sortDirection={sortDir} onsort={toggleSort} />
              </th>
              <th class="text-left p-4 font-semibold" style="width: 20%;">{labels.descriptionLabel}</th>
              <th class="text-right p-4 font-semibold w-10"></th>
            </tr>
          </thead>
            <tbody>
              {#each paginatedRoles as role (role.id)}
                {@const rolePerms = getRolePermissions(role)}
                <tr class="border-t border-border hover:bg-surface-hover/50 transition-colors cursor-pointer" onclick={() => openRoleDrawer(role)} onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); openRoleDrawer(role); } }} tabindex="0" role="button">
                  <td class="p-4">
                    <div class="flex items-center gap-2.5 min-w-0">
                      <div class="w-8 h-8 rounded-full bg-primary-subtle flex items-center justify-center shrink-0"><Shield size={14} class="text-primary-light" /></div>
                      <span class="text-sm font-semibold text-text-primary truncate">{role.name}</span>
                    </div>
                  </td>
                  <td class="p-4">{#if role.is_system}<Badge variant="primary" size="sm">{labels.system}</Badge>{:else}<Badge variant="muted" size="sm">{labels.custom}</Badge>{/if}</td>
                  <td class="p-4"><span class="text-sm text-text-primary">{rolePerms.length} {labels.permissions}</span></td>
                  <td class="p-4">{#if role.description}<span class="text-sm text-text-primary truncate block max-w-xs" title={role.description}>{role.description}</span>{:else}<span class="text-sm text-text-muted/50 italic">{labels.noDescription}</span>{/if}</td>
                  <td class="p-4">
                    <div class="flex items-center justify-end">
                      <Dropdown>
                        {#snippet trigger({ toggle })}
                          <button
                            onclick={(e) => { e.stopPropagation(); toggle(); }}
                            class="p-1.5 rounded-lg transition-colors hover:bg-surface-hover text-text-muted hover:text-text-primary"
                            title={labels.action}
                            aria-label={labels.roleActions}
                          >
                            <MoreVertical size={14} />
                          </button>
                        {/snippet}
                        {#snippet content({ close })}
                          {#if canEdit}
                            <button type="button" onclick={() => { closeAll(); openEdit(role); close(); }} class="w-full flex items-center gap-3 px-3 py-2 text-sm text-text-secondary hover:bg-surface-hover transition-colors" role="menuitem">
                              <Pencil size={14} /> {labels.edit}
                            </button>
                            <button type="button" onclick={() => { closeAll(); openDuplicate(role); close(); }} class="w-full flex items-center gap-3 px-3 py-2 text-sm text-text-secondary hover:bg-surface-hover transition-colors" role="menuitem">
                              <Copy size={14} /> {labels.duplicate}
                            </button>
                          {/if}
                          {#if canDelete && !role.is_system}
                            <button type="button" onclick={() => { selectedRole = role; showDeleteModal = true; close(); }} class="w-full flex items-center gap-3 px-3 py-2 text-sm text-danger hover:bg-danger-subtle transition-colors" role="menuitem">
                              <Trash2 size={14} /> {labels.delete}
                            </button>
                          {/if}
                        {/snippet}
                      </Dropdown>
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
<Modal bind:open={showModal} title={`${modalMode === 'add' ? `${labels.create} ${labels.role}` : `${labels.edit} ${labels.role}`} — ${modalStep} ${labels.of} 2`} size="lg" onkeydown={handleModalKeydown}>

<div class="space-y-4">
    {#if modalStep === 1}
      <div>
        <label for="role-name" class="block text-sm font-medium text-text-secondary mb-1.5">{labels.roleName} <span class="text-danger-light">*</span></label>
        <Input id="role-name" type="text" placeholder="e.g. manager" class={nameErrorText ? 'border-danger' : ''} bind:value={form.name} onblur={() => nameTouched = true} aria-invalid={!!nameErrorText} aria-describedby={nameErrorText ? 'role-name-error' : undefined} />
        {#if nameErrorText}<p id="role-name-error" class="text-xs text-danger mt-1.5" role="alert">{nameErrorText}</p>{/if}
      </div>
      <div>
        <label for="role-desc" class="block text-sm font-medium text-text-secondary mb-1.5">{labels.description}</label>
        <Input id="role-desc" type="text" placeholder={labels.shortDescriptionOfRole} bind:value={form.description} />
        <p class="text-xs text-text-muted mt-1">{labels.optionalRolePurpose}</p>
      </div>
    {:else}
      {#if modalMode === 'edit'}
        <div class="flex items-center gap-3 p-3 bg-surface-subtle rounded-xl border border-border-subtle">
<div class="w-8 h-8 rounded-full bg-primary-subtle flex items-center justify-center shrink-0"><Shield size={16} class="text-primary-light" /></div>
          <div class="min-w-0"><p class="text-sm font-semibold text-text-primary truncate">{selectedRole?.name}</p><p class="text-xs text-text-muted truncate">{selectedRole?.description || labels.noDescription}</p></div>
        </div>
      {/if}
      {#if isFormDirty}<div class="flex items-center gap-2 px-2.5 py-1.5 rounded-lg bg-warning-subtle/30 border border-warning/20"><div class="w-2 h-2 rounded-full bg-warning animate-pulse"></div><span class="text-xs text-warning-light">{labels.youHaveUnsavedChanges}</span></div>{/if}
      <div class="border-t border-border pt-4 mt-1">
        <div class="flex items-center justify-between mb-2">
          <p class="text-sm font-medium text-text-secondary">{labels.permissions}</p>
          <div class="flex items-center gap-2">
            <span class="text-xs text-text-muted">{form.permission_ids.length} {labels.of} {permissions.length}</span>
            {#if groupedPermissions.length > 1}<button type="button" class="text-xs text-primary hover:text-primary-light transition-colors" onclick={() => setAllExpanded(!allExpanded())}>{allExpanded() ? 'Collapse All' : 'Expand All'}</button>{/if}
          </div>
        </div>
        <div class="mb-3">
          <SearchBar bind:value={permissionSearch} placeholder={labels.searchPermissions} inputClass="text-sm" />
        </div>
        <div class="space-y-2 max-h-80 overflow-y-auto">
          {#if groupedPermissions.length > 0}
            {#each groupedPermissions as group (group.key)}
              {@const isCollapsed = collapsedGroups.has(group.key)} {@const Icon = group.icon} {@const allSel = isGroupAllSelected(group)} {@const partialSel = isGroupPartialSelected(group)}
              <div class="rounded-xl border border-border/50 overflow-hidden" data-group>
                <div class="flex items-center gap-2 px-3 py-2.5 bg-surface-subtle/40 hover:bg-surface-subtle/60 transition-colors">
                  <button type="button" class="flex items-center gap-2 flex-1 text-left min-w-0" aria-expanded={!isCollapsed} aria-controls="group-body-{group.key}" aria-label={labels.togglePermissions.replace('{group.label}', group.label)} onclick={() => toggleGroup(group.key)} onkeydown={(e) => handleGroupKeydown(e, group)} data-group-toggle data-group-key={group.key}>
                    {#if isCollapsed}<ChevronRight size={14} class="text-text-muted shrink-0" />{:else}<ChevronDown size={14} class="text-text-muted shrink-0" />{/if}
                    <Icon size={14} class="text-primary-light shrink-0" /><span class="text-sm font-medium text-text-primary truncate">{group.label}</span><span class="text-xs text-text-muted shrink-0">({group.permissions.length})</span>
                  </button>
                  <button type="button" class="text-[10px] font-semibold shrink-0 px-2 py-1 rounded-md transition-colors {allSel ? 'bg-primary/15 text-primary border border-primary/20' : partialSel ? 'bg-warning/15 text-warning-light border border-warning/20' : 'bg-surface-default text-text-muted border border-border/50 hover:border-border-strong hover:text-text-primary'}" onclick={() => toggleGroupAll(group)} aria-label={allSel ? 'Deselect all ' + group.label + ' permissions' : 'Select all ' + group.label + ' permissions'} aria-pressed={allSel}>
                    {#if partialSel}<ChevronsUpDown size={10} />{:else if allSel}<Check size={10} />{:else}{labels.all}{/if}
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
            <div class="py-6 text-center"><p class="text-sm text-text-muted italic">{labels.noResults}: "{permissionSearch}"</p></div>
          {/if}
        </div>
        {#if form.permission_ids.length > 0}
          <div class="flex items-center justify-between px-3 py-2 mt-2 rounded-lg bg-surface-subtle/40 border border-border/30">
            <span class="text-xs text-text-secondary"><span class="font-semibold text-text-primary">{form.permission_ids.length}</span> {labels.permissions} {#if selectedGroupCount > 0}<span class="text-text-muted"> across {selectedGroupCount} {selectedGroupCount === 1 ? 'group' : 'groups'}</span>{/if}</span>
            <button type="button" class="text-xs text-danger hover:text-danger-light transition-colors" onclick={() => form.permission_ids = []}>{labels.clearAll}</button>
          </div>
        {/if}
      </div>
    {/if}
  </div>
  {#snippet footer()}
    <div class="flex items-center justify-between w-full">
      <div>{#if modalStep === 2}<Button variant="ghost" size="sm" onclick={() => modalStep = 1} disabled={saving}>← {labels.back}</Button>{/if}</div>
      <div class="flex items-center gap-2">
        <Button variant="secondary" onclick={requestClose} disabled={saving}>{labels.cancel}</Button>
        {#if modalStep === 1}
          <Button variant="primary" class="min-w-28" onclick={proceedToPermissions} disabled={!form.name.trim()}>{labels.next} →</Button>
        {:else}
          <Button variant="primary" class="min-w-32" onclick={saveRole} disabled={saving || !!nameErrorText} aria-busy={saving}>
            {#if saving}<Loader2 size={16} class="animate-spin" /> {labels.saving}{:else}{modalMode === 'add' ? `${labels.create} ${labels.role}` : labels.save}{/if}
          </Button>
        {/if}
      </div>
    </div>
    {#if modalStep === 2}<div class="text-[10px] text-text-muted text-center mt-1 w-full">{labels.ctrlEnterToSave}</div>{/if}
  {/snippet}
</Modal>

<!-- Discard Confirmation -->
<Modal bind:open={showDiscardModal} title={labels.discardChanges} size="sm">
  <div class="text-center py-2">
    <div class="w-14 h-14 rounded-2xl bg-warning-subtle flex items-center justify-center mx-auto mb-4"><Eye size={24} class="text-warning" /></div>
    <p class="text-text-primary font-semibold mb-1">{labels.youHaveUnsavedChanges}</p>
    <p class="text-text-muted text-sm">Your permission selections will be lost if you close without saving.</p>
  </div>
  {#snippet footer()}<Button variant="secondary" onclick={cancelDiscard}>{labels.keepEditing}</Button><Button variant="danger" onclick={confirmDiscard}>{labels.discard}</Button>{/snippet}
</Modal>

<ConfirmDeleteModal bind:open={showDeleteModal} title={labels.deleteRole} itemName={selectedRole?.name} description={labels.thisActionCannotBeUndone} loading={false} onconfirm={confirmDelete} oncancel={() => showDeleteModal = false} />

<!-- Role Detail Drawer -->
<RoleDetailDrawer
  bind:open={showRoleDrawer}
  {selectedRole}
  {permissions}
  {canEdit}
  {canDelete}
  onclose={closeRoleDrawer}
  onedit={() => { showRoleDrawer = false; closeAll(); openEdit(selectedRole); }}
  onduplicate={() => { showRoleDrawer = false; closeAll(); openDuplicate(selectedRole); }}
  ondeleterequest={() => { showRoleDrawer = false; showDeleteModal = true; }}
/>
