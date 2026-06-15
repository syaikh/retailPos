<script>
  import { onMount } from 'svelte';
  import { apiFetch } from '$lib/api/client';
  import { toast } from '$lib/stores/toast';
  import { auth } from '$lib/stores/auth';

  import Modal from '$lib/components/ui/Modal.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import { Plus, Pencil, Trash2, Shield, Loader2, Search, X, ChevronRight, ChevronDown, Users, Package, Tag, ShoppingCart, Warehouse, UserPlus, BarChart3, LayoutDashboard, Settings, Store, Eye } from 'lucide-svelte';

  let loading = $state(true);
  let roles = $state([]);
  let permissions = $state([]);
  let showModal = $state(false);
  let showDeleteModal = $state(false);
  let showDiscardModal = $state(false);
  let selectedRole = $state(null);
  let modalMode = $state('add');
  let saving = $state(false);
  let permissionSearch = $state('');
  let nameTouched = $state(false);
  let pendingClose = $state(false);

  let userRole = $derived(
    $auth.user?.role?.name ||
    ($auth.user?.role && typeof $auth.user?.role === 'object' ? $auth.user.role.name : $auth.user?.role) ||
    ''
  );
  let userPermissions = $derived($auth.user?.permissions || []);
  let canView = $derived(userRole !== 'cashier' && userRole !== '');
  let canEdit = $derived(userRole === 'superadmin');
  let canDelete = $derived(userRole === 'superadmin');

  let collapsedGroups = $state(new Set());
  let initialPermissionIds = $state([]);

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
      // Merge role:* permissions into user group
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

  let searchResultCount = $derived(
    permissionSearch.trim()
      ? permissions.filter(p =>
          p.name.toLowerCase().includes(permissionSearch.toLowerCase()) ||
          p.code.toLowerCase().includes(permissionSearch.toLowerCase())
        ).length
      : permissions.length
  );

  let nameError = $derived(() => {
    if (!nameTouched) return '';
    if (!form.name.trim()) return 'Role name is required';
    if (roles.some(r => r.name.toLowerCase() === form.name.trim().toLowerCase() && r.id !== selectedRole?.id)) {
      return 'Role name already exists';
    }
    return '';
  });

  let nameErrorText = $derived(nameError());

  let hasUnsavedChanges = $derived(() => {
    const current = JSON.stringify([...form.permission_ids].sort());
    const initial = JSON.stringify([...initialPermissionIds].sort());
    return current !== initial;
  });

  function allExpanded() {
    return groupedPermissions().every(g => !collapsedGroups.has(g.key));
  }

  function toggleGroup(key) {
    const s = new Set(collapsedGroups);
    if (s.has(key)) s.delete(key);
    else s.add(key);
    collapsedGroups = s;
  }

  function setAllExpanded(expanded) {
    collapsedGroups = expanded ? new Set() : new Set(groupedPermissions().map(g => g.key));
  }

  function toggleGroupAll(group) {
    const allSelected = group.permissions.every(p => form.permission_ids.includes(p.id));
    if (allSelected) {
      form.permission_ids = form.permission_ids.filter(id => !group.permissions.some(p => p.id === id));
    } else {
      const toAdd = group.permissions.filter(p => !form.permission_ids.includes(p.id)).map(p => p.id);
      form.permission_ids = [...form.permission_ids, ...toAdd];
    }
  }

  function requestClose() {
    if (hasUnsavedChanges()) {
      pendingClose = true;
      showDiscardModal = true;
    } else {
      showModal = false;
    }
  }

  function confirmDiscard() {
    showDiscardModal = false;
    showModal = false;
    pendingClose = false;
  }

  function cancelDiscard() {
    showDiscardModal = false;
    pendingClose = false;
  }

  // Form State
  let form = $state({
    name: '',
    description: '',
    permission_ids: []
  });

  async function fetchData() {
    try {
      loading = true;
      const [rRes, pRes] = await Promise.all([
        apiFetch('/api/admin/roles'),
        apiFetch('/api/admin/permissions')
      ]);

      if (rRes.ok) {
        const data = await rRes.json();
        roles = data.data || [];
      }
      if (pRes.ok) {
        const data = await pRes.json();
        permissions = data.data || [];
      }
    } catch {
      toast.error('Failed to load roles and permissions');
    } finally {
      loading = false;
    }
  }

  function openAdd() {
    modalMode = 'add';
    permissionSearch = '';
    collapsedGroups = new Set(permissions.map(p => p.code.split(':')[0] === 'role' ? 'user' : p.code.split(':')[0]));
    nameTouched = false;
    initialPermissionIds = [];
    form = { name: '', description: '', permission_ids: [] };
    showModal = true;
  }

  function openEdit(role) {
    modalMode = 'edit';
    permissionSearch = '';
    collapsedGroups = new Set(permissions.map(p => p.code.split(':')[0] === 'role' ? 'user' : p.code.split(':')[0]));
    nameTouched = false;
    selectedRole = role;

    const currentPermIds = permissions
      .filter(p => role.permissions.includes(p.code))
      .map(p => p.id);

    initialPermissionIds = [...currentPermIds];
    form = {
      name: role.name,
      description: role.description,
      permission_ids: currentPermIds
    };
    showModal = true;
  }

  async function saveRole() {
    nameTouched = true;
    if (nameErrorText) {
      toast.error(nameErrorText);
      return;
    }

    try {
      saving = true;

      if (modalMode === 'add') {
        const r = await apiFetch('/api/admin/roles', {
          method: 'POST',
          body: JSON.stringify({ name: form.name, description: form.description })
        });

        if (r.ok) {
          const newRole = await r.json();
          await apiFetch(`/api/admin/roles/${newRole.data.id}/permissions`, {
            method: 'PUT',
            body: JSON.stringify({ permission_ids: form.permission_ids })
          });
          toast.success('Role created');
        } else {
          const data = await r.json().catch(() => ({}));
          throw new Error(data.error || 'Failed to create role');
        }
      } else {
        const r = await apiFetch(`/api/admin/roles/${selectedRole.id}/permissions`, {
          method: 'PUT',
          body: JSON.stringify({ permission_ids: form.permission_ids })
        });

        if (r.ok) {
          toast.success('Permissions updated');
        } else {
          const data = await r.json().catch(() => ({}));
          throw new Error(data.error || 'Failed to update permissions');
        }
      }

      showModal = false;
      await fetchData();
    } catch (err) {
      toast.error(err.message || 'Operation failed');
    } finally {
      saving = false;
    }
  }

  async function confirmDelete() {
    if (!selectedRole) return;
    if (selectedRole.is_system) {
      toast.error('Cannot delete system roles');
      return;
    }

    try {
      const r = await apiFetch(`/api/admin/roles/${selectedRole.id}`, { method: 'DELETE' });
      if (r.ok) {
        roles = roles.filter(r => r.id !== selectedRole.id);
        toast.success(`Role "${selectedRole.name}" removed`);
      } else {
        const data = await r.json().catch(() => ({}));
        toast.error(data.error || 'Failed to delete role');
      }
    } catch {
      toast.error('Connection lost. Check your network.');
    } finally {
      showDeleteModal = false;
      selectedRole = null;
    }
  }

  function togglePermission(id) {
    if (form.permission_ids.includes(id)) {
      form.permission_ids = form.permission_ids.filter(pid => pid !== id);
    } else {
      form.permission_ids = [...form.permission_ids, id];
    }
  }

  function handleGroupKeydown(e, group) {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      const next = e.target.closest('[data-group]')?.nextElementSibling?.querySelector('button[data-group-toggle]');
      next?.focus();
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      const prev = e.target.closest('[data-group]')?.previousElementSibling?.querySelector('button[data-group-toggle]');
      prev?.focus();
    }
  }

  onMount(fetchData);
