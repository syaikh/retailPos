<script lang="ts">
  import { Button, SearchBar, Dropdown, BulkActionDropdown, FilterChipBar } from '$shared/ui';
  import { Plus, ChevronDown, Calculator } from 'lucide-svelte';
  import { debounce } from '$shared/utils/debounce';
  import { labels } from '$shared/i18n';

  let {
    searchQuery = $bindable(''),
    approvalFilter = $bindable('all'),
    statusFilter = $bindable('all'),
    typeFilter = $bindable('all'),
    methodFilter = $bindable('all'),
    canCreate = false,
    pricingTypes = [],
    pricingMethods = [],
    typeLabel = labels.allTypes,
    methodLabel = labels.allMethods,
    oncreate = () => {},
    onfilter = () => {},
    onimport = () => {},
    onsimulate = () => {},
  }: {
    searchQuery: string;
    approvalFilter: string;
    statusFilter: string;
    typeFilter: string;
    methodFilter: string;
    canCreate: boolean;
    pricingTypes: { value: string; label: string }[];
    pricingMethods: { value: string; label: string }[];
    typeLabel: string;
    methodLabel: string;
    oncreate?: () => void;
    onfilter?: () => void;
    onimport?: () => void;
    onsimulate?: () => void;
  } = $props();

  const approvalLabels: Record<string, string> = { draft: labels.statusDraft, pending: labels.statusPending, approved: labels.statusApproved, rejected: labels.statusRejected };

  const debouncedSearch = debounce(() => {
    onfilter();
  }, 300);

  function handleSearch() {
    debouncedSearch();
  }

  function handleFilterChange() {
    onfilter();
  }

  let activeFilters = $derived.by(() => {
    const chips: { type: string; label: string }[] = [];
    if (approvalFilter !== 'all') {
      chips.push({ type: 'approval', label: labels.approvalChip.replace('{value}', approvalLabels[approvalFilter] || approvalFilter) });
    }
    if (statusFilter !== 'all') {
      const statusLabels: Record<string, string> = { active: labels.active, inactive: labels.inactive };
      chips.push({ type: 'status', label: labels.statusChip.replace('{value}', statusLabels[statusFilter] || statusFilter) });
    }
    if (typeFilter !== 'all') {
      const label = pricingTypes.find(t => t.value === typeFilter)?.label || typeFilter;
      chips.push({ type: 'type', label: labels.typeChip.replace('{value}', label) });
    }
    if (methodFilter !== 'all') {
      const label = pricingMethods.find(m => m.value === methodFilter)?.label || methodFilter;
      chips.push({ type: 'method', label: labels.methodChip.replace('{value}', label) });
    }
    return chips;
  });

  function clearFilterChip(type: string) {
    switch (type) {
      case 'approval': approvalFilter = 'all'; break;
      case 'status': statusFilter = 'all'; break;
      case 'type': typeFilter = 'all'; break;
      case 'method': methodFilter = 'all'; break;
    }
    handleFilterChange();
  }

  function clearAllFilters() {
    approvalFilter = 'all';
    statusFilter = 'all';
    typeFilter = 'all';
    methodFilter = 'all';
    handleFilterChange();
  }
</script>

