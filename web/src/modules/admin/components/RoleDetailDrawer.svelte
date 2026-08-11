<script>
  import { Badge, Button, Drawer, SearchBar } from '$shared/ui';
  import { ChevronRight, Copy, Pencil, Search, Shield, Trash2, Users } from 'lucide-svelte';
  import { labels } from '$shared/i18n';
  import { groupPermissions } from '$shared/utils/permissionGroups';

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
    if (!role || !role.permissions || !role.permissions.length) return [];
    const permCodes = new Set(role.permissions);
    return permissions.filter(p => permCodes.has(p.code));
  }

  let rolePerms = $derived(getRolePermissions(selectedRole));
  let permSearch = $state('');
  let collapsedKeys = $state(new Set());

  $effect(() => {
    if (open) {
      permSearch = '';
      collapsedKeys = new Set(groupPermissions(rolePerms).map(g => g.key));
    }
  });

  let isSearching = $derived(permSearch.trim().length > 0);

  let filteredGrouped = $derived.by(() => {
    const search = permSearch.trim().toLowerCase();
    const filtered = search
      ? rolePerms.filter(p => p.name.toLowerCase().includes(search) || p.code.toLowerCase().includes(search))
      : rolePerms;
    return groupPermissions(filtered);
  });

  function isCollapsed(key) {
    return !isSearching && collapsedKeys.has(key);
  }

  function toggleGroup(key) {
    const next = new Set(collapsedKeys);
    if (next.has(key)) next.delete(key);
    else next.add(key);
    collapsedKeys = next;
  }

  let allCollapsed = $derived(filteredGrouped.length > 0 && filteredGrouped.every(g => collapsedKeys.has(g.key)));

  function setAll(expanded) {
    collapsedKeys = expanded ? new Set() : new Set(filteredGrouped.map(g => g.key));
  }
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
      <div class="px-4 py-3 border-b border-border/60 flex items-center justify-between gap-2">
        <div class="flex items-center gap-2.5 min-w-0">
          <div class="w-7 h-7 rounded-lg bg-primary-subtle flex items-center justify-center shrink-0">
            <Users size={14} class="text-primary-light" />
          </div>
          <h3 class="text-sm font-semibold text-text-primary truncate">{labels.permissions}</h3>
          <span class="inline-flex items-center text-[11px] font-semibold px-1.5 py-0.5 rounded-full bg-primary/10 text-primary shrink-0">{rolePerms.length}</span>
        </div>
        {#if rolePerms.length > 0 && !isSearching && filteredGrouped.length > 1}
          <button
            type="button"
            class="text-xs font-semibold text-primary hover:text-primary-light transition-colors shrink-0"
            onclick={() => setAll(allCollapsed)}
          >
            {allCollapsed ? labels.expandAll : labels.collapseAll}
          </button>
        {/if}
      </div>
      {#if rolePerms.length > 0}
        <div class="p-3 border-b border-border/40">
          <SearchBar bind:value={permSearch} placeholder={labels.searchPermissions} inputClass="h-9 text-sm" />
        </div>
        <div class="p-3 space-y-2">
          {#if filteredGrouped.length > 0}
            {#each filteredGrouped as group (group.key)}
              {@const Icon = group.icon}
              {@const collapsed = isCollapsed(group.key)}
              <div class="rounded-xl border overflow-hidden transition-colors {collapsed ? 'border-border/70 bg-surface-subtle/30' : 'border-primary/20 bg-primary-subtle/10'}">
                <button
                  type="button"
                  class="w-full flex items-center gap-2.5 px-3 py-2.5 text-left transition-colors hover:bg-surface-hover/60"
                  aria-expanded={!collapsed}
                  aria-controls="perm-group-{group.key}"
                  onclick={() => toggleGroup(group.key)}
                >
                  <div class="w-6 h-6 rounded-md bg-primary-subtle flex items-center justify-center shrink-0">
                    <Icon size={12} class="text-primary-light" />
                  </div>
                  <span class="text-sm font-medium text-text-primary truncate flex-1">{group.label}</span>
                  <span class="inline-flex items-center text-[11px] font-semibold px-2 py-0.5 rounded-full bg-surface-default border border-border/60 text-text-secondary shrink-0">{group.permissions.length}</span>
                  <ChevronRight size={15} class="text-text-muted shrink-0 transition-transform duration-200 {collapsed ? '' : 'rotate-90'}" />
                </button>
                {#if !collapsed}
                  <div id="perm-group-{group.key}" class="grid grid-cols-2 gap-1.5 px-3 pt-2 pb-3" role="list">
                    {#each group.permissions as perm (perm.id)}
                      <div class="flex items-center gap-2 px-2.5 py-1.5 rounded-lg border border-border/50 bg-surface-default hover:border-primary/30 hover:bg-primary-subtle/20 transition-colors" role="listitem" title={perm.description || perm.code}>
                        <span class="w-1.5 h-1.5 rounded-full bg-primary/50 shrink-0"></span>
                        <span class="text-[13px] text-text-secondary truncate">{perm.name}</span>
                      </div>
                    {/each}
                  </div>
                {/if}
              </div>
            {/each}
          {:else}
            <div class="py-8 text-center">
              <div class="w-10 h-10 rounded-xl bg-surface-default border border-border mx-auto mb-2.5 flex items-center justify-center"><Search size={16} class="text-text-muted" /></div>
              <p class="text-sm text-text-muted">{labels.noResults}: "{permSearch}"</p>
            </div>
          {/if}
        </div>
      {:else}
        <div class="p-6 text-center">
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