</script>

<div class="space-y-5">
  {#if !canView}
    <div class="card px-4 py-12 text-center">
      <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
        <Shield size={32} class="text-text-muted" />
      </div>
      <p class="text-text-primary font-semibold mt-4">Access Denied</p>
      <p class="text-text-muted text-sm mt-1">You do not have permission to view roles</p>
    </div>
  {:else}
    <!-- Header Section -->
    <div class="flex items-center justify-between mb-8">
      <div>
        <h2 class="text-3xl font-extrabold text-text-primary tracking-tight">Roles Management</h2>
        <p class="text-text-muted mt-1">Define and orchestrate system-wide access control and permission sets</p>
      </div>
      <div class="flex items-center gap-3">
        {#if canEdit}
        <button class="btn btn-primary px-6 py-2.5 rounded-xl shadow-glow-primary-sm hover:shadow-glow-primary transition-all active:scale-95" onclick={openAdd}>
          <Plus size={18} class="mr-1.5" /> Create Role
        </button>
        {/if}
      </div>
    </div>

    {#if loading}
      <div class="grid gap-4">
        {#each { length: 3 } as _}
          <div class="card p-5 space-y-3">
            <Skeleton width="w-32" height="h-5" />
            <Skeleton width="w-full" height="h-3" />
            <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 pt-2">
              {#each { length: 6 } as _}
                <Skeleton height="h-8" rounded="rounded-lg" />
              {/each}
            </div>
          </div>
        {/each}
      </div>
    {:else if roles.length === 0}
      <div class="card px-4 py-12 text-center">
        <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
          <Shield size={32} class="text-text-muted" />
        </div>
        <p class="text-text-primary font-semibold mt-4">No roles defined</p>
        <p class="text-text-muted text-sm mt-1">Create your first role to get started</p>
        {#if canEdit}
        <button class="btn btn-primary mt-4" onclick={openAdd}>
          <Plus size={14} /> Add First Role
        </button>
        {/if}
      </div>
    {:else}
      <div class="grid gap-6">
        {#each roles as role (role.id)}
          <div class="card p-6 bg-surface/40 backdrop-blur-xl border-border/40 hover:border-primary/30 hover:shadow-glow-primary-sm transition-all duration-300 group relative overflow-hidden">
            <!-- Subtle decorative background element -->
            <div class="absolute -right-4 -top-4 w-24 h-24 bg-primary/5 rounded-full blur-3xl group-hover:bg-primary/10 transition-colors"></div>
            
            <div class="flex flex-col md:flex-row md:items-start justify-between gap-4 mb-5">
              <div class="flex items-center gap-4">
                <div class="w-12 h-12 rounded-2xl bg-linear-to-br from-primary/20 to-primary/5 flex items-center justify-center border border-primary/20 shadow-inner">
                  <Shield size={22} class="text-primary-light" />
                </div>
                <div>
                  <div class="flex items-center gap-3">
                    <h3 class="text-lg font-bold text-text-primary capitalize tracking-tight">{role.name}</h3>
                    {#if role.is_system}
                      <span class="px-2 py-0.5 rounded-md bg-info-subtle/30 text-info-light text-[10px] font-bold uppercase tracking-widest border border-info/20 shadow-sm">System</span>
                    {/if}
                    <span class="px-2 py-0.5 rounded-md bg-surface-hover/50 text-text-muted text-[10px] font-bold uppercase tracking-widest border border-border/50">
                      {Array.isArray(role.permissions) ? role.permissions.length : 0} Perms
                    </span>
                  </div>
                  {#if role.description}
                    <p class="text-sm text-text-muted mt-1 leading-relaxed max-w-xl">{role.description}</p>
                  {/if}
                </div>
              </div>
              
              <div class="flex items-center gap-2 self-end md:self-start bg-surface-subtle/30 p-1.5 rounded-xl border border-border/30 backdrop-blur-sm">
                {#if canEdit}
                <button 
                  class="btn-icon w-9 h-9 rounded-lg text-text-muted hover:text-primary-light hover:bg-primary-subtle/50 transition-all active:scale-90" 
                  title="Edit Permissions"
                  onclick={() => openEdit(role)}
                >
                  <Pencil size={15} />
                </button>
                {/if}
                {#if canDelete && !role.is_system}
                  <button
                    class="btn-icon w-9 h-9 rounded-lg text-text-muted hover:text-danger-light hover:bg-danger-subtle/50 transition-all active:scale-90"
                    onclick={() => { selectedRole = role; showDeleteModal = true; }}
                    title="Delete Role"
                  >
                    <Trash2 size={15} />
                  </button>
                {/if}
              </div>
            </div>

            <!-- Permission grid -->
            <div class="bg-surface-subtle/20 rounded-2xl p-4 border border-border/20">
              <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
                {#if Array.isArray(role.permissions) && role.permissions.length > 0}
                  {#each role.permissions as permCode}
                    {@const permObj = permissions.find(p => p.code === permCode)}
                    <div class="flex items-center gap-2.5 px-3.5 py-2.5 rounded-xl border border-primary/10 bg-surface/40 text-xs font-semibold text-text-secondary hover:border-primary/30 hover:bg-surface-hover/30 transition-all cursor-default shadow-sm group/tag">
                      <div class="w-2 h-2 rounded-full shrink-0 bg-primary/40 group-hover/tag:bg-primary-light transition-colors shadow-[0_0_8px_rgba(var(--primary-rgb),0.3)]"></div>
                      <span class="truncate">{permObj?.name || permCode}</span>
                    </div>
                  {/each}
                {:else}
                  <div class="col-span-full py-4 text-center">
                    <p class="text-sm text-text-muted italic opacity-60 flex items-center justify-center gap-2">
                      <Loader2 size={14} class="opacity-50" /> No permissions assigned to this role
                    </p>
                  </div>
                {/if}
              </div>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  {/if}
</div>

<!-- Add/Edit Role Modal -->
<Modal bind:open={showModal} title={modalMode === 'add' ? 'Create New Role' : 'Edit Permissions'} size="lg">
  <div class="space-y-4">
    {#if modalMode === 'add'}
      <div>
        <label for="role-name" class="block text-sm font-medium text-text-secondary mb-2">Role Name</label>
        <input
          id="role-name"
          type="text"
          placeholder="e.g. manager"
          class="input"
          class:border-danger={nameErrorText}
          bind:value={form.name}
          onblur={() => nameTouched = true}
          aria-invalid={!!nameErrorText}
          aria-describedby={nameErrorText ? 'role-name-error' : undefined}
          required
        />
        {#if nameErrorText}
          <p id="role-name-error" class="text-xs text-danger mt-1.5" role="alert">{nameErrorText}</p>
        {/if}
      </div>
      <div>
        <label for="role-desc" class="block text-sm font-medium text-text-secondary mb-2">Description</label>
        <input id="role-desc" type="text" placeholder="Short description of this role" class="input" bind:value={form.description} />
      </div>
    {:else}
      <div class="flex items-center gap-3 p-3 bg-bg-secondary rounded-xl border border-border-default">
        <Shield size={20} class="text-primary-light" />
        <div>
          <p class="text-sm font-semibold text-text-primary capitalize">{selectedRole?.name}</p>
          <p class="text-xs text-text-muted">{selectedRole?.description || 'No description'}</p>
        </div>
      </div>
    {/if}

    <div class="border-t border-border pt-5 mt-2">
      <div class="sticky top-0 bg-surface pb-3 z-10">
        <div class="flex items-center justify-between mb-3">
          <p class="text-sm font-medium text-text-secondary">Permissions</p>
          <div class="flex items-center gap-3">
            <span class="text-xs text-text-muted">{form.permission_ids.length} of {permissions.length} selected</span>
            {#if groupedPermissions().length > 1}
              <button
                type="button"
                class="text-xs text-primary hover:text-primary-light transition-colors"
                onclick={() => setAllExpanded(!allExpanded())}
              >
                {allExpanded() ? 'Collapse All' : 'Expand All'}
              </button>
            {/if}
          </div>
        </div>
        <div class="relative">
          <Search size={16} class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none" />
          <input
            type="text"
            placeholder="Cari permission..."
            class="input pl-9 pr-9 text-sm"
            bind:value={permissionSearch}
            role="searchbox"
            aria-label="Search permissions"
            aria-describedby="search-status"
            onkeydown={(e) => {
              if (e.key === 'Escape' && permissionSearch) {
                e.stopPropagation();
                permissionSearch = '';
              }
            }}
          />
          {#if permissionSearch}
            <button
              type="button"
              class="absolute right-3 top-1/2 -translate-y-1/2 p-0.5 text-text-muted hover:text-text-primary transition-colors"
              onclick={() => permissionSearch = ''}
              aria-label="Clear search"
            >
              <X size={14} />
            </button>
          {/if}
        </div>
        <div id="search-status" class="sr-only" aria-live="polite">
          {searchResultCount} permissions found
        </div>
      </div>
      <div class="space-y-3 max-h-64 overflow-y-auto">
        {#if groupedPermissions().length > 0}
          {#each groupedPermissions() as group (group.key)}
            {@const isCollapsed = collapsedGroups.has(group.key)}
            {@const Icon = group.icon}
            {@const allGroupSelected = group.permissions.every(p => form.permission_ids.includes(p.id))}
            {@const someGroupSelected = group.permissions.some(p => form.permission_ids.includes(p.id)) && !allGroupSelected}
            <div class="rounded-xl border border-border/50 overflow-hidden" data-group>
              <div class="flex items-center gap-3 px-3 py-3 bg-surface-subtle/40 hover:bg-surface-subtle/60 transition-colors">
                <button
                  type="button"
                  class="flex items-center gap-3 flex-1 text-left min-w-0"
                  aria-expanded={!isCollapsed}
                  aria-controls="group-body-{group.key}"
                  aria-label="Toggle {group.label} permissions"
                  onclick={() => toggleGroup(group.key)}
                  onkeydown={(e) => handleGroupKeydown(e, group)}
                  data-group-toggle
                >
                  {#if isCollapsed}
                    <ChevronRight size={14} class="text-text-muted shrink-0" />
                  {:else}
                    <ChevronDown size={14} class="text-text-muted shrink-0" />
                  {/if}
                  <Icon size={14} class="text-primary-light shrink-0" />
                  <span class="text-sm font-medium text-text-primary truncate">{group.label}</span>
                </button>
                {#if isCollapsed}
                  <span class="text-[10px] text-text-muted font-medium shrink-0">{group.permissions.length} permissions</span>
                {:else}
                  <button
                    type="button"
                    class="text-[10px] font-semibold shrink-0 px-2 py-1 rounded-md transition-colors {allGroupSelected ? 'bg-primary/10 text-primary border border-primary/20' : 'bg-surface-default text-text-muted border border-border/50 hover:border-border-strong hover:text-text-primary'}"
                    onclick={() => toggleGroupAll(group)}
                    aria-label={allGroupSelected ? 'Deselect all ' + group.label + ' permissions' : 'Select all ' + group.label + ' permissions'}
                  >
                    {allGroupSelected ? 'Deselect All' : 'Select All'}
                  </button>
                {/if}
                <span class="inline-flex items-center text-[10px] font-semibold px-1.5 py-0.5 rounded-md bg-primary/10 text-primary shrink-0">
                  {group.permissions.filter(p => form.permission_ids.includes(p.id)).length}/{group.permissions.length}
                </span>
              </div>
              {#if !isCollapsed}
                <div id="group-body-{group.key}" class="grid grid-cols-1 sm:grid-cols-2 gap-1.5 p-3 bg-surface/20">
                  {#each group.permissions as perm (perm.id)}
                    <label
                      class="flex items-center gap-3 px-3 py-2.5 rounded-lg border cursor-pointer transition-colors
                             {form.permission_ids.includes(perm.id) ? 'border-primary/40 bg-primary-subtle/40' : 'border-transparent hover:border-border hover:bg-surface-default'}"
                      title={perm.description || ''}
                    >
                      <input
                        type="checkbox"
                        id="perm-{perm.id}"
                        class="w-4 h-4 accent-primary rounded"
                        checked={form.permission_ids.includes(perm.id)}
                        onchange={() => togglePermission(perm.id)}
                      />
                      <div class="min-w-0">
                        <p class="text-sm font-medium text-text-primary truncate">{perm.name}</p>
                        <p class="text-[10px] text-text-muted uppercase tracking-wider">{perm.code}</p>
                      </div>
                    </label>
                  {/each}
                </div>
              {/if}
            </div>
          {/each}
        {:else}
          <div class="py-6 text-center">
            <p class="text-sm text-text-muted italic">Tidak ada permission yang cocok</p>
          </div>
        {/if}
      </div>
    </div>
  </div>
  {#snippet footer()}
    <button class="btn btn-secondary" onclick={requestClose} disabled={saving}>Cancel</button>
    <button class="btn btn-primary min-w-32" onclick={saveRole} disabled={saving || !!nameErrorText} aria-busy={saving}>
      {#if saving}
        <Loader2 size={16} class="animate-spin" /> Saving...
      {:else}
        {modalMode === 'add' ? 'Create Role' : 'Update Permissions'}
      {/if}
    </button>
  {/snippet}
</Modal>

<!-- Unsaved Changes Confirmation -->
<Modal bind:open={showDiscardModal} title="Discard Changes?" size="sm">
  <div class="text-center py-2">
    <div class="w-14 h-14 rounded-2xl bg-warning-subtle flex items-center justify-center mx-auto mb-4">
      <Eye size={24} class="text-warning" />
    </div>
    <p class="text-text-primary font-semibold mb-1">You have unsaved changes</p>
    <p class="text-text-muted text-sm">Your permission selections will be lost if you close without saving.</p>
  </div>
  {#snippet footer()}
    <button class="btn btn-secondary" onclick={cancelDiscard}>Keep Editing</button>
    <button class="btn btn-danger" onclick={confirmDiscard}>Discard</button>
  {/snippet}
</Modal>

<!-- Delete Confirm -->
<Modal bind:open={showDeleteModal} title="Delete Role" size="sm">
  <div class="text-center py-2">
    <div class="w-14 h-14 rounded-2xl bg-danger-subtle flex items-center justify-center mx-auto mb-4">
      <Trash2 size={24} class="text-danger" />
    </div>
    <p class="text-text-primary font-semibold mb-1">Delete role "{selectedRole?.name}"?</p>
    <p class="text-text-muted text-sm">Users with this role will lose their assigned permissions. This action cannot be undone.</p>
  </div>
  {#snippet footer()}
    <button class="btn btn-secondary" onclick={() => showDeleteModal = false}>Cancel</button>
    <button class="btn btn-danger" onclick={confirmDelete}>Delete</button>
  {/snippet}
</Modal>