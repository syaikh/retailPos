<script lang="ts">
  import { Button, SearchBar, BulkActionDropdown, FilterChipBar } from '$shared/ui';
  import { Plus, Users, UsersRound } from 'lucide-svelte';

  let {
    searchQuery = $bindable(''),
    statusFilter = $bindable('all'),
    hasCustomersFilter = $bindable('all'),
    canCreate = false,
    onsearch = () => {},
    onstatuschange = () => {},
    oncreate = () => {},
    onimport = () => {},
  }: {
    searchQuery?: string;
    statusFilter?: string;
    hasCustomersFilter?: string;
    canCreate?: boolean;
    onsearch?: () => void;
    onstatuschange?: () => void;
    oncreate?: () => void;
    onimport?: () => void;
  } = $props();

  let activeFilters = $derived.by(() => {
    const chips: { type: string; label: string }[] = [];
    if (searchQuery.trim()) {
      chips.push({ type: 'search', label: `Pencarian: ${searchQuery.trim()}` });
    }
    if (statusFilter !== 'all') {
      const labels: Record<string, string> = { active: 'Aktif', inactive: 'Nonaktif' };
      chips.push({ type: 'status', label: `Status: ${labels[statusFilter] || statusFilter}` });
    }
    if (hasCustomersFilter !== 'all') {
      chips.push({ type: 'has_customers', label: hasCustomersFilter === 'yes' ? 'Punya Customer' : 'Tanpa Customer' });
    }
    return chips;
  });

  function clearFilterChip(type: string) {
    if (type === 'search') searchQuery = '';
    else if (type === 'status') statusFilter = 'all';
    else if (type === 'has_customers') hasCustomersFilter = 'all';
    onstatuschange();
  }

  function clearAllFilters() {
    searchQuery = '';
    statusFilter = 'all';
    hasCustomersFilter = 'all';
    onstatuschange();
  }
</script>

<div class="card px-4 py-3">
    <div class="flex flex-wrap items-center gap-3 mb-3">
      <div class="flex-1 min-w-[200px]">
        <SearchBar bind:value={searchQuery} placeholder="Cari nama group..." oninput={onsearch} inputClass="h-10" id="customer-group-search" />
      </div>
      <BulkActionDropdown module="customer_groups" canExport={canCreate} canImport={canCreate} onImport={onimport} />
      {#if canCreate}
        <Button variant="primary" class="shrink-0 shadow-glow-primary-sm px-5" onclick={oncreate}>
          <Plus size={18} /> Tambah Group
        </Button>
      {/if}
    </div>

    <div class="flex flex-wrap items-center gap-3">
      <div class="flex items-center p-1 gap-1 bg-bg-secondary rounded-xl border border-border/30" role="group" aria-label="Filter status">
        <button
          class="h-8 px-3 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'all' ? 'bg-primary-subtle text-primary-light border border-primary-default/20' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
          onclick={() => { statusFilter = 'all'; onstatuschange(); }}
          aria-pressed={statusFilter === 'all'}
        >Semua</button>
        <button
          class="h-8 px-3 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'active' ? 'bg-success-subtle text-success-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
          onclick={() => { statusFilter = 'active'; onstatuschange(); }}
          aria-pressed={statusFilter === 'active'}
        >Aktif</button>
        <button
          class="h-8 px-3 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'inactive' ? 'bg-danger-subtle text-danger-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
          onclick={() => { statusFilter = 'inactive'; onstatuschange(); }}
          aria-pressed={statusFilter === 'inactive'}
        >Nonaktif</button>
      </div>

      <div class="flex items-center p-1 gap-1 bg-bg-secondary rounded-xl border border-border/30" role="group" aria-label="Filter customer">
        <button
          class="h-8 px-3 rounded-lg text-xs font-medium transition-all duration-200 flex items-center gap-1.5 {hasCustomersFilter === 'all' ? 'bg-primary-subtle text-primary-light border border-primary-default/20' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
          onclick={() => { hasCustomersFilter = 'all'; onstatuschange(); }}
          aria-pressed={hasCustomersFilter === 'all'}
        >Semua</button>
        <button
          class="h-8 px-3 rounded-lg text-xs font-medium transition-all duration-200 flex items-center gap-1.5 {hasCustomersFilter === 'yes' ? 'bg-success-subtle text-success-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
          onclick={() => { hasCustomersFilter = 'yes'; onstatuschange(); }}
          aria-pressed={hasCustomersFilter === 'yes'}
        ><Users size={14} /> Ada</button>
        <button
          class="h-8 px-3 rounded-lg text-xs font-medium transition-all duration-200 flex items-center gap-1.5 {hasCustomersFilter === 'no' ? 'bg-danger-subtle text-danger-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
          onclick={() => { hasCustomersFilter = 'no'; onstatuschange(); }}
          aria-pressed={hasCustomersFilter === 'no'}
        ><UsersRound size={14} /> Kosong</button>
      </div>
    </div>

    <FilterChipBar
      chips={activeFilters}
      onclear={clearFilterChip}
      onclearall={clearAllFilters}
      clearLabel="Hapus semua"
    />
  </div>
