<script lang="ts">
  import { Button, SearchBar, BulkActionDropdown, FilterChipBar } from '$shared/ui';
  import { Plus, Users, UsersRound } from 'lucide-svelte';
  import { labels, t } from '$shared/i18n';

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
      chips.push({ type: 'search', label: t('filterChipSearch', { q: searchQuery.trim() }) });
    }
    if (statusFilter !== 'all') {
      const statusLabels: Record<string, string> = { active: labels.active, inactive: labels.inactive };
      chips.push({ type: 'status', label: t('filterChipStatus', { s: statusLabels[statusFilter] || statusFilter }) });
    }
    if (hasCustomersFilter !== 'all') {
      chips.push({ type: 'has_customers', label: hasCustomersFilter === 'yes' ? labels.filterChipHasCustomers : labels.filterChipNoCustomers });
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
        <SearchBar bind:value={searchQuery} placeholder={labels.cariNamaGroup} oninput={onsearch} inputClass="h-10" id="customer-group-search" />
      </div>
      <BulkActionDropdown module="customer_groups" canExport={canCreate} canImport={canCreate} onImport={onimport} />
      {#if canCreate}
        <Button variant="primary" class="shrink-0 shadow-glow-primary-sm px-5" onclick={oncreate}>
          <Plus size={18} /> {labels.tambahGroup}
        </Button>
      {/if}
    </div>

    <div class="flex flex-wrap items-center gap-3">
      <div class="flex items-center p-1 gap-1 bg-bg-secondary rounded-xl border border-border/30" role="group" aria-label={labels.filterStatus}>
        <button
          class="h-8 px-3 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'all' ? 'bg-primary-subtle text-primary-light border border-primary-default/20' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
          onclick={() => { statusFilter = 'all'; onstatuschange(); }}
          aria-pressed={statusFilter === 'all'}
        >{labels.semua}</button>
        <button
          class="h-8 px-3 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'active' ? 'bg-success-subtle text-success-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
          onclick={() => { statusFilter = 'active'; onstatuschange(); }}
          aria-pressed={statusFilter === 'active'}
        >{labels.active}</button>
        <button
          class="h-8 px-3 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'inactive' ? 'bg-danger-subtle text-danger-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
          onclick={() => { statusFilter = 'inactive'; onstatuschange(); }}
          aria-pressed={statusFilter === 'inactive'}
        >{labels.inactive}</button>
      </div>

      <div class="flex items-center p-1 gap-1 bg-bg-secondary rounded-xl border border-border/30" role="group" aria-label={labels.filterCustomer}>
        <button
          class="h-8 px-3 rounded-lg text-xs font-medium transition-all duration-200 flex items-center gap-1.5 {hasCustomersFilter === 'all' ? 'bg-primary-subtle text-primary-light border border-primary-default/20' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
          onclick={() => { hasCustomersFilter = 'all'; onstatuschange(); }}
          aria-pressed={hasCustomersFilter === 'all'}
        >{labels.semua}</button>
        <button
          class="h-8 px-3 rounded-lg text-xs font-medium transition-all duration-200 flex items-center gap-1.5 {hasCustomersFilter === 'yes' ? 'bg-success-subtle text-success-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
          onclick={() => { hasCustomersFilter = 'yes'; onstatuschange(); }}
          aria-pressed={hasCustomersFilter === 'yes'}
        ><Users size={14} /> {labels.hasCustomers}</button>
        <button
          class="h-8 px-3 rounded-lg text-xs font-medium transition-all duration-200 flex items-center gap-1.5 {hasCustomersFilter === 'no' ? 'bg-danger-subtle text-danger-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
          onclick={() => { hasCustomersFilter = 'no'; onstatuschange(); }}
          aria-pressed={hasCustomersFilter === 'no'}
        ><UsersRound size={14} /> {labels.noCustomers}</button>
      </div>
    </div>

    <FilterChipBar
      chips={activeFilters}
      onclear={clearFilterChip}
      onclearall={clearAllFilters}
      clearLabel={labels.clearAll}
    />
  </div>
