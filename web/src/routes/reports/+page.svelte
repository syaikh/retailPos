<script>
  import { 
    BarChart3, 
    Download, 
    Calendar,
    Filter,
    ArrowUpRight,
    Search
  } from 'lucide-svelte';

  import { onMount, tick } from 'svelte';
  import api from '$lib/api.js';
  import Pagination from '$lib/components/Pagination.svelte';

  let transactions = $state([]);
  let loading = $state(true);
  let selectedTransaction = $state(null);
  let showDetailModal = $state(false);

  let limit = $state(10);
  let offset = $state(0);
  let total = $state(0);
  let searchQuery = $state('');
  let sortField = $state('created_at');
  let sortDir = $state('desc');
  // activeSearch is the committed search term (min 3 chars or empty)
  let activeSearch = $state('');

  onMount(async () => {
    // trigger initial load
  });

  async function fetchTransactions() {
    loading = true;
    try {
      const q = activeSearch.trim();
      const searchParam = q.length >= 3 ? `&search=${encodeURIComponent(q)}` : '';
      const url = `/sales?limit=${limit}&offset=${offset}${searchParam}&sortBy=${sortField}&sortDir=${sortDir}`;
      const resp = await api.get(url);
      const { data, total: totalCount } = resp.data;
      transactions = Array.isArray(data) ? data : [];
      total = totalCount ?? 0;
    } catch (e) {
      console.error('Failed to fetch transactions:', e);
      transactions = [];
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    // Svelte tracks: activeSearch, offset, limit, sortField, sortDir
    void activeSearch;
    void offset;
    void limit;
    void sortField;
    void sortDir;
    fetchTransactions();
  });

  let debounceTimer;
  function handleSearchInput() {
    clearTimeout(debounceTimer);
    const q = searchQuery.trim();
    if (q.length === 0) {
      // Immediately clear search
      activeSearch = '';
      offset = 0;
    } else if (q.length >= 3) {
      // Debounce to avoid firing on every keystroke
      debounceTimer = setTimeout(() => {
        activeSearch = q;
        offset = 0;
      }, 400);
    }
    // 1-2 chars: show warning, don't fetch
  }

  function handlePageChange(newOffset, newLimit) {
    if (newLimit !== undefined) limit = newLimit;
    offset = newOffset;
  }

  function handleSort(field) {
    if (sortField === field) {
      sortDir = sortDir === 'asc' ? 'desc' : 'asc';
    } else {
      sortField = field;
      sortDir = 'asc';
    }
    offset = 0;
  }

  function openDetail(tx) {
    selectedTransaction = tx;
    showDetailModal = true;
  }
</script>

