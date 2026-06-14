<script lang="ts">
  import { X, Search, Check } from 'lucide-svelte';
  import { fade, fly } from 'svelte/transition';
  let {
    open = $bindable(false),
    categories = [],
    selectedCategories = $bindable([]),
    popularCategories = [],
    onClose,
    onApply,
  } = $props();

  let searchQuery = $state('');

  // Temp state - only modified inside modal, committed to `selectedCategories` on Apply
  let tempSelectedCategories = $state([]);

  let filteredCategories = $derived(
    categories.filter(cat => cat !== 'All' && cat.toLowerCase().includes(searchQuery.toLowerCase()))
  );

  // Sync tempSelectedCategories from selectedCategories when modal opens,
  // preserving selectedCategories in parent state for persistence outside modal
  $effect(() => {
    if (open) {
      tempSelectedCategories = selectedCategories.filter(c => c !== 'All');
    }
  });

  // Active count excludes 'All' (used for button counter)
  let activeCount = $derived(
    tempSelectedCategories.filter(c => c !== 'All').length
  );

  function toggleCategory(cat) {
    if (tempSelectedCategories.includes(cat)) {
      tempSelectedCategories = tempSelectedCategories.filter(c => c !== cat);
    } else {
      tempSelectedCategories = [...tempSelectedCategories, cat];
    }
  }

  function togglePopularCategory(cat) {
    toggleCategory(cat);
  }

  function resetFilters() {
    tempSelectedCategories = [];
  }

  function applyFilters() {
    // Commit temp state to parent, then close
    selectedCategories = tempSelectedCategories.length > 0 ? [...tempSelectedCategories] : [];
    onApply?.(selectedCategories);
    open = false;
  }

  // Only allow Escape key to close - remove backdrop click handler
  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') open = false;
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
  <!-- Backdrop - no click handler to prevent closing when clicking outside -->
  <div
    class="fixed inset-0 z-50 bg-black/50 "
    transition:fade={{ duration: 200 }}
  ></div>

  <!-- Side Drawer - separate from backdrop -->
  <div
    class="fixed right-0 top-0 h-full w-full max-w-md bg-surface border-l border-border flex flex-col shadow-2xl z-[55]"
    transition:fly={{ x: '100%', duration: 300 }}
    role="dialog"
    aria-modal="true"
    aria-label="Filter Kategori"
    tabindex="-1"
  >
    <!-- Header -->
    <div class="flex items-center justify-between px-6 py-4 border-b border-border">
      <h2 class="text-lg font-semibold text-text-primary">Filter Produk</h2>
      <button
        onclick={() => open = false}
        class="p-2 rounded-lg text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors"
        aria-label="Close"
      >
        <X size={20} />
      </button>
    </div>

    <!-- Content -->
    <div class="flex-1 overflow-y-auto px-6 py-4 space-y-6">
      <!-- Search Bar -->
      <div class="relative">
        <Search size={18} class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" />
        <input
          type="text"
          placeholder="Cari kategori..."
          bind:value={searchQuery}
          class="w-full h-10 pl-10 pr-4 rounded-lg border border-border bg-surface focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
        />
      </div>

      <!-- Popular Categories Chips -->
      {#if popularCategories.length > 0}
        <div>
          <h3 class="text-sm font-medium text-text-secondary mb-3">Kategori Populer</h3>
          <div class="flex flex-wrap gap-2">
            {#each popularCategories as cat}
              {@const isSelected = tempSelectedCategories.includes(cat)}
              <button
                type="button"
                onclick={() => togglePopularCategory(cat)}
                class={`inline-flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-full transition-all ${
                  isSelected
                    ? 'bg-primary text-white'
                    : 'bg-surface-subtle text-text-secondary hover:bg-surface-hover'
                }`}
              >
                {#if isSelected}
                  <Check size={14} />
                {/if}
                {cat}
              </button>
            {/each}
          </div>
        </div>
      {/if}

      <!-- All Categories Grid -->
      <div>
        <h3 class="text-sm font-medium text-text-secondary mb-3">Semua Kategori (A-Z)</h3>
        <div class="grid grid-cols-2 md:grid-cols-3 gap-2">
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
          <p class="text-sm text-text-muted text-center py-4">Tidak ada kategori ditemukan</p>
        {/if}
      </div>
    </div>

    <!-- Sticky Footer -->
    <div class="border-t border-border px-6 py-4 flex items-center justify-between">
      <button
        type="button"
        onclick={resetFilters}
        class="px-4 py-2 text-sm font-medium text-text-secondary hover:text-text-primary hover:bg-surface-hover rounded-lg transition-colors"
      >
        Reset Semua
      </button>
      <button
        type="button"
        onclick={applyFilters}
        class="px-5 py-2 text-sm font-medium text-white bg-primary hover:bg-primary/90 rounded-lg transition-colors flex items-center gap-2"
      >
        Terapkan Filter
        {#if activeCount > 0}
          <span class="bg-white/20 px-1.5 py-0.5 rounded-full text-xs">{activeCount}</span>
        {/if}
      </button>
    </div>
  </div>
{/if}