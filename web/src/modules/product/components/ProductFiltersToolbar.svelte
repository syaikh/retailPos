<script lang="ts">
  import { Button, SearchBar, BulkActionDropdown, Dropdown, FilterChipBar } from '$shared/ui';
  import { Plus, SlidersHorizontal, ChevronDown, AlertTriangle } from 'lucide-svelte';

  let {
    searchQuery = $bindable(''),
    selectedCategories = $bindable(['All']),
    categories = ['All'],
    filterStatus = $bindable('all'),
    lowStockOnly = $bindable(false),
    canManageInventory = false,
    canCreate = false,
    supplierFilterId = $bindable(null),
    supplierFilterName = $bindable(''),
    onsearch = () => {},
    onfiltercategory = () => {},
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
    canManageInventory?: boolean;
    canCreate?: boolean;
    supplierFilterId?: number | null;
    supplierFilterName?: string;
    onsearch?: () => void;
    onfiltercategory?: () => void;
    onrefresh?: () => void;
    onclearall?: () => void;
    onadd?: () => void;
    onImport?: () => void;
  } = $props();

  let statusLabel = $derived(
    filterStatus === 'all' ? 'All Status' : filterStatus.charAt(0).toUpperCase() + filterStatus.slice(1)
  );

  let categoryBtnStyle = $derived(selectedCategories.length > 0
    ? 'background: rgba(124,58,236,0.12); border-color: rgba(124,58,236,0.35); color: #c4b5fd;'
    : 'background: rgba(30,27,36,0.7); border-color: #374151; color: #9ca3af;'
  );

  let activeChips = $derived.by(() => {
    const chips: { type: string; label: string }[] = [];
    if (filterStatus !== 'all') {
      chips.push({ type: 'status', label: statusLabel });
    }
    if (selectedCategories.length > 0 && !(selectedCategories.length === 1 && selectedCategories[0] === 'All')) {
      chips.push({ type: 'category', label: `${selectedCategories.length} Kategori` });
    }
    if (lowStockOnly) {
      chips.push({ type: 'stock', label: 'Low Stock' });
    }
    if (supplierFilterId !== null) {
      chips.push({ type: 'supplier', label: `Supplier: ${supplierFilterName}` });
    }
    return chips;
  });

  function clearFilter(type: string) {
    if (type === 'status') filterStatus = 'all';
    if (type === 'category') selectedCategories = ['All'];
    if (type === 'stock') lowStockOnly = false;
    if (type === 'supplier') { supplierFilterId = null; supplierFilterName = ''; }
    onrefresh();
  }
</script>

<div class="card p-4">
  <div class="flex items-center gap-4">
    <div class="flex-2">
      <SearchBar bind:value={searchQuery} placeholder="Search by name, SKU, or barcode..." oninput={onsearch} inputClass="h-10" />
    </div>
    <button
      type="button"
      onclick={onfiltercategory}
      class="flex items-center gap-[9px] h-10 px-[14px] rounded-xl shrink-0 transition-all duration-200"
      style={categoryBtnStyle}
    >
      <SlidersHorizontal size={15} style="color: {selectedCategories.length > 0 ? '#c4b5fd' : '#9ca3af'}" />
      <span class="text-[13px] font-medium whitespace-nowrap">
        {#if selectedCategories.length > 0 && !(selectedCategories.length === 1 && selectedCategories[0] === 'All')}
          {selectedCategories.length} Kategori Dipilih
        {:else}
          Kategori
        {/if}
      </span>
      <ChevronDown size={13} class="shrink-0 transition-opacity duration-150" style="color: {selectedCategories.length > 0 ? '#c4b5fd' : '#9ca3af'}; opacity: {selectedCategories.length > 0 ? 0.7 : 0.4}" />
    </button>
    <Dropdown placement="bottom-start" items={[
      { label: 'All Status', checked: filterStatus === 'all', onclick: () => { filterStatus = 'all'; onrefresh(); } },
      { label: 'Active', checked: filterStatus === 'active', onclick: () => { filterStatus = 'active'; onrefresh(); } },
      { label: 'Inactive', checked: filterStatus === 'inactive', onclick: () => { filterStatus = 'inactive'; onrefresh(); } },
      { label: 'Archived', checked: filterStatus === 'archived', onclick: () => { filterStatus = 'archived'; onrefresh(); } },
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
      <span class="text-[13px] font-medium whitespace-nowrap">Low Stock</span>
    </button>
    <BulkActionDropdown module="products" canExport={canCreate} canImport={canCreate} {onImport} />
    <Button
      onclick={onadd}
      disabled={!canManageInventory}
      variant="primary"
      class="shrink-0 shadow-glow-primary-sm px-5 disabled:opacity-50 disabled:cursor-not-allowed"
      title={canManageInventory ? 'Add product' : 'Requires inventory role'}
    >
      <Plus size={18} />
      Add Product
    </Button>
  </div>

  <FilterChipBar chips={activeChips} onclear={clearFilter} onclearall={onclearall} clearLabel="Clear all" />
</div>
