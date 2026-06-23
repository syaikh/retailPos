<script>
  import { fly } from 'svelte/transition';
  import { Badge, Button } from '$shared/ui';
  import { X, Shield, Users, Copy, Pencil, Trash2 } from 'lucide-svelte';

  let {
    selectedRole = null,
    permissions = [],
    canEdit = false,
    canDelete = false,
    onclose = () => {},
    onedit = () => {},
    onduplicate = () => {},
    ondeleterequest = () => {},
  } = $props();

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

  let rolePerms = $derived(getRolePermissions(selectedRole));
  let grouped = $derived(getGroupedPermissions(rolePerms));

  const groupMeta = {
    'user': { label: 'User & Role', icon: Users },
    'product': { label: 'Product', icon: Shield },
    'category': { label: 'Category', icon: Shield },
    'sale': { label: 'Sales', icon: Shield },
    'inventory': { label: 'Inventory', icon: Shield },
    'customer': { label: 'Customer', icon: Shield },
    'report': { label: 'Report', icon: Shield },
    'dashboard': { label: 'Dashboard', icon: Shield },
    'pos': { label: 'POS', icon: Shield },
    'audit': { label: 'System', icon: Shield },
  };

  function handleDrawerKeydown(e) {
    if (e.key === 'Escape') {
      e.preventDefault();
      onclose();
    }
  }
</script>

<div class="fixed inset-0 bg-black/60 z-50" onclick={onclose} aria-hidden="true"></div>
<div
  class="fixed inset-y-0 right-0 w-[520px] max-w-full bg-surface-default border-l border-border shadow-2xl z-[55] flex flex-col transition-transform duration-300 ease-out"
  transition:fly={{ x: 520, duration: 300, easing: t => t * (2 - t) }}
  role="dialog" aria-modal="true" aria-labelledby="role-detail-heading" tabindex="-1"
  onkeydown={handleDrawerKeydown}
>
  <div class="flex items-center justify-between px-6 py-5 border-b border-border shrink-0">
    <div class="flex items-center gap-3">
      <div class="w-9 h-9 rounded-lg bg-primary-subtle flex items-center justify-center shrink-0"><Shield size={16} class="text-primary-light" /></div>
      <h2 id="role-detail-heading" class="text-lg font-bold text-text-primary">{selectedRole.name}</h2>
      {#if selectedRole.is_system}<Badge variant="primary" size="sm">System</Badge>{:else}<Badge variant="muted" size="sm">Custom</Badge>{/if}
    </div>
    <button
      class="p-2 rounded-lg text-text-muted hover:bg-surface-hover hover:text-text-secondary transition-colors"
      onclick={onclose}
      onkeydown={(e) => { if (e.key === 'Enter' || e.key === 'Escape' || e.key === ' ') { e.preventDefault(); onclose(); } }}
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
          <Button
            variant="secondary"
            class="flex-1 rounded-xl px-4 h-11 text-sm font-semibold text-text-secondary border border-border hover:border-primary hover:text-primary hover:bg-primary-subtle transition-all duration-200"
            onclick={onduplicate}
          >
            <Copy size={15} class="mr-1.5" />Duplicate
          </Button>
          <Button
            variant="primary"
            class="flex-1 rounded-xl px-4 h-11 text-sm font-semibold text-white shadow-glow-primary-sm transition-all duration-200"
            onclick={onedit}
          >
            <Pencil size={15} class="mr-1.5" />Edit
          </Button>
        {/if}
        {#if canDelete && !selectedRole.is_system}
          <Button
            variant="danger"
            class="rounded-xl px-4 h-11 text-sm font-semibold transition-all duration-200"
            onclick={ondeleterequest}
          >
            <Trash2 size={15} class="mr-1.5" />Delete
          </Button>
        {/if}
      </div>
    </div>
  {/if}
</div>
