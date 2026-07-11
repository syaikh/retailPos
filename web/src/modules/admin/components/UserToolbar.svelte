<script lang="ts">
  import { Button, SearchBar, Dropdown, FilterChipBar } from '$shared/ui';
  import { Plus, ChevronDown, SlidersHorizontal } from 'lucide-svelte';

  let {
    searchQuery = $bindable(''),
    roles = [],
    filterRole = $bindable('all'),
    filterStatus = $bindable('all'),
    canCreate = false,
    onadd = () => {},
    onclearall = () => {},
  }: {
    searchQuery?: string;
    roles?: any[];
    filterRole?: string;
    filterStatus?: string;
    canCreate?: boolean;
    onadd?: () => void;
    onclearall?: () => void;
  } = $props();

  let roleLabel = $derived(filterRole === 'all' ? 'All Roles' : roles.find(r => String(r.id) === filterRole)?.name || filterRole);
  let statusLabel = $derived(filterStatus === 'all' ? 'All Status' : filterStatus === 'true' ? 'Active' : 'Inactive');

  let activeChips = $derived.by(() => {
    const chips: { type: string; label: string }[] = [];
    if (filterRole !== 'all') {
      const r = roles.find(role => String(role.id) === filterRole);
      chips.push({ type: 'role', label: r ? r.name : filterRole });
    }
    if (filterStatus !== 'all') {
      chips.push({ type: 'status', label: filterStatus === 'true' ? 'Active' : 'Inactive' });
    }
    return chips;
  });

  function clearFilter(type: string) {
    if (type === 'role') filterRole = 'all';
    if (type === 'status') filterStatus = 'all';
  }
</script>

<div class="card p-4">
  <div class="flex items-center gap-3">
    <div class="flex-1">
      <SearchBar bind:value={searchQuery} placeholder="Search by username or email…" />
    </div>
    <Dropdown placement="bottom-start" menuClass="p-2 min-w-[360px] max-h-64 overflow-y-auto">
      {#snippet trigger({ toggle })}
        <button
          class="flex items-center gap-2 px-3 h-10 rounded-xl border border-border bg-surface-default text-sm hover:border-border-strong hover:bg-surface-hover transition-colors {filterRole !== 'all' ? 'text-text-primary' : 'text-text-secondary'}"
          style="min-width: 140px;"
          onclick={toggle}
        >
          <span class="flex-1 text-left truncate">{roleLabel}</span>
          <ChevronDown size={14} class="text-text-muted shrink-0" />
        </button>
      {/snippet}
      {#snippet content({ close })}
        <button
          class="w-full text-left px-3 py-2 text-sm rounded-lg transition-colors truncate {filterRole === 'all' ? 'text-primary-light bg-primary-subtle/30 font-medium' : 'text-text-secondary hover:bg-surface-hover'}"
          onclick={() => { filterRole = 'all'; close(); }}
        >All Roles</button>
        <div class="grid grid-cols-2 gap-1 mt-1">
          {#each roles as role}
            <button
              class="text-left px-3 py-2 text-sm rounded-lg transition-colors truncate {filterRole === String(role.id) ? 'text-primary-light bg-primary-subtle/30 font-medium' : 'text-text-secondary hover:bg-surface-hover'}"
              onclick={() => { filterRole = String(role.id); close(); }}
              title={role.name}
            >{role.name}</button>
          {/each}
        </div>
      {/snippet}
    </Dropdown>
    <Dropdown placement="bottom-start" items={[
      { label: 'All Status', checked: filterStatus === 'all', onclick: () => filterStatus = 'all' },
      { label: 'Active', checked: filterStatus === 'true', onclick: () => filterStatus = 'true' },
      { label: 'Inactive', checked: filterStatus === 'false', onclick: () => filterStatus = 'false' },
    ]}>
      {#snippet trigger({ toggle })}
        <button
          class="flex items-center gap-2 px-3 h-10 w-32 rounded-xl border border-border bg-surface-default text-sm hover:border-border-strong hover:bg-surface-hover transition-colors {filterStatus !== 'all' ? 'text-text-primary' : 'text-text-secondary'}"
          onclick={toggle}
        >
          <span class="flex-1 text-left truncate">{statusLabel}</span>
          <ChevronDown size={14} class="text-text-muted shrink-0" />
        </button>
      {/snippet}
    </Dropdown>
    {#if canCreate}
      <Button onclick={onadd} variant="primary" class="shrink-0 shadow-glow-primary-sm px-5">
        <Plus size={18} />
        Add User
      </Button>
    {/if}
  </div>
  <FilterChipBar chips={activeChips} onclear={clearFilter} onclearall={onclearall} clearLabel="Clear all" />
</div>
