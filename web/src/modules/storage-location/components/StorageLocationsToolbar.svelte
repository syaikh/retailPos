<script lang="ts">
  import { Button, SearchBar, FilterChipBar } from '$shared/ui';
  import { Plus } from 'lucide-svelte';
  import { labels } from '$shared/i18n';

  let {
    searchQuery = $bindable(''),
    statusFilter = $bindable('all'),
    canCreate = false,
    onsearch = () => {},
    onstatuschange = () => {},
    oncreate = () => {},
  }: {
    searchQuery?: string;
    statusFilter?: string;
    canCreate?: boolean;
    onsearch?: () => void;
    onstatuschange?: () => void;
    oncreate?: () => void;
  } = $props();

  let activeFilters = $derived.by(() => {
    const chips: { type: string; label: string }[] = [];
    if (searchQuery.trim()) {
      chips.push({ type: 'search', label: labels.filterChipSearch.replace('{q}', searchQuery.trim()) });
    }
    if (statusFilter !== 'all') {
      const labelMap: Record<string, string> = { active: labels.active, inactive: labels.inactive };
      chips.push({ type: 'status', label: labels.filterChipStatus.replace('{s}', labelMap[statusFilter] || statusFilter) });
    }
    return chips;
  });

  function clearFilterChip(type: string) {
    if (type === 'search') searchQuery = '';
    else if (type === 'status') statusFilter = 'all';
    onstatuschange();
  }

  function clearAllFilters() {
    searchQuery = '';
    statusFilter = 'all';
    onstatuschange();
  }
</script>

<div class="card px-4 py-3">
    <div class="flex flex-wrap items-center gap-3 mb-3">
      <div class="flex-1 min-w-[200px]">
        <SearchBar bind:value={searchQuery} placeholder={labels.cariKodeAtauNamaLokasi} oninput={onsearch} inputClass="h-10" id="storage-location-search" />
      </div>
      {#if canCreate}
        <Button variant="primary" class="shrink-0 shadow-glow-primary-sm px-5" onclick={oncreate}>
          <Plus size={18} /> {labels.tambahLokasi}
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
    </div>

    <FilterChipBar
      chips={activeFilters}
      onclear={clearFilterChip}
      onclearall={clearAllFilters}
      clearLabel={labels.removeAll}
    />
  </div>
