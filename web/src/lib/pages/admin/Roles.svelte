<script>
  import { onMount } from 'svelte';
  import { apiFetch } from '$lib/api/client';
  import { toast } from '$lib/stores/toast';

  import Badge from '$lib/components/ui/Badge.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import { Plus, Pencil, Trash2, Shield, Loader2 } from 'lucide-svelte';

  let loading = $state(true);
  let roles = $state([]);
  let permissions = $state([]);
  let showModal = $state(false);
  let showDeleteModal = $state(false);
  let selectedRole = $state(null);
  let modalMode = $state('add');
  let saving = $state(false);

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
    form = { name: '', description: '', permission_ids: [] };
    showModal = true;
  }

  function openEdit(role) {
    modalMode = 'edit';
    selectedRole = role;
    
    // Map permission codes to IDs for the form
    const currentPermIds = permissions
      .filter(p => role.permissions.includes(p.code))
      .map(p => p.id);

    form = { 
      name: role.name, 
      description: role.description, 
      permission_ids: currentPermIds 
    };
    showModal = true;
  }

  async function saveRole() {
    if (!form.name) {
      toast.error('Role name is required');
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
          // After creating role, update its permissions
          await apiFetch(`/api/admin/roles/${newRole.data.id}/permissions`, {
            method: 'PUT',
            body: JSON.stringify({ permission_ids: form.permission_ids })
          });
          toast.success('Role created');
        } else {
          throw new Error('Failed to create role');
        }
      } else {
        // Update permissions for existing role
        const r = await apiFetch(`/api/admin/roles/${selectedRole.id}/permissions`, {
          method: 'PUT',
          body: JSON.stringify({ permission_ids: form.permission_ids })
        });
        
        if (r.ok) {
          toast.success('Permissions updated');
        } else {
          throw new Error('Failed to update permissions');
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
        toast.error('Failed to delete role');
      }
    } catch {
      toast.error('Failed to delete role');
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

  onMount(fetchData);
</script>

<div class="space-y-5">
  <!-- Header Section -->
  <div class="flex items-center justify-between mb-8">
    <div>
      <h2 class="text-3xl font-extrabold text-text-primary tracking-tight">Roles Management</h2>
      <p class="text-text-muted mt-1">Define and orchestrate system-wide access control and permission sets</p>
    </div>
    <div class="flex items-center gap-3">
      <button class="btn btn-primary px-6 py-2.5 rounded-xl shadow-glow-primary-sm hover:shadow-glow-primary transition-all active:scale-95" onclick={openAdd}>
        <Plus size={18} class="mr-1.5" /> Create Role
      </button>
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
      <div class="empty-state-icon bg-surface w-20 h-20 mx-auto">
        <Shield size={32} class="text-text-muted" />
      </div>
      <p class="text-text-primary font-semibold mt-4">No roles defined</p>
      <p class="text-text-muted text-sm mt-1">Create your first role to get started</p>
      <button class="btn btn-primary mt-4" onclick={openAdd}>
        <Plus size={14} /> Add First Role
      </button>
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
              <button 
                class="btn-icon w-9 h-9 rounded-lg text-text-muted hover:text-primary-light hover:bg-primary-subtle/50 transition-all active:scale-90" 
                title="Edit Permissions"
                onclick={() => openEdit(role)}
              >
                <Pencil size={15} />
              </button>
              {#if !role.is_system}
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
</div>

<!-- Add/Edit Role Modal -->
<Modal bind:open={showModal} title={modalMode === 'add' ? 'Create New Role' : 'Edit Permissions'} size="md">
  <div class="space-y-4">
    {#if modalMode === 'add'}
      <div>
        <label for="role-name" class="block text-sm font-medium text-text-secondary mb-2">Role Name</label>
        <input id="role-name" type="text" placeholder="e.g. manager" class="input" bind:value={form.name} required />
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
    
<div>
  <p class="block text-sm font-medium text-text-secondary mb-3">Permissions</p>
  <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
        {#each permissions as perm}
          <label 
            class="flex items-center gap-3 px-3 py-2.5 rounded-xl border cursor-pointer transition-colors
                   {form.permission_ids.includes(perm.id) ? 'border-primary/40 bg-primary-subtle/40' : 'border-border hover:border-border-strong hover:bg-surface-default'}"
          >
            <input 
              type="checkbox" 
              class="w-4 h-4 accent-primary rounded" 
              checked={form.permission_ids.includes(perm.id)}
              onchange={() => togglePermission(perm.id)}
            />
            <div>
              <p class="text-sm font-medium text-text-primary">{perm.name}</p>
              <p class="text-[10px] text-text-muted uppercase tracking-wider">{perm.code}</p>
            </div>
          </label>
        {/each}
      </div>
    </div>
  </div>
  {#snippet footer()}
    <button class="btn btn-secondary" onclick={() => showModal = false} disabled={saving}>Cancel</button>
    <button class="btn btn-primary min-w-32" onclick={saveRole} disabled={saving}>
      {#if saving}
        <Loader2 size={16} class="animate-spin" /> Saving...
      {:else}
        {modalMode === 'add' ? 'Create Role' : 'Update Permissions'}
      {/if}
    </button>
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