<div class="card px-4 py-3">
  <div class="flex flex-wrap items-center gap-3 mb-3">
    <div class="flex-1 min-w-[200px]">
      <SearchBar bind:value={searchQuery} placeholder={labels.searchRules} oninput={handleSearch} inputClass="h-10" id="pricing-search" />
    </div>
    <BulkActionDropdown module="pricing_rules" canExport={canCreate} canImport={canCreate} onImport={onimport} />
    {#if canCreate}
      <Button variant="secondary" onclick={onsimulate}>
        <Calculator size={14} /> <span class="hidden sm:inline">{labels.simulation}</span>
      </Button>
      <Button variant="primary" class="shrink-0 shadow-glow-primary-sm px-5" onclick={oncreate}>
        <Plus size={18} /> {labels.addRule}
      </Button>
    {/if}
  </div>

  <div class="flex flex-wrap items-center gap-2">
    <div class="flex items-center p-1 gap-1 bg-bg-secondary rounded-xl border border-border/30" role="group" aria-label={labels.filterStatusActive}>
      <button
        class="h-8 px-3 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'all' ? 'bg-primary-subtle text-primary-light border border-primary-default/20' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
        onclick={() => { statusFilter = 'all'; handleFilterChange(); }}
        aria-pressed={statusFilter === 'all'}
      >{labels.all}</button>
      <button
        class="h-8 px-3 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'active' ? 'bg-success-subtle text-success-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
        onclick={() => { statusFilter = 'active'; handleFilterChange(); }}
        aria-pressed={statusFilter === 'active'}
      >{labels.active}</button>
      <button
        class="h-8 px-3 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'inactive' ? 'bg-danger-subtle text-danger-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
        onclick={() => { statusFilter = 'inactive'; handleFilterChange(); }}
        aria-pressed={statusFilter === 'inactive'}
      >{labels.inactive}</button>
    </div>
    <Dropdown placement="bottom-start" items={[
      { label: labels.allApproval, checked: approvalFilter === 'all', onclick: () => { approvalFilter = 'all'; handleFilterChange(); } },
      { label: labels.statusDraft, checked: approvalFilter === 'draft', onclick: () => { approvalFilter = 'draft'; handleFilterChange(); } },
      { label: labels.statusPending, checked: approvalFilter === 'pending', onclick: () => { approvalFilter = 'pending'; handleFilterChange(); } },
      { label: labels.statusApproved, checked: approvalFilter === 'approved', onclick: () => { approvalFilter = 'approved'; handleFilterChange(); } },
      { label: labels.statusRejected, checked: approvalFilter === 'rejected', onclick: () => { approvalFilter = 'rejected'; handleFilterChange(); } }
    ]}>
      {#snippet trigger({ toggle })}
        <button
          class="flex items-center gap-2 px-3 h-8 rounded-lg border text-xs font-medium transition-colors {approvalFilter !== 'all' ? 'border-primary-default/40 bg-primary-subtle/30 text-text-primary' : 'border-border bg-surface-default text-text-secondary hover:border-border-strong hover:bg-surface-hover'}"
          onclick={toggle}
          aria-label={labels.filterApproval.replace('{value}', approvalFilter === 'all' ? labels.all : approvalLabels[approvalFilter] || approvalFilter)}
        >
          <span class="flex-1 text-left truncate">{approvalFilter === 'all' ? labels.allApproval : approvalLabels[approvalFilter] || approvalFilter}</span>
          <ChevronDown size={12} class="text-text-muted shrink-0" />
        </button>
      {/snippet}
    </Dropdown>
    <Dropdown placement="bottom-start" items={[
      { label: labels.allTypes, checked: typeFilter === 'all', onclick: () => { typeFilter = 'all'; handleFilterChange(); } },
      ...pricingTypes.map(pt => ({
        label: pt.label,
        checked: typeFilter === pt.value,
        onclick: () => { typeFilter = pt.value; handleFilterChange(); }
      }))
    ]}>
      {#snippet trigger({ toggle })}
        <button
          class="flex items-center gap-2 px-3 h-8 rounded-lg border text-xs font-medium transition-colors {typeFilter !== 'all' ? 'border-primary-default/40 bg-primary-subtle/30 text-text-primary' : 'border-border bg-surface-default text-text-secondary hover:border-border-strong hover:bg-surface-hover'}"
          onclick={toggle}
          aria-label={labels.filterTipe.replace('{typeLabel}', typeLabel)}
        >
          <span class="flex-1 text-left truncate">{typeLabel}</span>
          <ChevronDown size={12} class="text-text-muted shrink-0" />
        </button>
      {/snippet}
    </Dropdown>
    <Dropdown placement="bottom-start" items={[
      { label: labels.allMethods, checked: methodFilter === 'all', onclick: () => { methodFilter = 'all'; handleFilterChange(); } },
      ...pricingMethods.map(pm => ({
        label: pm.label,
        checked: methodFilter === pm.value,
        onclick: () => { methodFilter = pm.value; handleFilterChange(); }
      }))
    ]}>
      {#snippet trigger({ toggle })}
        <button
          class="flex items-center gap-2 px-3 h-8 rounded-lg border text-xs font-medium transition-colors {methodFilter !== 'all' ? 'border-primary-default/40 bg-primary-subtle/30 text-text-primary' : 'border-border bg-surface-default text-text-secondary hover:border-border-strong hover:bg-surface-hover'}"
          onclick={toggle}
          aria-label={labels.filterMetode.replace('{methodLabel}', methodLabel)}
        >
          <span class="flex-1 text-left truncate">{methodLabel}</span>
          <ChevronDown size={12} class="text-text-muted shrink-0" />
        </button>
      {/snippet}
    </Dropdown>
  </div>

  <FilterChipBar
    chips={activeFilters}
    onclear={clearFilterChip}
    onclearall={clearAllFilters}
    clearLabel={labels.clearAll}
  />
</div>
