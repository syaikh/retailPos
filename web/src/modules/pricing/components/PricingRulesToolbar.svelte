<script lang="ts">
  import { Button, SearchBar, Dropdown, BulkActionDropdown } from '$shared/ui';
  import { Plus, ChevronDown, X, Calculator } from 'lucide-svelte';
  import { debounce } from '$shared/utils/debounce';

  let {
    searchQuery = $bindable(''),
    statusFilter = $bindable('all'),
    typeFilter = $bindable('all'),
    methodFilter = $bindable('all'),
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
    const chips: { key: string; label: string; clear: () => void }[] = [];
    if (statusFilter !== 'all') {
      const statusLabels: Record<string, string> = { active: 'Aktif', inactive: 'Nonaktif' };
      chips.push({ key: 'status', label: `Status: ${statusLabels[statusFilter] || statusFilter}`, clear: () => { statusFilter = 'all'; handleFilterChange(); } });
    }
    if (typeFilter !== 'all') {
      const label = pricingTypes.find(t => t.value === typeFilter)?.label || typeFilter;
      chips.push({ key: 'type', label: `Tipe: ${label}`, clear: () => { typeFilter = 'all'; handleFilterChange(); } });
    }
    if (methodFilter !== 'all') {
      const label = pricingMethods.find(m => m.value === methodFilter)?.label || methodFilter;
      chips.push({ key: 'method', label: `Metode: ${label}`, clear: () => { methodFilter = 'all'; handleFilterChange(); } });
    }
    return chips;
  });

  let hasActiveFilters = $derived(activeFilters.length > 0);

  function clearAllFilters() {
    statusFilter = 'all';
    typeFilter = 'all';
    methodFilter = 'all';
    handleFilterChange();
  }
</script>

<div class="card p-4">
  <div class="flex items-center gap-3">
    <div class="flex-1">
      <SearchBar bind:value={searchQuery} placeholder="Cari rule..." oninput={handleSearch} inputClass="h-10" id="pricing-search" />
    </div>
    <div class="flex items-center p-1 gap-1 bg-bg-secondary rounded-xl border border-border-default" role="group" aria-label="Filter status">
      <button
        class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'all' ? 'bg-primary-subtle text-primary-light border border-primary-default/20' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
        onclick={() => { statusFilter = 'all'; handleFilterChange(); }}
        aria-pressed={statusFilter === 'all'}
      >Semua</button>
      <button
        class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'active' ? 'bg-success-subtle text-success-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
        onclick={() => { statusFilter = 'active'; handleFilterChange(); }}
        aria-pressed={statusFilter === 'active'}
      >Aktif</button>
      <button
        class="h-8 px-4 rounded-lg text-xs font-medium transition-all duration-200 {statusFilter === 'inactive' ? 'bg-danger-subtle text-danger-light' : 'text-text-muted hover:text-text-secondary hover:bg-surface-hover'}"
        onclick={() => { statusFilter = 'inactive'; handleFilterChange(); }}
        aria-pressed={statusFilter === 'inactive'}
      >Nonaktif</button>
    </div>
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
          class="flex items-center gap-2 px-3 h-10 rounded-xl border text-sm transition-colors {typeFilter !== 'all' ? 'border-primary-default/40 bg-primary-subtle/30 text-text-primary' : 'border-border bg-surface-default text-text-secondary hover:border-border-strong hover:bg-surface-hover'}"
          style="min-width: 130px;"
          onclick={toggle}
          aria-label="Filter tipe: {typeLabel}"
        >
          <span class="flex-1 text-left truncate">{typeLabel}</span>
          <ChevronDown size={14} class="text-text-muted shrink-0" />
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
          class="flex items-center gap-2 px-3 h-10 rounded-xl border text-sm transition-colors {methodFilter !== 'all' ? 'border-primary-default/40 bg-primary-subtle/30 text-text-primary' : 'border-border bg-surface-default text-text-secondary hover:border-border-strong hover:bg-surface-hover'}"
          style="min-width: 130px;"
          onclick={toggle}
          aria-label="Filter metode: {methodLabel}"
        >
          <span class="flex-1 text-left truncate">{methodLabel}</span>
          <ChevronDown size={14} class="text-text-muted shrink-0" />
        </button>
      {/snippet}
    </Dropdown>
    <BulkActionDropdown module="pricing_rules" canExport={canCreate} canImport={canCreate} onImport={onimport} />
    {#if canCreate}
      <Button variant="ghost" size="sm" class="text-text-muted hover:text-accent-light" onclick={onsimulate} aria-label="Simulasi harga">
        <Calculator size={16} />
      </Button>
      <Button variant="primary" class="shrink-0 shadow-glow-primary-sm px-5" onclick={oncreate}>
        <Plus size={18} /> Tambah Rule
      </Button>
    {/if}
  </div>

  {#if hasActiveFilters}
    <div class="flex items-center gap-2 mt-3 pt-3 border-t border-border/40">
      <span class="text-[11px] text-text-muted shrink-0">Filter aktif:</span>
      <div class="flex flex-wrap items-center gap-1.5">
        {#each activeFilters as chip (chip.key)}
          <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-primary-subtle/50 text-primary-light text-[11px] font-medium border border-primary-default/15">
            {chip.label}
            <button
              type="button"
              onclick={chip.clear}
              class="rounded-full p-0.5 hover:bg-primary-default/20 transition-colors"
              aria-label="Hapus filter {chip.label}"
            >
              <X size={10} />
            </button>
          </span>
        {/each}
        {#if activeFilters.length > 1}
          <button
            type="button"
            onclick={clearAllFilters}
            class="text-[11px] text-text-muted hover:text-danger transition-colors underline underline-offset-2"
          >
            Hapus semua
          </button>
        {/if}
      </div>
    </div>
  {/if}
</div>
