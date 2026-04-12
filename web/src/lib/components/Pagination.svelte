<script>
  import { ChevronLeft, ChevronRight } from 'lucide-svelte';

  let { total = 0, limit = 10, offset = 0, onPageChange } = $props();

  let currentPage = $derived(Math.floor(offset / limit) + 1);
  let totalPages = $derived(Math.ceil(total / limit));

  function goToPage(page) {
    if (page < 1 || page > totalPages) return;
    onPageChange((page - 1) * limit);
  }

  function handleLimitChange(e) {
    const newLimit = parseInt(e.target.value);
    onPageChange(0, newLimit); // Reset to first page when limit changes
  }
</script>

<div class="pagination-container">
  <div class="limit-selector">
    <label for="limit-select">Tampilkan:</label>
    <select id="limit-select" value={limit} onchange={handleLimitChange}>
      <option value={10}>10</option>
      <option value={20}>20</option>
      <option value={40}>40</option>
    </select>
    <span class="total-text">Total {total} data</span>
  </div>

  <div class="pages">
    <button 
      class="nav-btn" 
      disabled={currentPage === 1} 
      onclick={() => goToPage(currentPage - 1)}
    >
      <ChevronLeft size={18} />
    </button>
    
    <span class="page-info">Halaman <strong>{currentPage}</strong> dari <strong>{totalPages || 1}</strong></span>

    <button 
      class="nav-btn" 
      disabled={currentPage === totalPages || totalPages === 0} 
      onclick={() => goToPage(currentPage + 1)}
    >
      <ChevronRight size={18} />
    </button>
  </div>
</div>

<style>
  .pagination-container {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px;
    background: var(--bg-surface);
    border-top: 1px solid var(--border);
    border-radius: 0 0 12px 12px;
  }

  .limit-selector {
    display: flex;
    align-items: center;
    gap: 12px;
    font-size: 0.875rem;
    color: var(--text-secondary);
  }

  select {
    background: var(--bg-main);
    border: 1px solid var(--border);
    color: white;
    padding: 4px 8px;
    border-radius: 6px;
    outline: none;
    cursor: pointer;
  }

  .pages {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .page-info {
    font-size: 0.875rem;
    color: var(--text-secondary);
  }

  .page-info strong {
    color: white;
  }

  .nav-btn {
    background: var(--bg-main);
    border: 1px solid var(--border);
    color: white;
    width: 36px;
    height: 36px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 8px;
    transition: all 0.2s;
  }

  .nav-btn:hover:not(:disabled) {
    background: var(--primary);
    border-color: var(--primary);
  }

  .nav-btn:disabled {
    opacity: 0.3;
    cursor: not-allowed;
  }
</style>
