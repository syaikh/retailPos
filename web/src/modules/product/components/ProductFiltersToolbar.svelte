<script lang="ts">
  import { Button, SearchBar, ExportImportButtons } from '$shared/ui';
  import { Plus, SlidersHorizontal, ChevronDown, AlertTriangle, X } from 'lucide-svelte';

  let {
    searchQuery = $bindable(''),
    selectedCategories = $bindable(['All']),
    categories = ['All'],
    filterStatus = $bindable('all'),
    lowStockOnly = $bindable(false),
    canManageInventory = false,
    canCreate = false,
    onsearch = () => {},
    onfiltercategory = () => {},
    onrefresh = () => {},
    onclearall = () => {},
    onadd = () => {},
    onExport = (_format: 'csv' | 'xlsx') => {},
    onImport = () => {},
  }: {
    searchQuery?: string;
    selectedCategories?: string[];
    categories?: string[];
    filterStatus?: string;
    lowStockOnly?: boolean;
    canManageInventory?: boolean;
    canCreate?: boolean;
    onsearch?: () => void;
    onfiltercategory?: () => void;
    onrefresh?: () => void;
    onclearall?: () => void;
    onadd?: () => void;
    onExport?: (format: 'csv' | 'xlsx') => void;
    onImport?: () => void;
  } = $props();

  let showStatusDropdown = $state(false);

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
    return chips;
  });

  function clearFilter(type: string) {
    if (type === 'status') filterStatus = 'all';
    if (type === 'category') selectedCategories = ['All'];
    if (type === 'stock') lowStockOnly = false;
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
    <div class="relative shrink-0 status-filter-container">
      <button
        type="button"
        class="flex items-center gap-2 px-3 h-10 rounded-xl border transition-all duration-200 text-[13px] font-medium whitespace-nowrap {filterStatus !== 'all' ? 'bg-primary/10 border-primary/30 text-primary-light' : 'bg-surface-default border-border-strong text-text-muted hover:text-text-secondary hover:border-border-strong'}"
        onclick={() => showStatusDropdown = !showStatusDropdown}
      >
        <span>{statusLabel}</span>
        <ChevronDown size={14} class="text-text-muted shrink-0" />
      </button>
      {#if showStatusDropdown}
        <div class="absolute left-0 top-full mt-2 z-50 bg-surface-default border border-border rounded-lg shadow-xl py-1 min-w-[160px]" onclick={(e) => e.stopPropagation()} role="none" onkeydown={(e) => { if (e.key !== 'Escape') e.stopPropagation(); }}>
          <button
            class="w-full text-left px-4 py-2 text-sm transition-colors {filterStatus === 'all' ? 'text-primary-light bg-primary-subtle/30 font-medium' : 'text-text-secondary hover:bg-surface-hover'}"
            onclick={() => { filterStatus = 'all'; showStatusDropdown = false; onrefresh(); }}
          >All Status</button>
          {#each ['active', 'inactive', 'archived'] as status}
            <button
              class="w-full text-left px-4 py-2 text-sm transition-colors {filterStatus === status ? 'text-primary-light bg-primary-subtle/30 font-medium' : 'text-text-secondary hover:bg-surface-hover'}"
              onclick={() => { filterStatus = status; showStatusDropdown = false; onrefresh(); }}
            >
              {status.charAt(0).toUpperCase() + status.slice(1)}
            </button>
          {/each}
        </div>
      {/if}
    </div>
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
    <ExportImportButtons canExportImport={canCreate} {onExport} {onImport} />
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
