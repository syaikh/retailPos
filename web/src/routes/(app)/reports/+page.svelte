<script lang="ts">
  import { 
    BarChart3, 
    Download, 
    Calendar,
    ArrowUpRight,
    Search,
    ChevronDown
  } from 'lucide-svelte';

  import { onMount, onDestroy, tick } from 'svelte';
  import api from '$lib/api.js';
  import Pagination from '$lib/components/Pagination.svelte';
  import DatePicker from '$lib/components/DatePicker.svelte';

  interface TransactionItem {
    id: number;
    product_name: string;
    price_at_sale: number;
    quantity: number;
  }

  interface Transaction {
    id: number;
    created_at: string;
    total_amount: number;
    payment_method: string;
    items: TransactionItem[];
  }

  const DEBOUNCE_DELAY = 300;

  const indonesianLocale = {
    weekdays: ['Min', 'Sen', 'Sel', 'Rab', 'Kam', 'Jum', 'Sab'],
    months: ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'],
    weekStartsOn: 1
  };

  function dateToString(date: Date | null) {
    if (!date) return '';
    // Use Asia/Jakarta timezone to match backend database timezone
    const formatter = new Intl.DateTimeFormat('en-CA', {
      timeZone: 'Asia/Jakarta',
      year: 'numeric',
      month: '2-digit',
      day: '2-digit'
    });
    return formatter.format(date);
  }

  let transactions = $state<Transaction[]>([]);
  let loading = $state(true);
  let selectedTransaction = $state<Transaction | null>(null);
  let showDetailModal = $state(false);

  let limit = $state(10);
  let offset = $state(0);
  let total = $state(0);
  let searchQuery = $state('');
  let sortField = $state('created_at');
  let sortDir = $state('desc');
  let activeSearch = $state('');

  let chartCanvas = $state<HTMLCanvasElement | null>(null);
  let chartInstance = $state<any>(null);
  let chartLoading = $state(true);
  let chartError = $state('');
  let dateRangeStart = $state<Date | null>(null);
  let dateRangeEnd = $state<Date | null>(null);
  let groupBy = $state('day');
  let chartInitialized = $state(false);
  let exporting = $state(false);
  let showExportMenu = $state(false);

  $effect(() => {
    if (showExportMenu) {
      function handleClickOutside(e: MouseEvent) {
        const dropdown = document.querySelector('.export-dropdown');
        if (dropdown && !dropdown.contains(e.target as Node)) {
          showExportMenu = false;
        }
      }
      window.addEventListener('click', handleClickOutside);
      return () => window.removeEventListener('click', handleClickOutside);
    }
  });

  const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];

  function formatIndonesianDate(dateStr: string) {
    if (!dateStr) return '';
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return '';
    // Use Asia/Jakarta timezone to match backend
    const formatter = new Intl.DateTimeFormat('en-CA', {
      timeZone: 'Asia/Jakarta',
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      hour12: false
    });
    const parts = formatter.formatToParts(d);
    const year = parts.find(p => p.type === 'year')?.value || '';
    const month = parts.find(p => p.type === 'month')?.value || '';
    const day = parts.find(p => p.type === 'day')?.value || '';
    const hour = parts.find(p => p.type === 'hour')?.value || '';
    const minute = parts.find(p => p.type === 'minute')?.value || '';
    
    // Convert month number to name
    const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des'];
    const monthIndex = parseInt(month) - 1;
    const monthName = monthNames[monthIndex] || month;
    
    return `${day} ${monthName} ${year} ${hour}:${minute}`;
  }

  function getPaymentMethodLabel(method: string) {
    if (method === 'cash') return 'Tunai';
    if (method === 'card') return 'Kartu';
    return method;
  }

  // Get today's date in Asia/Jakarta timezone
  function getTodayInJakarta() {
    const now = new Date();
    const formatter = new Intl.DateTimeFormat('en-CA', {
      timeZone: 'Asia/Jakarta',
      year: 'numeric',
      month: '2-digit',
      day: '2-digit'
    });
    const dateStr = formatter.format(now);
    return new Date(dateStr + 'T00:00:00');
  }
  
  const today = getTodayInJakarta();
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
      await renderChart(data.labels || [], data.values || []);
    } catch (e: any) {
      console.error('Failed to fetch chart data:', e);
      chartError = e.response?.data?.error || e.message;
      chartLoading = false;
    }
  }

  function handleEndDateChange(newEndDate: Date | null) {
    if (!newEndDate) {
      dateRangeEnd = newEndDate;
      return;
    }
    
    // Validate that endDate is not less than startDate
    if (dateRangeStart && newEndDate < dateRangeStart) {
      // If endDate is less than startDate, set it to startDate
      dateRangeEnd = new Date(dateRangeStart);
    } else {
      dateRangeEnd = newEndDate;
    }
    
    fetchChartData();
  }

  function handleStartDateChange(newStartDate: Date | null) {
    if (!newStartDate) {
      dateRangeStart = newStartDate;
      return;
    }
    
    // Validate that startDate is not greater than endDate
    if (dateRangeEnd && newStartDate > dateRangeEnd) {
      // If startDate is greater than endDate, adjust endDate
      dateRangeEnd = new Date(newStartDate);
    }
    
    dateRangeStart = newStartDate;
    fetchChartData();
  }

  function formatChartLabel(label: string, groupByValue: string) {
    if (!label) return '';
    
    // Weekly format: "2024-15" (ISO week "YYYY-IW")
    if (groupByValue === 'week') {
      const parts = label.split('-');
      if (parts.length === 2) {
        const year = parts[0];
        const week = parts[1];
        return `M${week}, ${year}`;
      }
      return label;
    }
    
    // Monthly format: "2024-03"
    if (groupByValue === 'month') {
      const parts = label.split('-');
      if (parts.length === 2) {
        const year = parts[0];
        const monthNum = parseInt(parts[1]);
        if (monthNum >= 1 && monthNum <= 12) {
          const month = monthNames[monthNum - 1];
          return `${month} ${year}`;
        }
      }
      return label;
    }
    
    // Daily format: "2024-03-15"
    const d = new Date(label);
    if (isNaN(d.getTime())) return label;
    const day = d.getDate().toString().padStart(2, '0');
    const month = monthNames[d.getMonth()];
    const year = d.getFullYear();
    return `${day} ${month} ${year}`;
  }

  function getWeekDateRange(year: number, week: number) {
    const simple = new Date(year, 0, 1 + (week - 1) * 7);
    const dow = simple.getDay();
    const isoWeekStart = simple;
    if (dow <= 4) {
      isoWeekStart.setDate(simple.getDate() - simple.getDay() + 1);
    } else {
      isoWeekStart.setDate(simple.getDate() + 8 - simple.getDay());
    }
    const isoWeekEnd = new Date(isoWeekStart);
    isoWeekEnd.setDate(isoWeekStart.getDate() + 6);
    return { start: isoWeekStart, end: isoWeekEnd };
  }

  function formatTooltipTitle(label: string, groupByValue: string, startDate: Date | null, endDate: Date | null) {
    if (!label) return '';
    
    // WEEKLY
    if (groupByValue === 'week') {
      const parts = label.split('-');
      if (parts.length === 2) {
        const year = parseInt(parts[0]);
        const week = parseInt(parts[1]);
        const { start, end } = getWeekDateRange(year, week);
        
        const rangeStart = startDate ? new Date(startDate) : start;
        const rangeEnd = endDate ? new Date(endDate) : end;
        
        const effectiveStart = start < rangeStart ? rangeStart : start;
        const effectiveEnd = end > rangeEnd ? rangeEnd : end;
        
        const startDay = effectiveStart.getDate();
        const endDay = effectiveEnd.getDate();
        const startMonth = monthNames[effectiveStart.getMonth()];
        const endMonth = monthNames[effectiveEnd.getMonth()];
        
        if (startMonth === endMonth) {
          return `Minggu ke-${week}, ${year} (${startDay}–${endDay} ${startMonth})`;
        } else {
          return `Minggu ke-${week}, ${year} (${startDay} ${startMonth} – ${endDay} ${endMonth})`;
        }
      }
    }
    
    // MONTHLY
    if (groupByValue === 'month') {
      const parts = label.split('-');
      if (parts.length === 2) {
        const year = parseInt(parts[0]);
        const monthNum = parseInt(parts[1]);
        const monthName = monthNames[monthNum - 1];
        
        const rangeStart = startDate ? new Date(startDate) : null;
        const rangeEnd = endDate ? new Date(endDate) : null;
        
        // Jika ini bulan pertama dalam range
        if (rangeStart && monthNum === rangeStart.getMonth() + 1 && year === rangeStart.getFullYear()) {
          const lastDay = new Date(year, monthNum, 0).getDate();
          return `${monthName} ${year} (${rangeStart.getDate()}-${lastDay})`;
        }
        
        // Jika ini bulan terakhir dalam range
        if (rangeEnd && monthNum === rangeEnd.getMonth() + 1 && year === rangeEnd.getFullYear()) {
          const lastDay = rangeEnd.getDate();
          return `${monthName} ${year} (1-${lastDay})`;
        }
        
        // Bulan penuh - tetap tambahkan rentang (1-31)
        const lastDay = new Date(year, monthNum, 0).getDate();
        return `${monthName} ${year} (1-${lastDay})`;
      }
    }
    
    return formatChartLabel(label, groupByValue);
  }

  async function renderChart(labels: string[], values: number[]) {
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
    
    const Chart = (await import('chart.js/auto')).default;
    
    // Check if there's already a chart on this canvas in Chart.js registry and destroy it
    const existingChart = Chart.getChart(chartCanvas);
    if (existingChart) {
      console.log('Destroying existing chart from registry');
      existingChart.destroy();
    }
    
    // Also destroy our tracked instance
    if (chartInstance) {
      chartInstance.destroy();
      chartInstance = null;
    }
    
    // Clear canvas
    const ctx = chartCanvas.getContext('2d');
    if (!ctx) return;
    ctx.clearRect(0, 0, chartCanvas.width || 300, chartCanvas.height || 200);
    
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
              title: function(context: any) {
                const label = context[0].label;
                return formatTooltipTitle(label, groupBy, dateRangeStart, dateRangeEnd);
              },
              label: function(context: any) {
                return 'Rp ' + context.raw.toLocaleString();
              }
            }
          }
        },
        scales: {
          y: {
            beginAtZero: true,
            ticks: {
              callback: function(value: string | number) {
                return 'Rp ' + (Number(value) / 1000).toLocaleString() + 'rb';
              }
            }
          },
          x: {
            ticks: {
              callback: function(value: any, index: number) {
                const label = labels[index];
                return formatChartLabel(label, groupBy);
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
    } catch (e: any) {
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

  let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null;

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
    // Properly destroy chart on component unmount
    if (chartInstance) {
      chartInstance.destroy();
      chartInstance = null;
    }
  });

  function handlePageChange(newOffset: number, newLimit?: number) {
    if (newLimit !== undefined) limit = newLimit;
    offset = newOffset;
  }

  function handleSort(field: string) {
    if (sortField === field) {
      sortDir = sortDir === 'asc' ? 'desc' : 'asc';
    } else {
      sortField = field;
      sortDir = 'asc';
    }
    offset = 0;
  }

  function openDetail(tx: Transaction) {
    selectedTransaction = tx;
    showDetailModal = true;
  }

  async function fetchAllTransactionsForExport() {
    const startStr = dateToString(dateRangeStart);
    const endStr = dateToString(dateRangeEnd);
    const searchParam = activeSearch.trim().length >= 3 ? `&search=${encodeURIComponent(activeSearch.trim())}` : '';
    const url = `/sales?limit=10000&offset=0&sortBy=${sortField}&sortDir=${sortDir}&start_date=${startStr}&end_date=${endStr}${searchParam}`;
    const resp = await api.get(url);
    return Array.isArray(resp.data.data) ? resp.data.data : [];
  }

  async function exportPDF() {
    exporting = true;
    showExportMenu = false;
    try {
      const [jsPDFModule, autoTableModule] = await Promise.all([
        import('jspdf'),
        import('jspdf-autotable')
      ]);
      const jsPDF = jsPDFModule.default;
      const autoTable = autoTableModule.default;
      
      const allTransactions = await fetchAllTransactionsForExport();
      
      const doc = new jsPDF();
      const pageWidth = doc.internal.pageSize.getWidth();
      
      doc.setFontSize(16);
      doc.text('Laporan Penjualan', pageWidth / 2, 15, { align: 'center' });
      
      doc.setFontSize(10);
      const dateRangeText = `${dateToString(dateRangeStart)} - ${dateToString(dateRangeEnd)}`;
      doc.text(dateRangeText, pageWidth / 2, 22, { align: 'center' });
      
      let yPos = 30;
      
      if (chartCanvas && chartInstance) {
        try {
          const chartImage = chartCanvas.toDataURL('image/png', 1.0);
          const imgWidth = pageWidth - 20;
          const imgHeight = 80;
          doc.addImage(chartImage, 'PNG', 10, yPos, imgWidth, imgHeight);
          yPos += imgHeight + 10;
        } catch (e) {
          console.warn('Could not capture chart:', e);
        }
      }
      
      const tableData = allTransactions.map((tx: Transaction) => [
        `#TRX-${tx.id.toString().padStart(4, '0')}`,
        formatIndonesianDate(tx.created_at),
        `Rp ${tx.total_amount.toLocaleString()}`,
        getPaymentMethodLabel(tx.payment_method),
        `${tx.items?.reduce((sum: number, i: TransactionItem) => sum + i.quantity, 0) || 0} unit`
      ]);
      
      autoTable(doc, {
        head: [['ID', 'Waktu', 'Total', 'Metode', 'Qty Item']],
        body: tableData,
        startY: yPos,
        styles: { fontSize: 8 },
        headStyles: { fillColor: [99, 102, 241] }
      });
      
      doc.save(`laporan-penjualan-${dateToString(dateRangeStart)}-${dateToString(dateRangeEnd)}.pdf`);
    } catch (e: any) {
      console.error('Export PDF failed:', e);
      alert('Gagal mengekspor PDF: ' + e.message);
    } finally {
      exporting = false;
    }
  }

  async function exportExcel() {
    exporting = true;
    showExportMenu = false;
    try {
      const XLSX = await import('xlsx');
      
      const allTransactions = await fetchAllTransactionsForExport();
      
       const data = allTransactions.map((tx: Transaction) => ({
         ID: `#TRX-${tx.id.toString().padStart(4, '0')}`,
         Waktu: formatIndonesianDate(tx.created_at),
         Total: tx.total_amount,
         Metode: getPaymentMethodLabel(tx.payment_method),
         'Qty Item': tx.items?.reduce((sum: number, i: TransactionItem) => sum + i.quantity, 0) || 0
       }));
      
      const ws = XLSX.utils.json_to_sheet(data);
      const wb = XLSX.utils.book_new();
      XLSX.utils.book_append_sheet(wb, ws, 'Transaksi');
      
      const colWidths = [
        { wch: 12 },
        { wch: 18 },
        { wch: 12 },
        { wch: 10 },
        { wch: 10 }
      ];
      ws['!cols'] = colWidths;
      
      XLSX.writeFile(wb, `laporan-penjualan-${dateToString(dateRangeStart)}-${dateToString(dateRangeEnd)}.xlsx`);
    } catch (e: any) {
      console.error('Export Excel failed:', e);
      alert('Gagal mengekspor Excel: ' + e.message);
    } finally {
      exporting = false;
    }
  }

  function handleExportClick() {
    showExportMenu = !showExportMenu;
  }

  function closeExportMenu() {
    showExportMenu = false;
  }
</script>

<div class="reports">
  <div class="header">
    <div class="title">
      <BarChart3 size={32} color="var(--primary)" />
      <h1>Laporan Penjualan</h1>
    </div>
    <div class="export-dropdown">
      <button class="export-btn" onclick={handleExportClick} disabled={exporting}>
        <Download size={18} />
        {exporting ? 'Mengekspor...' : 'Ekspor'}
        <ChevronDown size={16} />
      </button>
      {#if showExportMenu}
        <div class="export-menu">
          <button onclick={exportPDF} disabled={exporting}>
            <Download size={16} /> Ekspor PDF
          </button>
          <button onclick={exportExcel} disabled={exporting}>
            <Download size={16} /> Ekspor Excel
          </button>
        </div>
      {/if}
    </div>
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
          onkeydown={(e: KeyboardEvent) => { if (e.key === 'Enter' && searchQuery.trim().length >= 3) { if (searchDebounceTimer) clearTimeout(searchDebounceTimer); activeSearch = searchQuery.trim(); offset = 0; } }}
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
          <DatePicker 
            bind:value={dateRangeStart}
            format="dd MMM yyyy"
            locale={indonesianLocale}
            max={today}
            placeholder="Pilih tanggal"
            onchange={handleStartDateChange}
          />
        </div>
        <span class="separator">-</span>
        <div class="date-display">
          <DatePicker 
            bind:value={dateRangeEnd}
            format="dd MMM yyyy"
            locale={indonesianLocale}
            max={today}
            placeholder="Pilih tanggal"
            onchange={handleEndDateChange}
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
               <td><span class="method-badge">{getPaymentMethodLabel(tx.payment_method)}</span></td>
              <td>{tx.items?.reduce((sum: number, i: TransactionItem) => sum + i.quantity, 0) || 0} unit</td>
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
         <p><span>Metode Pembayaran:</span> {getPaymentMethodLabel(selectedTransaction.payment_method)}</p>
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
    z-index: 100;
  }

  .title {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .export-btn {
    background: var(--success);
    color: white;
    padding: 8px 12px;
    border-radius: 6px;
    border: none;
    display: flex;
    align-items: center;
    gap: 4px;
    cursor: pointer;
  }

  .export-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .export-dropdown {
    position: relative;
    z-index: 100;
  }

  .export-menu {
    position: absolute;
    top: 100%;
    right: 0;
    margin-top: 4px;
    background: #1e293b;
    border: 1px solid var(--border);
    border-radius: 6px;
    overflow: hidden;
    z-index: 101;
    min-width: 160px;
  }

  .export-menu button {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 8px 16px;
    background: transparent;
    border: none;
    color: white;
    text-align: left;
    cursor: pointer;
    font-size: 0.875rem;
    white-space: nowrap;
  }

  .export-menu button:hover {
    background: rgba(99, 102, 241, 0.1);
  }

  .export-menu button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
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
