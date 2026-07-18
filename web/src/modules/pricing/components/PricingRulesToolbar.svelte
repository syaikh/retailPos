<script lang="ts">
  import { Button, SearchBar, Dropdown, BulkActionDropdown, FilterChipBar } from '$shared/ui';
  import { Plus, ChevronDown, Calculator, Columns3 } from 'lucide-svelte';
  import { debounce } from '$shared/utils/debounce';

  let {
    searchQuery = $bindable(''),
    approvalFilter = $bindable('all'),
    statusFilter = $bindable('all'),
    typeFilter = $bindable('all'),
    methodFilter = $bindable('all'),
    showDetailCols = $bindable(false),
    canCreate = false,
    pricingTypes = [],
    pricingMethods = [],
    typeLabel = 'Semua Tipe',
    methodLabel = 'Semua Metode',
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
    showDetailCols?: boolean;
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

  const approvalLabels: Record<string, string> = { draft: 'Draft', pending: 'Pending', approved: 'Approved', rejected: 'Rejected' };

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
      chips.push({ type: 'approval', label: `Approval: ${approvalLabels[approvalFilter] || approvalFilter}` });
    }
    if (statusFilter !== 'all') {
      const statusLabels: Record<string, string> = { active: 'Aktif', inactive: 'Nonaktif' };
      chips.push({ type: 'status', label: `Status: ${statusLabels[statusFilter] || statusFilter}` });
    }
    if (typeFilter !== 'all') {
      const label = pricingTypes.find(t => t.value === typeFilter)?.label || typeFilter;
      chips.push({ type: 'type', label: `Tipe: ${label}` });
    }
    if (methodFilter !== 'all') {
      const label = pricingMethods.find(m => m.value === methodFilter)?.label || methodFilter;
      chips.push({ type: 'method', label: `Metode: ${label}` });
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
      <SearchBar bind:value={searchQuery} placeholder="Cari rule..." oninput={handleSearch} inputClass="h-10" id="pricing-search" />
    </div>
    <BulkActionDropdown module="pricing_rules" canExport={canCreate} canImport={canCreate} onImport={onimport} />
    <Button variant="secondary" class="lg:hidden {showDetailCols ? 'bg-primary-subtle/20 text-primary-light border-primary-default/40' : ''}" onclick={() => showDetailCols = !showDetailCols}>
      <Columns3 size={14} /> <span class="hidden sm:inline">{showDetailCols ? 'Sembunyikan Detail' : 'Tampilkan Detail'}</span>
    </Button>
    {#if canCreate}
      <Button variant="secondary" onclick={onsimulate}>
        <Calculator size={14} /> <span class="hidden sm:inline">Simulasi</span>
      </Button>
      <Button variant="primary" class="shrink-0 shadow-glow-primary-sm px-5" onclick={oncreate}>
        <Plus size={18} /> Tambah Rule
      </Button>
    {/if}
  </div>

  <div class="flex flex-wrap items-center gap-2">
    <div class="flex items-center p-1 gap-1 bg-bg-secondary rounded-xl border border-border/30" role="group" aria-label="Filter status aktif">
      <button
        class="h-8 px-3 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'all' ? 'bg-primary-subtle text-primary-light border border-primary-default/20' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
        onclick={() => { statusFilter = 'all'; handleFilterChange(); }}
        aria-pressed={statusFilter === 'all'}
      >Semua</button>
      <button
        class="h-8 px-3 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'active' ? 'bg-success-subtle text-success-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
        onclick={() => { statusFilter = 'active'; handleFilterChange(); }}
        aria-pressed={statusFilter === 'active'}
      >Aktif</button>
      <button
        class="h-8 px-3 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'inactive' ? 'bg-danger-subtle text-danger-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
        onclick={() => { statusFilter = 'inactive'; handleFilterChange(); }}
        aria-pressed={statusFilter === 'inactive'}
      >Nonaktif</button>
    </div>
    <Dropdown placement="bottom-start" items={[
      { label: 'Semua Approval', checked: approvalFilter === 'all', onclick: () => { approvalFilter = 'all'; handleFilterChange(); } },
      { label: 'Draft', checked: approvalFilter === 'draft', onclick: () => { approvalFilter = 'draft'; handleFilterChange(); } },
      { label: 'Pending', checked: approvalFilter === 'pending', onclick: () => { approvalFilter = 'pending'; handleFilterChange(); } },
      { label: 'Approved', checked: approvalFilter === 'approved', onclick: () => { approvalFilter = 'approved'; handleFilterChange(); } },
      { label: 'Rejected', checked: approvalFilter === 'rejected', onclick: () => { approvalFilter = 'rejected'; handleFilterChange(); } }
    ]}>
      {#snippet trigger({ toggle })}
        <button
          class="flex items-center gap-2 px-3 h-8 rounded-lg border text-xs font-medium transition-colors {approvalFilter !== 'all' ? 'border-primary-default/40 bg-primary-subtle/30 text-text-primary' : 'border-border bg-surface-default text-text-secondary hover:border-border-strong hover:bg-surface-hover'}"
          onclick={toggle}
          aria-label="Filter approval: {approvalFilter === 'all' ? 'Semua' : approvalLabels[approvalFilter] || approvalFilter}"
        >
          <span class="flex-1 text-left truncate">{approvalFilter === 'all' ? 'Semua Approval' : approvalLabels[approvalFilter] || approvalFilter}</span>
          <ChevronDown size={12} class="text-text-muted shrink-0" />
        </button>
      {/snippet}
    </Dropdown>
    <Dropdown placement="bottom-start" items={[
      { label: 'Semua Tipe', checked: typeFilter === 'all', onclick: () => { typeFilter = 'all'; handleFilterChange(); } },
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
          aria-label="Filter tipe: {typeLabel}"
        >
          <span class="flex-1 text-left truncate">{typeLabel}</span>
          <ChevronDown size={12} class="text-text-muted shrink-0" />
        </button>
      {/snippet}
    </Dropdown>
    <Dropdown placement="bottom-start" items={[
      { label: 'Semua Metode', checked: methodFilter === 'all', onclick: () => { methodFilter = 'all'; handleFilterChange(); } },
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
          aria-label="Filter metode: {methodLabel}"
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
    clearLabel="Hapus semua"
  />
</div>
