<script>
  import { Badge, Button, Drawer } from '$shared/ui';
  import { Shield, Users, Copy, Pencil, Trash2 } from 'lucide-svelte';
  import { labels } from '$shared/i18n';

  let {
    open = $bindable(false),
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
    'user': { label: `${labels.user} & ${labels.role}`, icon: Users },
    'product': { label: labels.product, icon: Shield },
    'category': { label: labels.category, icon: Shield },
    'sale': { label: 'Sales', icon: Shield },
    'inventory': { label: labels.inventory, icon: Shield },
    'customer': { label: labels.customer, icon: Shield },
    'report': { label: labels.reports, icon: Shield },
    'dashboard': { label: labels.dashboard, icon: Shield },
    'pos': { label: 'POS', icon: Shield },
    'audit': { label: labels.system, icon: Shield },
  };
</script>

<Drawer bind:open width={520} ariaLabel={`${labels.role} ${labels.details}`} onclose={() => onclose()}>
  <div class="flex items-center gap-3 mb-4">
    <div class="w-9 h-9 rounded-lg bg-primary-subtle flex items-center justify-center shrink-0"><Shield size={16} class="text-primary-light" /></div>
    <h2 class="text-lg font-bold text-text-primary">{selectedRole.name}</h2>
    {#if selectedRole.is_system}<Badge variant="primary" size="sm">{labels.system}</Badge>{:else}<Badge variant="muted" size="sm">{labels.custom}</Badge>{/if}
  </div>

  <div class="space-y-4">
    {#if selectedRole.description}
      <div class="rounded-2xl bg-surface-default border border-border p-4">
        <p class="text-xs font-semibold text-text-muted uppercase tracking-wider mb-1.5">{labels.description}</p>
        <p class="text-sm text-text-secondary leading-relaxed">{selectedRole.description}</p>
      </div>
    {/if}

    <div class="rounded-2xl bg-surface-default border border-border overflow-hidden">
      <div class="px-4 py-2.5 border-b border-border/60 flex items-center gap-2">
        <Users size={14} class="text-text-muted" />
        <h3 class="text-xs font-semibold uppercase tracking-wide text-text-muted/80">{labels.permissions} ({rolePerms.length})</h3>
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
          <p class="text-sm text-text-muted italic">{labels.noPermissionsAssigned}</p>
        </div>
      {/if}
    </div>
  </div>

  {#snippet footer()}
    {#if canEdit || canDelete}
      <div class="flex items-center gap-3">
        {#if canEdit}
          <Button
            variant="secondary"
            class="flex-1 rounded-xl px-4 h-11 text-sm font-semibold text-text-secondary border border-border hover:border-primary hover:text-primary hover:bg-primary-subtle transition-all duration-200"
            onclick={onduplicate}
          >
            <Copy size={15} class="mr-1.5" />{labels.duplicate}
          </Button>
          <Button
            variant="primary"
            class="flex-1 rounded-xl px-4 h-11 text-sm font-semibold text-white shadow-glow-primary-sm transition-all duration-200"
            onclick={onedit}
          >
            <Pencil size={15} class="mr-1.5" />{labels.edit}
          </Button>
        {/if}
        {#if canDelete && !selectedRole.is_system}
          <Button
            variant="danger"
            class="rounded-xl px-4 h-11 text-sm font-semibold transition-all duration-200"
            onclick={ondeleterequest}
          >
            <Trash2 size={15} class="mr-1.5" />{labels.delete}
          </Button>
        {/if}
      </div>
    {/if}
  {/snippet}
</Drawer>