<div class="reports">
  <div class="header">
    <div class="title">
      <BarChart3 size={32} color="var(--primary)" />
      <h1>Laporan Penjualan</h1>
    </div>
    <button class="export-btn">
      <Download size={18} />
      Ekspor PDF/Excel
    </button>
  </div>

  <div class="filter-bar premium-card glass">
    <div class="search-section">
      <div class="search-wrapper">
        <span class="icon"><Search size={18} /></span>
        <input
          type="text"
          placeholder="Cari ID TRX atau Nama Produk..."
          bind:value={searchQuery}
          oninput={handleSearchInput}
          onkeydown={(e) => { if (e.key === 'Enter' && searchQuery.trim().length >= 3) { clearTimeout(debounceTimer); activeSearch = searchQuery.trim(); offset = 0; } }}
        />
        {#if searchQuery.trim().length > 0 && searchQuery.trim().length < 3}
          <div class="search-warning">Minimal 3 karakter</div>
        {/if}
      </div>
    </div>
    <div class="filter-actions">
      <div class="filter-item">
        <Calendar size={18} />
        <span>7 Hari Terakhir</span>
      </div>
      <div class="filter-item">
        <Filter size={18} />
        <span>Semua Kasir</span>
      </div>
    </div>
  </div>

  <div class="chart-placeholder premium-card glass">
    <div class="placeholder-content">
      <BarChart3 size={64} opacity={0.2} />
      <p>Visualisasi Grafik Penjualan</p>
      <span>Pilih rentang tanggal untuk melihat statistik detail</span>
    </div>
  </div>

  <div class="table-container premium-card">
    <div class="table-wrapper">
      <table>
        <thead>
          <tr>
            <th onclick={() => handleSort('id')} class="sortable">
              ID {#if sortField === 'id'}<span class="sort-icon">{sortDir === 'asc' ? '▲' : '▼'}</span>{/if}
            </th>
            <th onclick={() => handleSort('created_at')} class="sortable">
              Waktu {#if sortField === 'created_at'}<span class="sort-icon">{sortDir === 'asc' ? '▲' : '▼'}</span>{/if}
            </th>
            <th onclick={() => handleSort('total')} class="sortable">
              Total {#if sortField === 'total'}<span class="sort-icon">{sortDir === 'asc' ? '▲' : '▼'}</span>{/if}
            </th>
            <th>Metode</th>
            <th>Qty Item</th>
            <th>Aksi</th>
          </tr>
        </thead>
        <tbody>
          {#each transactions as tx}
            <tr>
              <td><code>#TRX-{tx.id.toString().padStart(4, '0')}</code></td>
              <td>{new Date(tx.created_at).toLocaleString()}</td>
              <td><strong>Rp {tx.total_amount.toLocaleString()}</strong></td>
              <td><span class="method-badge">{tx.payment_method}</span></td>
              <td>{tx.items?.reduce((sum, i) => sum + i.quantity, 0) || 0} unit</td>
              <td>
                <button class="detail-link" onclick={() => openDetail(tx)}>
                  Detail <ArrowUpRight size={14} />
                </button>
              </td>
            </tr>
          {/each}
          {#if !loading && transactions.length === 0}
            <tr>
              <td colspan="6" class="empty">Belum ada transaksi terekam</td>
            </tr>
          {/if}
        </tbody>
      </table>
    </div>
    <Pagination 
      total={total} 
      limit={limit} 
      offset={offset} 
      onPageChange={handlePageChange} 
    />
  </div>
</div>

{#if showDetailModal && selectedTransaction}
  <div 
    class="modal-overlay" 
    role="button" 
    tabindex="-1" 
    onclick={() => showDetailModal = false}
    onkeydown={(e) => e.key === 'Enter' && (showDetailModal = false)}
  >
    <div class="modal premium-card" role="presentation" onclick={(e) => e.stopPropagation()}>
      <div class="modal-header">
        <h2>Detail Transaksi #TRX-{selectedTransaction.id.toString().padStart(4, '0')}</h2>
        <button class="close-btn" onclick={() => showDetailModal = false}>&times;</button>
      </div>
      
      <div class="tx-meta">
        <p><span>Waktu:</span> {new Date(selectedTransaction.created_at).toLocaleString()}</p>
        <p><span>Metode Pembayaran:</span> {selectedTransaction.payment_method}</p>
      </div>

      <div class="items-table">
        <table>
          <thead>
            <tr>
              <th>Nama Produk (Snapshot)</th>
              <th>Harga Satuan</th>
              <th>Qty</th>
              <th>Subtotal</th>
            </tr>
          </thead>
          <tbody>
            {#each selectedTransaction.items as item}
              <tr>
                <td>{item.product_name}</td>
                <td>Rp {item.price_at_sale.toLocaleString()}</td>
                <td>{item.quantity}</td>
                <td>Rp {(item.price_at_sale * item.quantity).toLocaleString()}</td>
              </tr>
            {/each}
          </tbody>
          <tfoot>
            <tr>
              <td colspan="3" class="total-label">Total Gede</td>
              <td class="total-val">Rp {selectedTransaction.total_amount.toLocaleString()}</td>
            </tr>
          </tfoot>
        </table>
      </div>
    </div>
  </div>
{/if}

<style>
  .table-container {
    height: 600px;
    display: flex;
    flex-direction: column;
    padding: 0;
    overflow: hidden;
  }

  .table-wrapper {
    flex: 1;
    overflow-y: auto;
  }

  .table-wrapper table {
    width: 100%;
    border-collapse: separate;
    border-spacing: 0;
  }

  .table-wrapper th {
    position: sticky;
    top: 0;
    background: var(--bg-surface);
    z-index: 10;
  }

  .reports {
    display: flex;
    flex-direction: column;
    gap: 24px;
    padding-bottom: 40px;
  }

  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .title {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .export-btn {
    background: var(--bg-surface);
    border: 1px solid var(--border);
    color: var(--text-primary);
    padding: 10px 20px;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .filter-bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 24px;
    gap: 20px;
    flex-wrap: wrap;
  }

  .search-section {
    flex: 1;
    max-width: 400px;
  }

  .search-wrapper {
    position: relative;
    width: 100%;
  }

  .search-wrapper .icon {
    position: absolute;
    left: 12px;
    top: 50%;
    transform: translateY(-50%);
    color: var(--text-secondary);
  }

  .search-wrapper input {
    width: 100%;
    padding: 10px 10px 10px 40px;
    background: #0f172a;
    border: 1px solid var(--border);
    border-radius: 8px;
    color: white;
    font-size: 1rem;
    font-family: inherit;
    outline: none;
    transition: border-color 0.2s;
  }

  .search-wrapper input:focus {
    border-color: var(--primary);
  }

  .search-warning {
    position: absolute;
    top: calc(100% + 12px);
    left: 40px;
    font-size: 0.75rem;
    color: #000;
    background: #f59e0b;
    padding: 6px 12px;
    border-radius: 6px;
    pointer-events: none;
    z-index: 100;
    white-space: nowrap;
    font-weight: 700;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
    animation: slideDown 0.2s ease-out;
  }

  .search-warning::before {
    content: '';
    position: absolute;
    top: -6px;
    left: 12px;
    border-left: 6px solid transparent;
    border-right: 6px solid transparent;
    border-bottom: 6px solid #f59e0b;
  }

  @keyframes slideDown {
    from { opacity: 0; transform: translateY(-8px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .filter-actions {
    display: flex;
    gap: 24px;
  }

  .filter-item {
    display: flex;
    align-items: center;
    gap: 10px;
    color: var(--text-secondary);
    font-size: 0.875rem;
    cursor: pointer;
  }

  .filter-item:hover {
    color: var(--primary);
  }

  th.sortable {
    cursor: pointer;
    user-select: none;
    transition: color 0.2s;
  }
  
  th.sortable:hover {
    color: white;
  }
  
  .sort-icon {
    font-size: 0.8em;
    margin-left: 4px;
  }

  .chart-placeholder {
    height: 300px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: 2px dashed var(--border);
  }

  .placeholder-content {
    text-align: center;
    color: var(--text-secondary);
  }

  .placeholder-content p {
    font-size: 1.125rem;
    font-weight: 600;
    margin: 16px 0 4px;
  }

  .placeholder-content span {
    font-size: 0.875rem;
  }

  .detail-link {
    background: transparent;
    color: var(--primary);
    font-size: 0.875rem;
    display: flex;
    align-items: center;
    gap: 4px;
    cursor: pointer;
  }

  .method-badge {
    padding: 2px 8px;
    background: rgba(99, 102, 241, 0.1);
    color: var(--primary);
    border-radius: 4px;
    font-size: 0.8rem;
    font-weight: 600;
  }

  code {
    background: var(--bg-main);
    padding: 2px 6px;
    border-radius: 4px;
    color: var(--accent);
  }

  .empty {
    text-align: center;
    padding: 40px;
    color: var(--text-secondary);
  }

  /* Modal Styles */
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    backdrop-filter: blur(4px);
  }

  .modal {
    width: 90%;
    max-width: 700px;
    padding: 24px;
    max-height: 85vh;
    overflow-y: auto;
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
    border-bottom: 1px solid var(--border);
    padding-bottom: 12px;
  }

  .close-btn {
    background: transparent;
    font-size: 2rem;
    line-height: 1;
    color: var(--text-secondary);
    cursor: pointer;
  }

  .tx-meta {
    margin-bottom: 20px;
    font-size: 0.9rem;
    color: var(--text-secondary);
  }

  .tx-meta span {
    font-weight: 600;
    color: var(--text-primary);
    margin-right: 8px;
  }

  .items-table table {
    width: 100%;
    border-collapse: collapse;
  }

  .items-table th, .items-table td {
    text-align: left;
    padding: 12px;
    border-bottom: 1px solid var(--border);
  }

  .total-label {
    text-align: right;
    font-weight: 700;
    padding-top: 16px;
  }

  .total-val {
    font-size: 1.2rem;
    font-weight: 800;
    color: var(--primary);
    padding-top: 16px;
  }
</style>
