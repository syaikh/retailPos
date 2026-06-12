<script lang="ts">
  import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight } from 'lucide-svelte';

  let { 
    total = 0, 
    limit = 20, 
    offset = 0, 
    onPageChange 
  }: {
    total: number;
    limit: number;
    offset: number;
    onPageChange: (newOffset: number, newLimit: number) => void;
  } = $props();

  const currentPage = $derived(Math.floor(offset / limit) + 1);
  const totalPages = $derived(Math.max(1, Math.ceil(total / limit)));
  
  const canPrev = $derived(currentPage > 1);
  const canNext = $derived(currentPage < totalPages);

  let pageInput = $state<string>('');
  let editing = $derived(pageInput !== '');

  function goToPage(page: number) {
    if (page < 1 || page > totalPages) return;
    onPageChange((page - 1) * limit, limit);
  }

  function handleLimitChange(e: Event) {
    const newLimit = parseInt((e.target as HTMLSelectElement).value);
    onPageChange(0, newLimit);
  }

  function startEdit() {
    pageInput = String(currentPage);
  }

  function cancelEdit() {
    pageInput = '';
  }

  function submitEdit() {
    const parsed = parseInt(pageInput);
    if (isNaN(parsed) || parsed < 1) {
      goToPage(1);
    } else if (parsed > totalPages) {
      goToPage(totalPages);
    } else {
      goToPage(parsed);
    }
    pageInput = '';
  }

  function handleInput(e: Event) {
    const val = (e.target as HTMLInputElement).value;
    // Allow only digits
    pageInput = val.replace(/[^0-9]/g, '');
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault();
      submitEdit();
    } else if (e.key === 'Escape') {
      e.preventDefault();
      cancelEdit();
    }
  }
</script>

<div class="flex flex-col sm:flex-row items-center justify-between gap-4 pt-6 mt-4 border-t border-border-subtle">
  <!-- Rows per page -->
  <div class="flex items-center gap-3 text-sm text-text-secondary">
    <span>Rows per page:</span>
    <select 
      class="bg-surface-default border border-border-strong rounded-lg px-2 py-1 text-text-primary focus:outline-none focus:ring-1 focus:ring-primary-default cursor-pointer"
      value={limit}
      onchange={handleLimitChange}
    >
      <option value={20}>20</option>
      <option value={40}>40</option>
    </select>
    <span class="ml-2">
      Showing {Math.min(offset + 1, total)}-{Math.min(offset + limit, total)} of {total}
    </span>
  </div>

  <!-- Navigation -->
  <div class="flex items-center gap-1">
    <button 
      class="btn-icon btn-ghost p-1.5 disabled:opacity-30"
      onclick={() => goToPage(1)}
      disabled={!canPrev}
      title="First Page"
    >
      <ChevronsLeft size={18} />
    </button>
    <button 
      class="btn-icon btn-ghost p-1.5 disabled:opacity-30"
      onclick={() => goToPage(currentPage - 1)}
      disabled={!canPrev}
      title="Previous"
    >
      <ChevronLeft size={18} />
    </button>
    
    {#if editing}
      <input
        type="text"
        inputmode="numeric"
        class="px-3 py-1.5 text-sm font-medium bg-surface-default border border-primary-default rounded-xl w-16 text-center focus:outline-none focus:ring-1 focus:ring-primary-default [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
        value={pageInput}
        oninput={handleInput}
        onkeydown={handleKeydown}
        onblur={submitEdit}
        autofocus
      />
      <span class="text-sm text-text-muted">/ {totalPages}</span>
    {:else}
      <button
        class="px-4 py-1.5 text-sm font-medium bg-surface-default border border-border-strong rounded-xl min-w-[100px] text-center hover:border-primary-default/50 transition-colors cursor-text"
        onclick={startEdit}
        title="Click to jump to page"
      >
        Page {currentPage} of {totalPages}
      </button>
    {/if}

    <button 
      class="btn-icon btn-ghost p-1.5 disabled:opacity-30"
      onclick={() => goToPage(currentPage + 1)}
      disabled={!canNext}
      title="Next"
    >
      <ChevronRight size={18} />
    </button>
    <button 
      class="btn-icon btn-ghost p-1.5 disabled:opacity-30"
      onclick={() => goToPage(totalPages)}
      disabled={!canNext}
      title="Last Page"
    >
      <ChevronsRight size={18} />
    </button>
  </div>
</div>
