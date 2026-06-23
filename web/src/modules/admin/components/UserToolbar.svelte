<script lang="ts">
  import { Button, SearchBar } from '$shared/ui';
  import { Plus, ChevronDown, SlidersHorizontal, X } from 'lucide-svelte';

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

  let showRoleDropdown = $state(false);
  let showStatusDropdown = $state(false);

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
      <Button onclick={onadd} variant="primary" class="shrink-0 shadow-glow-primary-sm px-5">
        <Plus size={18} />
        Add User
      </Button>
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
              aria-label="Hapus filter"
            >
              <X size={12} />
            </button>
          </div>
        {/each}
        <button
          class="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-text-muted hover:text-text-primary bg-surface-default/50 border border-border/50 rounded-full transition-colors"
          onclick={onclearall}
        >
          Clear all
          <X size={12} />
        </button>
      </div>
    </div>
  </div>
</div>

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
