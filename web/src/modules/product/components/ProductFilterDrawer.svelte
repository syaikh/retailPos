<script lang="ts">
  import { SearchBar } from '$shared/ui';
  import { X } from 'lucide-svelte';
  import { fade, fly } from 'svelte/transition';
  import { labels } from '$shared/i18n';
  import type { Brand } from '$modules/product/types';

  let {
    open = $bindable(false),
    categories = [],
    selectedCategories = $bindable<string[]>([]),
    brands = [] as Brand[],
    selectedBrandIDs = $bindable<number[]>([]),
    onClose,
    onApply,
  }: {
    open?: boolean;
    categories?: string[];
    selectedCategories?: string[];
    brands?: Brand[];
    selectedBrandIDs?: number[];
    onClose?: () => void;
    onApply?: () => void;
  } = $props();

  let activeTab = $state<'category' | 'brand'>('category');
  let searchQuery = $state('');

  let tempSelectedCategories = $state<string[]>([]);
  let tempSelectedBrandIDs = $state<number[]>([]);

  let filteredCategories = $derived(
    categories.filter(cat => cat !== 'All' && cat.toLowerCase().includes(searchQuery.toLowerCase()))
  );

  let filteredBrands = $derived(
    brands.filter(b => b.name.toLowerCase().includes(searchQuery.toLowerCase()))
  );

  $effect(() => {
    if (open) {
      tempSelectedCategories = selectedCategories.filter(c => c !== 'All');
      tempSelectedBrandIDs = [...selectedBrandIDs];
      searchQuery = '';
    }
  });

  let activeCount = $derived(
    tempSelectedCategories.filter(c => c !== 'All').length + tempSelectedBrandIDs.length
  );

  function toggleCategory(cat: string) {
    if (tempSelectedCategories.includes(cat)) {
      tempSelectedCategories = tempSelectedCategories.filter(c => c !== cat);
    } else {
      tempSelectedCategories = [...tempSelectedCategories, cat];
    }
  }

  function toggleBrand(id: number) {
    if (tempSelectedBrandIDs.includes(id)) {
      tempSelectedBrandIDs = tempSelectedBrandIDs.filter(i => i !== id);
    } else {
      tempSelectedBrandIDs = [...tempSelectedBrandIDs, id];
    }
  }

  function resetFilters() {
    tempSelectedCategories = [];
    tempSelectedBrandIDs = [];
  }

  function applyFilters() {
    selectedCategories = tempSelectedCategories.length > 0 ? [...tempSelectedCategories] : ['All'];
    selectedBrandIDs = tempSelectedBrandIDs.length > 0 ? [...tempSelectedBrandIDs] : [];
    onApply?.();
    open = false;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') open = false;
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
  <div
    class="fixed inset-0 z-50 bg-black/50"
    transition:fade={{ duration: 200 }}
  ></div>

  <div
    class="fixed right-0 top-0 h-full w-full max-w-md bg-surface border-l border-border flex flex-col shadow-2xl z-[55]"
    transition:fly={{ x: '100%', duration: 300 }}
    role="dialog"
    aria-modal="true"
    aria-label={labels.filterProduk}
    tabindex="-1"
  >
    <!-- Header -->
    <div class="flex items-center justify-between px-6 py-4 border-b border-border">
      <h2 class="text-lg font-semibold text-text-primary">{labels.filterProduk}</h2>
      <button type="button"
        onclick={() => open = false}
        class="p-2 rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors"
        aria-label={labels.close}
      >
        <X size={20} />
      </button>
    </div>

    <!-- Tabs -->
    <div class="flex border-b border-border">
      <button
        type="button"
        class="flex-1 px-4 py-3 text-sm font-medium transition-colors border-b-2 {activeTab === 'category' ? 'border-primary text-primary' : 'border-transparent text-text-muted hover:text-text-secondary'}"
        onclick={() => { activeTab = 'category'; searchQuery = ''; }}
      >
        {labels.category}
        {#if tempSelectedCategories.length > 0}
          <span class="ml-1.5 bg-primary/15 text-primary px-1.5 py-0.5 rounded-full text-xs">{tempSelectedCategories.length}</span>
        {/if}
      </button>
      <button
        type="button"
        class="flex-1 px-4 py-3 text-sm font-medium transition-colors border-b-2 {activeTab === 'brand' ? 'border-primary text-primary' : 'border-transparent text-text-muted hover:text-text-secondary'}"
        onclick={() => { activeTab = 'brand'; searchQuery = ''; }}
      >
        {labels.brand}
        {#if tempSelectedBrandIDs.length > 0}
          <span class="ml-1.5 bg-primary/15 text-primary px-1.5 py-0.5 rounded-full text-xs">{tempSelectedBrandIDs.length}</span>
        {/if}
      </button>
    </div>

    <!-- Content -->
    <div class="flex-1 overflow-y-auto px-6 py-4 space-y-6">
      <SearchBar bind:value={searchQuery} placeholder={activeTab === 'category' ? labels.cariNama : labels.searchBrand} inputClass="bg-surface border-border focus:ring-primary focus:border-transparent" />

      {#if activeTab === 'category'}
        <div>
          <h3 class="text-sm font-medium text-text-secondary mb-3">{labels.allCategoriesAZ}</h3>
          <div class="grid grid-cols-2 gap-2">
            {#each filteredCategories as cat}
              {@const isSelected = tempSelectedCategories.includes(cat)}
              <label
                class={`flex items-center gap-2 px-3 py-2 rounded-lg border cursor-pointer transition-all ${
                  isSelected
                    ? 'bg-primary/10 border-primary/30 text-primary'
                    : 'bg-surface border-border text-text-secondary hover:bg-surface-hover'
                }`}
              >
                <input
                  type="checkbox"
                  checked={isSelected}
                  onchange={() => toggleCategory(cat)}
                  class="w-4 h-4 rounded border-border text-primary focus:ring-primary"
                />
                <span class="text-sm truncate">{cat}</span>
              </label>
            {/each}
          </div>
          {#if filteredCategories.length === 0}
            <p class="text-sm text-text-muted text-center py-4">{labels.noCategoriesFound}</p>
          {/if}
        </div>
      {:else}
        <div>
          <h3 class="text-sm font-medium text-text-secondary mb-3">{labels.allBrandsAZ}</h3>
          <div class="grid grid-cols-2 gap-2">
            {#each filteredBrands as brand}
              {@const isSelected = tempSelectedBrandIDs.includes(brand.id)}
              <label
                class={`flex items-center gap-2 px-3 py-2 rounded-lg border cursor-pointer transition-all ${
                  isSelected
                    ? 'bg-primary/10 border-primary/30 text-primary'
                    : 'bg-surface border-border text-text-secondary hover:bg-surface-hover'
                }`}
              >
                <input
                  type="checkbox"
                  checked={isSelected}
                  onchange={() => toggleBrand(brand.id)}
                  class="w-4 h-4 rounded border-border text-primary focus:ring-primary"
                />
                <span class="text-sm truncate">{brand.name}</span>
              </label>
            {/each}
          </div>
          {#if filteredBrands.length === 0}
            <p class="text-sm text-text-muted text-center py-4">{labels.noBrandsFound}</p>
          {/if}
        </div>
      {/if}
    </div>

    <!-- Sticky Footer -->
    <div class="border-t border-border px-6 py-4 flex items-center justify-between">
      <button type="button"
        onclick={resetFilters}
        class="px-4 py-2 text-sm font-medium text-text-secondary hover:text-text-primary hover:bg-surface-hover rounded-lg transition-colors"
      >
        {labels.resetAll}
      </button>
      <button type="button"
        onclick={applyFilters}
        class="px-5 py-2 text-sm font-medium text-white bg-primary hover:bg-primary/90 rounded-lg transition-colors flex items-center gap-2"
      >
        {labels.applyFilter}
        {#if activeCount > 0}
          <span class="bg-white/20 px-1.5 py-0.5 rounded-full text-xs">{activeCount}</span>
        {/if}
      </button>
    </div>
  </div>
{/if}
