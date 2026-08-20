<script lang="ts">
  import { Button, SearchBar, BulkActionDropdown, Dropdown, FilterChipBar } from '$shared/ui';
  import { Plus, SlidersHorizontal, ChevronDown, AlertTriangle } from 'lucide-svelte';
  import { labels, t } from '$shared/i18n';

  let {
    searchQuery = $bindable(''),
    selectedCategories = $bindable(['All']),
    categories = ['All'],
    filterStatus = $bindable('all'),
    lowStockOnly = $bindable(false),
    canCreate = false,
    canExport = false,
    canImport = false,
    supplierFilterId = $bindable(null),
    supplierFilterName = $bindable(''),
    selectedBrandIDs = $bindable<number[]>([]),
    onsearch = () => {},
    onfilter = () => {},
    onrefresh = () => {},
    onclearall = () => {},
    onadd = () => {},
    onImport = () => {},
  }: {
    searchQuery?: string;
    selectedCategories?: string[];
    categories?: string[];
    filterStatus?: string;
    lowStockOnly?: boolean;
    canCreate?: boolean;
    canExport?: boolean;
    canImport?: boolean;
    supplierFilterId?: number | null;
    supplierFilterName?: string;
    selectedBrandIDs?: number[];
    onsearch?: () => void;
    onfilter?: () => void;
    onrefresh?: () => void;
    onclearall?: () => void;
    onadd?: () => void;
    onImport?: () => void;
  } = $props();

  let statusLabel = $derived(
    filterStatus === 'all' ? labels.allStatus
    : filterStatus === 'active' ? labels.active
    : filterStatus === 'inactive' ? labels.inactive
    : filterStatus === 'archived' ? labels.archived
    : filterStatus.charAt(0).toUpperCase() + filterStatus.slice(1)
  );

  let hasActiveFilters = $derived(
    (selectedCategories.length > 0 && !(selectedCategories.length === 1 && selectedCategories[0] === 'All'))
    || selectedBrandIDs.length > 0
  );

  let activeChips = $derived.by(() => {
    const chips: { type: string; label: string }[] = [];
    if (filterStatus !== 'all') {
      chips.push({ type: 'status', label: statusLabel });
    }
    if (selectedCategories.length > 0 && !(selectedCategories.length === 1 && selectedCategories[0] === 'All')) {
      chips.push({ type: 'category', label: t('categoriesCount', { count: selectedCategories.length }) });
    }
    if (lowStockOnly) {
      chips.push({ type: 'stock', label: labels.lowStock });
    }
    if (supplierFilterId !== null) {
      chips.push({ type: 'supplier', label: t('supplierWithName', { name: supplierFilterName }) });
    }
    if (selectedBrandIDs.length > 0) {
      chips.push({ type: 'brand', label: t('brandsCount', { count: selectedBrandIDs.length }) });
    }
    return chips;
  });

  function clearFilter(type: string) {
    if (type === 'status') filterStatus = 'all';
    if (type === 'category') selectedCategories = ['All'];
    if (type === 'stock') lowStockOnly = false;
    if (type === 'supplier') { supplierFilterId = null; supplierFilterName = ''; }
    if (type === 'brand') { selectedBrandIDs = []; }
    onrefresh();
  }
</script>

<div class="card p-4">
  <div class="flex items-center gap-3">
    <div class="flex-2">
      <SearchBar bind:value={searchQuery} placeholder={labels.searchByNameSkuBarcode} oninput={onsearch} inputClass="h-10" />
    </div>
    <button
      type="button"
      onclick={onfilter}
      class="flex items-center gap-[9px] h-10 px-[14px] rounded-xl shrink-0 transition-all duration-200 border {hasActiveFilters ? 'bg-primary/10 border-primary/30 text-primary-light' : 'bg-surface-default border-border-strong text-text-muted hover:text-text-secondary hover:border-border-strong'}"
    >
      <SlidersHorizontal size={15} class={hasActiveFilters ? 'text-primary-light' : 'text-text-muted'} />
      <span class="text-[13px] font-medium whitespace-nowrap">
        {#if hasActiveFilters}
          {#if selectedCategories.length > 0 && !(selectedCategories.length === 1 && selectedCategories[0] === 'All')}
            {#if selectedBrandIDs.length > 0}
              {t('categoriesCount', { count: selectedCategories.length })} + {t('brandsCount', { count: selectedBrandIDs.length })}
            {:else}
              {t('categoriesSelectedCount', { count: selectedCategories.length })}
            {/if}
          {:else if selectedBrandIDs.length > 0}
            {t('brandsSelectedCount', { count: selectedBrandIDs.length })}
          {/if}
        {:else}
          {labels.filter}
        {/if}
      </span>
      <ChevronDown size={13} class="shrink-0 transition-opacity duration-150 {hasActiveFilters ? 'text-primary-light' : 'text-text-muted'}" />
    </button>
    <Dropdown placement="bottom-start" items={[
      { label: labels.allStatus, checked: filterStatus === 'all', onclick: () => { filterStatus = 'all'; onrefresh(); } },
      { label: labels.active, checked: filterStatus === 'active', onclick: () => { filterStatus = 'active'; onrefresh(); } },
      { label: labels.inactive, checked: filterStatus === 'inactive', onclick: () => { filterStatus = 'inactive'; onrefresh(); } },
      { label: labels.archived, checked: filterStatus === 'archived', onclick: () => { filterStatus = 'archived'; onrefresh(); } },
    ]}>
      {#snippet trigger({ toggle })}
        <button
          type="button"
          class="flex items-center gap-2 px-3 h-10 rounded-xl border transition-all duration-200 text-[13px] font-medium whitespace-nowrap {filterStatus !== 'all' ? 'bg-primary/10 border-primary/30 text-primary-light' : 'bg-surface-default border-border-strong text-text-muted hover:text-text-secondary hover:border-border-strong'}"
          onclick={toggle}
        >
          <span>{statusLabel}</span>
          <ChevronDown size={14} class="text-text-muted shrink-0" />
        </button>
      {/snippet}
    </Dropdown>
    <button
      type="button"
      role="switch"
      aria-checked={lowStockOnly}
      onclick={() => { lowStockOnly = !lowStockOnly; onrefresh(); }}
      class="flex items-center gap-[9px] h-10 px-[14px] rounded-xl shrink-0 transition-all duration-200 border {lowStockOnly ? 'bg-warning/10 border-warning/30 text-warning-light' : 'bg-surface-default border-border-strong text-text-muted hover:text-text-secondary hover:border-border-strong'}"
    >
      <AlertTriangle size={14} class={lowStockOnly ? 'text-warning-light' : 'text-text-muted'} />
      <span class="text-[13px] font-medium whitespace-nowrap">{labels.lowStock}</span>
    </button>
    <BulkActionDropdown module="products" canExport={canExport} canImport={canImport} {onImport} />
    <Button
      onclick={onadd}
      disabled={!canCreate}
      variant="primary"
      class="shrink-0 shadow-glow-primary-sm px-5 disabled:opacity-50 disabled:cursor-not-allowed"
      title={canCreate ? labels.addProductTitle : labels.requiresProductCreatePermission}
    >
      <Plus size={18} />
      {labels.tambahProduk}
    </Button>
  </div>

  <FilterChipBar chips={activeChips} onclear={clearFilter} onclearall={onclearall} clearLabel={labels.clearAll} />
</div>
