<script>
  import { 
    BarChart3, 
    Download, 
    Calendar,
    ArrowUpRight,
    Search
  } from 'lucide-svelte';

  import { onMount, onDestroy, tick } from 'svelte';
  import { DateInput } from 'date-picker-svelte';
  import api from '$lib/api.js';
  import Pagination from '$lib/components/Pagination.svelte';
  import Chart from 'chart.js/auto';

  const DEBOUNCE_DELAY = 300;

  const indonesianLocale = {
    weekdays: ['Min', 'Sen', 'Sel', 'Rab', 'Kam', 'Jum', 'Sab'],
    months: ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'],
    weekStartsOn: 1
  };

  function dateToString(date) {
    if (!date) return '';
    const d = new Date(date);
    if (isNaN(d.getTime())) return '';
    return d.toISOString().split('T')[0];
  }

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
  let activeSearch = $state('');

  let chartCanvas = $state(null);
  let chartInstance = $state(null);
  let chartLoading = $state(true);
  let chartError = $state('');
  let dateRangeStart = $state(null);
  let dateRangeEnd = $state(null);
  let groupBy = $state('day');
  let chartInitialized = $state(false);

  const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];

  function formatIndonesianDate(dateStr) {
    if (!dateStr) return '';
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return '';
    const day = d.getDate();
    const month = monthNames[d.getMonth()];
    const year = d.getFullYear();
    const hours = d.getHours().toString().padStart(2, '0');
    const minutes = d.getMinutes().toString().padStart(2, '0');
    return `${day} ${month} ${year} ${hours}:${minutes}`;
  }

  const today = new Date();
  const defaultStart = new Date(today);
  defaultStart.setDate(today.getDate() - 7);
  dateRangeStart = defaultStart;
  dateRangeEnd = today;

  onMount(async () => {
    chartInitialized = true;
    await tick();
    await fetchChartData();
  });

  async function fetchChartData() {
    if (!dateRangeStart || !dateRangeEnd) return;
    chartLoading = true;
    chartError = '';
    try {
      const startStr = dateToString(dateRangeStart);
      const endStr = dateToString(dateRangeEnd);
      const url = `/sales/chart?start_date=${startStr}&end_date=${endStr}&group_by=${groupBy}`;
      console.log('Fetching chart from:', url);
      const resp = await api.get(url);
      console.log('Chart response status:', resp.status);
      console.log('Chart response data:', resp.data);
      const data = resp.data;
      
      chartLoading = false;
      
      // Check if we got valid data
      if (!data || !data.labels || data.labels.length === 0) {
        chartError = 'Tidak ada penjualan pada rentang tanggal ini';
        return;
      }
      
      await tick();
      renderChart(data.labels || [], data.values || []);
    } catch (e) {
      console.error('Failed to fetch chart data:', e);
      chartError = e.response?.data?.error || e.message;
      chartLoading = false;
    }
  }

  function renderChart(labels, values) {
    console.log('Rendering chart with labels:', labels, 'values:', values, 'canvas:', !!chartCanvas);
    
    if (!chartCanvas) {
      console.log('No chart canvas - exiting');
      return;
    }
    
    if (labels.length === 0) {
      chartError = 'Tidak ada penjualan pada rentang tanggal ini';
      return;
    }

    console.log('Creating chart...');
    
    if (chartInstance) {
      chartInstance.destroy();
    }

    const ctx = chartCanvas.getContext('2d');
    chartInstance = new Chart(ctx, {
      type: 'bar',
      data: {
        labels: labels,
        datasets: [{
          label: 'Total Penjualan (Rp)',
          data: values,
          backgroundColor: 'rgba(99, 102, 241, 0.6)',
          borderColor: 'rgba(99, 102, 241, 1)',
          borderWidth: 1,
          borderRadius: 4,
        }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: {
            display: false
          },
          tooltip: {
            callbacks: {
              label: function(context) {
                return 'Rp ' + context.raw.toLocaleString();
              }
            }
          }
        },
        scales: {
          y: {
            beginAtZero: true,
            ticks: {
              callback: function(value) {
                return 'Rp ' + (Number(value) / 1000).toLocaleString() + 'rb';
              }
            }
          },
          x: {
            ticks: {
              callback: function(value, index) {
                const label = labels[index];
                if (!label) return label;
                const d = new Date(label);
                if (isNaN(d.getTime())) return label;
                const day = d.getDate().toString().padStart(2, '0');
                const month = monthNames[d.getMonth()];
                const year = d.getFullYear();
                return `${day} ${month} ${year}`;
              },
              maxRotation: 45,
              minRotation: 45
            }
          }
        }
      }
    });
  }

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
    void activeSearch;
    void offset;
    void limit;
    void sortField;
    void sortDir;
    fetchTransactions();
  });

  $effect(() => {
    if (!chartInitialized || !dateRangeStart || !dateRangeEnd) return;
    void dateRangeStart;
    void dateRangeEnd;
    void groupBy;
    fetchChartData();
  });

  let searchDebounceTimer = null;

  function handleSearchInput() {
    if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
    const q = searchQuery.trim();
    if (q.length === 0) {
      activeSearch = '';
      offset = 0;
    } else if (q.length >= 3) {
      searchDebounceTimer = setTimeout(() => {
        activeSearch = q;
        offset = 0;
      }, DEBOUNCE_DELAY);
    }
  }

  onDestroy(() => {
    if (searchDebounceTimer) clearTimeout(searchDebounceTimer);
  });

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
          onkeydown={(e) => { if (e.key === 'Enter' && searchQuery.trim().length >= 3) { if (searchDebounceTimer) clearTimeout(searchDebounceTimer); activeSearch = searchQuery.trim(); offset = 0; } }}
        />
        {#if searchQuery.trim().length > 0 && searchQuery.trim().length < 3}
          <div class="search-warning">Minimal 3 karakter</div>
        {/if}
      </div>
    </div>
    <div class="filter-actions">
      <div class="filter-item date-filter">
        <Calendar size={18} />
        <div class="date-display">
          <DateInput 
            bind:value={dateRangeStart}
            format="dd MMM yyyy"
            locale={indonesianLocale}
            max={dateRangeEnd}
            placeholder="Pilih tanggal"
          />
        </div>
        <span class="separator">-</span>
        <div class="date-display">
          <DateInput 
            bind:value={dateRangeEnd}
            format="dd MMM yyyy"
            locale={indonesianLocale}
            min={dateRangeStart}
            placeholder="Pilih tanggal"
          />
        </div>
      </div>
      <div class="filter-item">
        <select bind:value={groupBy} class="group-select">
          <option value="day">Harian</option>
          <option value="week">Mingguan</option>
          <option value="month">Bulanan</option>
        </select>
      </div>
    </div>
  </div>

  <div class="chart-container premium-card glass">
    {#if chartLoading}
      <div class="chart-loading">
        <BarChart3 size={48} class="spin" />
        <p>Memuat data...</p>
      </div>
    {:else if chartError}
      <div class="chart-error">
        <p>Error: {chartError}</p>
        <button onclick={fetchChartData}>Coba Lagi</button>
      </div>
    {:else}
      <canvas bind:this={chartCanvas}></canvas>
    {/if}
  </div>

  <div class="table-container premium-card">
    <div class="table-wrapper">
      {#if loading}
        <div class="loading-state">
          <div class="loading-spinner"></div>
          <p>Memuat transaksi...</p>
        </div>
      {:else}
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
              <td>{formatIndonesianDate(tx.created_at)}</td>
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
      {/if}
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
        <p><span>Waktu:</span> {formatIndonesianDate(selectedTransaction.created_at)}</p>
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
    overflow: visible;
    z-index: 50;
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

  .date-filter {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .date-display {
    display: inline-block;
    position: relative;
  }

  :global(.date-input) {
    background: var(--bg-main);
    border: 1px solid var(--border);
    color: var(--text-primary);
    padding: 6px 10px;
    border-radius: 6px;
    font-size: 0.875rem;
    font-family: inherit;
    width: 100px;
  }

  :global(.date-input:focus) {
    border-color: var(--primary);
    outline: none;
  }

  :global(.picker) {
    z-index: 9999;
  }

  :global(.picker.visible) {
    display: block;
  }

  .date-filter .separator {
    color: var(--text-secondary);
  }

  .group-select {
    background: var(--bg-main);
    border: 1px solid var(--border);
    color: var(--text-primary);
    padding: 6px 10px;
    border-radius: 6px;
    font-size: 0.875rem;
    font-family: inherit;
    outline: none;
    cursor: pointer;
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

  .chart-container {
    height: 300px;
    padding: 16px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .chart-container canvas {
    width: 100% !important;
    height: 100% !important;
  }

  .chart-loading {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    color: var(--text-secondary);
  }

  .chart-loading :global(.spin) {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  .chart-error {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    color: #ef4444;
  }

  .chart-error button {
    background: var(--primary);
    color: white;
    border: none;
    padding: 8px 16px;
    border-radius: 6px;
    cursor: pointer;
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

  .loading-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 300px;
    color: var(--text-secondary);
    gap: 16px;
  }

  .loading-spinner {
    width: 40px;
    height: 40px;
    border: 3px solid rgba(99, 102, 241, 0.2);
    border-top-color: var(--primary);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
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
