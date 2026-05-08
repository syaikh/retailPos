<script>
  import { onMount } from 'svelte';
  import { fly } from 'svelte/transition';
  import { apiFetch } from '$lib/api/client';
  import { toast } from '$lib/stores/toast';
  import { chart } from '$lib/actions/chart';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import Pagination from '$lib/components/ui/Pagination.svelte';
  import Modal from '$lib/components/ui/Modal.svelte';
  import {
    Receipt, BarChart3,
    CalendarDays, Download, FileSpreadsheet,
    ChevronDown, Eye,
  } from 'lucide-svelte';

  let loading = $state(true);
  let salesData = $state([]);
  let chartData = $state([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);

  // KPI data
  let kpiData = $state({
    totalRevenue: 0,
    previousRevenue: 0,
    totalOrders: 0,
    previousOrders: 0,
    avgOrderValue: 0,
    previousAvgOrderValue: 0,
    revenuePerDay: 0,
    previousRevenuePerDay: 0,
    percentChange: 0,
    comparisonType: 'zero',
    isPartial: false,
    periodInfo: null
  });

  // Date range - default to last 7 days inclusive of today
  const now = new Date();
  let startDate = $state(new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000).toISOString().slice(0, 10));
  let endDate = $state(now.toISOString().slice(0, 10));

  // Export dropdown
  let showExportDropdown = $state(false);

  // Tab state for granularity switching
  let activeTab = $state('daily'); // 'daily', 'weekly', 'monthly'

  // Transaction details modal
  let showTransactionModal = $state(false);
  let selectedTransaction = $state(null);

  // Format date: dd mmm yyyy (English locale)
  const formatDate = (dateString) => {
    if (!dateString) return '';
    const date = new Date(dateString);
    const day = date.getDate().toString().padStart(2, '0');
    const month = date.toLocaleString('en-US', { month: 'short' });
    const year = date.getFullYear();
    return `${day} ${month} ${year}`;
  };

  // Format date and time: dd mmm yyyy hh:mm:ss (English locale)
  const formatDateTime = (date) => {
    const day = date.getDate().toString().padStart(2, '0');
    const month = date.toLocaleString('en-US', { month: 'short' });
    const year = date.getFullYear();
    const hours = date.getHours().toString().padStart(2, '0');
    const minutes = date.getMinutes().toString().padStart(2, '0');
    const seconds = date.getSeconds().toString().padStart(2, '0');

    return `${day} ${month} ${year} ${hours}:${minutes}:${seconds}`;
  };

  // Reactive tooltips
  const startDateTooltip = $derived(formatDate(startDate));
  const endDateTooltip = $derived(formatDate(endDate));

  // Chart configuration
  const chartConfig = $derived.by(() => {
    let labels = [];
    let values = [];
    let chartType = 'line';

    if (activeTab === 'daily') {
      chartType = 'line';
      labels = chartData.map(d => {
        const date = new Date(d.date);
        return date.toLocaleString('en-US', { month: 'short', day: 'numeric' });
      });
      values = chartData.map(d => d.total);
    } else if (activeTab === 'weekly') {
      chartType = 'bar';
      labels = chartData.map(d => {
        const start = new Date(d.week_start);
        const end = new Date(d.week_end);
        const startStr = start.toLocaleString('en-US', { month: 'short', day: 'numeric' });
        const endStr = end.toLocaleString('en-US', { month: 'short', day: 'numeric' });
        return `${startStr} - ${endStr}`;
      });
      values = chartData.map(d => d.total);
    } else if (activeTab === 'monthly') {
      chartType = 'bar';
      labels = chartData.map(d => {
        const date = new Date(d.month_start);
        return date.toLocaleString('en-US', { month: 'short', year: '2-digit' });
      });
      values = chartData.map(d => d.total);
    }

    return {
      type: chartType,
      data: {
        labels,
        datasets: [{
          label: 'Revenue',
          data: values,
          borderColor: '#7c3aed', // Primary
          backgroundColor: chartType === 'bar' ? '#7c3aed' : 'rgba(124, 58, 237, 0.1)',
          borderWidth: chartType === 'bar' ? 0 : 2,
          pointBackgroundColor: '#fff',
          pointBorderColor: '#7c3aed',
          pointBorderWidth: 2,
          pointRadius: chartType === 'bar' ? 0 : 4,
          pointHoverRadius: chartType === 'bar' ? 0 : 6,
          fill: chartType === 'bar' ? true : true,
          tension: chartType === 'bar' ? 0 : 0.4
        }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: { display: false },
          tooltip: {
            callbacks: {
              label: function(context) {
                let label = context.dataset.label || '';
                if (label) label += ': ';
                if (context.parsed.y !== null) {
                  label += 'Rp ' + context.parsed.y.toLocaleString('id-ID');
                }
                return label;
              }
            }
          }
        },
        scales: {
          x: {
            grid: { display: false },
            ticks: { color: '#9ca3af', font: { family: 'inherit' } }
          },
          y: {
            border: { display: false },
            grid: { color: 'rgba(255, 255, 255, 0.05)' },
            ticks: {
              color: '#9ca3af',
              font: { family: 'inherit' },
              callback: function(value) {
                if (value >= 1000000) return 'Rp ' + (value / 1000000).toFixed(1) + 'M';
                if (value >= 1000) return 'Rp ' + (value / 1000).toFixed(0) + 'k';
                return 'Rp ' + value;
              }
            }
          }
        }
      }
    };
  });



  async function fetchSales() {
    try {
      loading = true;
      const params = new URLSearchParams({
        startDate,
        endDate,
        limit: limit.toString(),
        offset: offset.toString()
      });

      // Choose chart endpoint based on active tab
      const chartEndpoint = activeTab === 'weekly' ? '/api/dashboard/chart/weekly' :
                           activeTab === 'monthly' ? '/api/dashboard/chart/monthly' :
                           '/api/dashboard/chart';

      const [salesRes, chartRes, comparisonRes] = await Promise.all([
        apiFetch(`/api/sales?${params.toString()}`),
        apiFetch(`${chartEndpoint}?startDate=${startDate}&endDate=${endDate}`),
        apiFetch(`/api/dashboard/comparison?period=${activeTab}&mode=todate&date=${endDate}`)
      ]);

      if (salesRes.ok) {
        const data = await salesRes.json();
        salesData = data.data || [];
        total = data.total || 0;
      }

      if (chartRes.ok) {
        const cData = await chartRes.json();
        chartData = cData.data || [];
      }

      // Enhanced KPI calculation with comparison data
      if (comparisonRes.ok) {
        const compData = await comparisonRes.json();
        const comparison = compData.data;
        const meta = compData.meta;

        // Calculate percent change with type detection
        let percentChange = 0;
        let comparisonType = 'zero';

        if (comparison.previous_revenue === 0 && comparison.current_revenue > 0) {
          comparisonType = 'new';
          percentChange = Infinity;
        } else if (comparison.previous_revenue === 0 && comparison.current_revenue === 0) {
          comparisonType = 'zero';
          percentChange = 0;
        } else if (comparison.previous_revenue > 0) {
          comparisonType = 'normal';
          percentChange = ((comparison.current_revenue - comparison.previous_revenue) / comparison.previous_revenue) * 100;
        }

        kpiData = {
          totalRevenue: comparison.current_revenue,
          previousRevenue: comparison.previous_revenue,
          totalOrders: comparison.current_orders,
          previousOrders: comparison.previous_orders,
          avgOrderValue: comparison.current_aov,
          previousAvgOrderValue: comparison.previous_aov,
          revenuePerDay: comparison.revenue_per_day,
          previousRevenuePerDay: comparison.previous_revenue_per_day,
          percentChange,
          comparisonType,
          isPartial: meta.is_partial,
          periodInfo: meta
        };
      }

    } catch {
      toast.error('Failed to load sales data');
    } finally {
      loading = false;
    }
  }

  function handlePageChange(newOffset, newLimit) {
    offset = newOffset;
    limit = newLimit;
    fetchSales();
  }

  function handleTabChange(newTab) {
    activeTab = newTab;
    offset = 0; // Reset pagination

    // Set default date ranges based on tab
    const now = new Date();
    if (newTab === 'daily') {
      startDate = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000).toISOString().slice(0, 10); // 7 days ago
      endDate = now.toISOString().slice(0, 10);
    } else if (newTab === 'weekly') {
      startDate = new Date(now.getTime() - 84 * 24 * 60 * 60 * 1000).toISOString().slice(0, 10); // 12 weeks ago
      endDate = now.toISOString().slice(0, 10);
    } else if (newTab === 'monthly') {
      startDate = new Date(now.getFullYear(), now.getMonth() - 11, 1).toISOString().slice(0, 10); // 12 months ago
      endDate = now.toISOString().slice(0, 10);
    }

    fetchSales();
  }

  async function exportToExcel() {
    try {
      const { utils, writeFile } = await import('xlsx');

      // Create summary sheet
      const summaryData = [
        ['Revenue Report Summary', ''],
        ['Period', `${startDate} to ${endDate}`],
        ['Granularity', activeTab],
        ['', ''],
        ['Total Revenue', `Rp ${kpiData.totalRevenue.toLocaleString('id-ID')}`],
        ['Total Orders', kpiData.totalOrders],
        ['Average Order Value', `Rp ${kpiData.avgOrderValue.toLocaleString('id-ID', { maximumFractionDigits: 0 })}`],
        ['Change vs Previous Period', `${kpiData.percentChange >= 0 ? '+' : ''}${kpiData.percentChange.toFixed(1)}%`]
      ];

      // Create data sheet
      const headers = activeTab === 'daily' ? ['Date', 'Revenue'] :
                     activeTab === 'weekly' ? ['Week Start', 'Week End', 'Revenue', 'Orders'] :
                     ['Month', 'Revenue', 'Orders'];

      const dataRows = chartData.map(item => {
        if (activeTab === 'daily') return [item.date, item.total];
        if (activeTab === 'weekly') return [item.week_start, item.week_end, item.total, item.order_count];
        return [item.month, item.total, item.order_count];
      });

      const dataData = [headers, ...dataRows];

      const workbook = utils.book_new();
      utils.book_append_sheet(workbook, utils.aoa_to_sheet(summaryData), 'Summary');
      utils.book_append_sheet(workbook, utils.aoa_to_sheet(dataData), 'Data');

      const fileName = `revenue-report-${activeTab}-${startDate}-to-${endDate}.xlsx`;
      writeFile(workbook, fileName);

      toast.success('Excel export completed');
    } catch (error) {
      toast.error('Failed to export to Excel');
    }
  }

  async function exportToPDF() {
    try {
      const { jsPDF } = await import('jspdf');
      const { default: autoTable } = await import('jspdf-autotable');

      const doc = new jsPDF();

      // Title
      doc.setFontSize(16);
      doc.text(`Revenue Report - ${activeTab.charAt(0).toUpperCase() + activeTab.slice(1)}`, 20, 20);

      // Period
      doc.setFontSize(12);
      doc.text(`Period: ${startDate} to ${endDate}`, 20, 30);

      // Summary table
      const summaryBody = [
        ['Total Revenue', `Rp ${kpiData.totalRevenue.toLocaleString('id-ID')}`],
        ['Total Orders', kpiData.totalOrders.toString()],
        ['Average Order Value', `Rp ${kpiData.avgOrderValue.toLocaleString('id-ID', { maximumFractionDigits: 0 })}`],
        ['Change vs Previous Period', `${kpiData.percentChange >= 0 ? '+' : ''}${kpiData.percentChange.toFixed(1)}%`]
      ];

      autoTable(doc, {
        startY: 40,
        head: [['Metric', 'Value']],
        body: summaryBody,
        theme: 'grid'
      });

      // Data table
      let dataHeaders, dataBody;
      if (activeTab === 'daily') {
        dataHeaders = ['Date', 'Revenue'];
        dataBody = chartData.map(item => [item.date, `Rp ${item.total.toLocaleString('id-ID')}`]);
      } else if (activeTab === 'weekly') {
        dataHeaders = ['Week Start', 'Week End', 'Revenue', 'Orders'];
        dataBody = chartData.map(item => [
          item.week_start,
          item.week_end,
          `Rp ${item.total.toLocaleString('id-ID')}`,
          item.order_count
        ]);
      } else {
        dataHeaders = ['Month', 'Revenue', 'Orders'];
        dataBody = chartData.map(item => [
          item.month,
          `Rp ${item.total.toLocaleString('id-ID')}`,
          item.order_count
        ]);
      }

      autoTable(doc, {
        startY: doc.lastAutoTable.finalY + 10,
        head: [dataHeaders],
        body: dataBody,
        theme: 'grid'
      });

      const fileName = `revenue-report-${activeTab}-${startDate}-to-${endDate}.pdf`;
      doc.save(fileName);

      toast.success('PDF export completed');
    } catch (error) {
      toast.error('Failed to export to PDF');
    }
  }

  const statusVariant = (s) =>
    s === 'completed' ? 'success' : s === 'refunded' ? 'danger' : 'warning';

  function openTransactionDetails(transaction) {
    selectedTransaction = transaction;
    showTransactionModal = true;
  }

  onMount(fetchSales);
</script>

<svelte:window 
  onclick={() => { if(showExportDropdown) showExportDropdown = false; }} 
  onkeydown={(e) => { if(e.key === 'Escape' && showExportDropdown) showExportDropdown = false; }} 
/>

<div class="space-y-5">

  <!-- Date range filter -->
  <div class="card p-4 flex flex-wrap items-center gap-4">
    <div class="flex items-center gap-2 text-sm font-medium text-text-secondary">
      <CalendarDays size={16} class="text-white" />
      Date Range
    </div>
    <div class="flex items-center gap-1 bg-surface-subtle border border-border/50 rounded-full p-1 shadow-inner ring-1 ring-black/20">
      <input type="date" class="bg-transparent text-sm text-text-primary outline-none px-3 py-1 cursor-pointer w-36 focus:text-primary-light transition-colors" bind:value={startDate} max={endDate} title={startDateTooltip} />
      <span class="text-text-muted text-sm px-1">-</span>
      <input type="date" class="bg-transparent text-sm text-text-primary outline-none px-3 py-1 cursor-pointer w-36 focus:text-primary-light transition-colors" bind:value={endDate} min={startDate} max={new Date().toISOString().slice(0,10)} title={endDateTooltip} />
    </div>
    <button class="btn btn-primary btn-sm" onclick={() => { offset = 0; fetchSales(); }}>Apply</button>
    <div class="ml-auto relative">
      <button
        class="btn btn-primary flex items-center gap-2 transition-all duration-300"
        onclick={(e) => { e.stopPropagation(); showExportDropdown = !showExportDropdown; }}
        aria-haspopup="menu"
        aria-expanded={showExportDropdown}
      >
        <Download size={15} />
        Export
        <ChevronDown 
          size={14} 
          class="transition-transform duration-300 {showExportDropdown ? 'rotate-180' : ''}" 
        />
      </button>
      {#if showExportDropdown}
        <div 
          class="absolute right-0 top-full mt-2 card-glass p-1.5 z-50 min-w-44 flex flex-col gap-0.5" 
          onclick={(e) => e.stopPropagation()} 
          onkeydown={(e) => e.stopPropagation()} 
          role="menu"
          tabindex="-1"
          transition:fly={{ y: -8, duration: 200 }}
        >
          <button
            class="flex items-center gap-3 px-3 py-2 text-sm font-medium text-text-secondary hover:text-text-primary hover:bg-surface-hover rounded-xl transition-all duration-200 active:scale-[0.98] w-full text-left"
            role="menuitem"
            onclick={() => { showExportDropdown = false; exportToExcel(); }}
          >
            <FileSpreadsheet size={16} class="text-success-light" />
            Export Chart to Excel
          </button>
          <button
            class="flex items-center gap-3 px-3 py-2 text-sm font-medium text-text-secondary hover:text-text-primary hover:bg-surface-hover rounded-xl transition-all duration-200 active:scale-[0.98] w-full text-left"
            role="menuitem"
            onclick={() => { showExportDropdown = false; exportToPDF(); }}
          >
            <Download size={16} class="text-danger-light" />
            Export Chart to PDF
          </button>
        </div>
      {/if}
    </div>
  </div>

  <!-- Tab Navigation -->
  <div class="flex items-center gap-1 bg-surface-subtle border border-border/50 rounded-full p-1 shadow-inner overflow-x-auto">
    <button
      class="px-4 py-2 text-sm font-medium rounded-full transition-all duration-200 {activeTab === 'daily' ? 'bg-primary text-white shadow-sm' : 'text-text-secondary hover:text-text-primary hover:bg-surface-hover'}"
      onclick={() => handleTabChange('daily')}
    >
      Daily
    </button>
    <button
      class="px-4 py-2 text-sm font-medium rounded-full transition-all duration-200 {activeTab === 'weekly' ? 'bg-primary text-white shadow-sm' : 'text-text-secondary hover:text-text-primary hover:bg-surface-hover'}"
      onclick={() => handleTabChange('weekly')}
    >
      Weekly
    </button>
    <button
      class="px-4 py-2 text-sm font-medium rounded-full transition-all duration-200 {activeTab === 'monthly' ? 'bg-primary text-white shadow-sm' : 'text-text-secondary hover:text-text-primary hover:bg-surface-hover'}"
      onclick={() => handleTabChange('monthly')}
    >
      Monthly
    </button>
  </div>

  <!-- Chart -->
  <div class="card p-5">
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-sm font-semibold text-text-primary">
        Revenue Overview - {activeTab.charAt(0).toUpperCase() + activeTab.slice(1)}
      </h3>
    </div>

    <!-- KPI Cards -->
    <div class="grid grid-cols-2 lg:grid-cols-5 gap-4 mb-6">
      {#if loading}
        {#each { length: 5 } as _}
          <div class="bg-surface rounded-lg p-4 border border-border/50">
            <Skeleton width="w-20" height="h-3" class="mb-2" />
            <Skeleton width="w-16" height="h-6" />
          </div>
        {/each}
      {:else}
        <div class="bg-surface rounded-lg p-4 border border-border/50">
          <div class="text-xs font-medium text-text-secondary uppercase tracking-wide">Total Revenue</div>
          <div class="text-lg font-bold text-text-primary mt-1">
            Rp {kpiData.totalRevenue.toLocaleString('id-ID')}
          </div>
          {#if kpiData.previousRevenue > 0}
            <div class="text-xs text-text-muted mt-1">
              vs Rp {kpiData.previousRevenue.toLocaleString('id-ID')}
            </div>
          {/if}
        </div>

        <div class="bg-surface rounded-lg p-4 border border-border/50">
          <div class="text-xs font-medium text-text-secondary uppercase tracking-wide">Total Orders</div>
          <div class="text-lg font-bold text-text-primary mt-1">
            {kpiData.totalOrders.toLocaleString()}
          </div>
          {#if kpiData.previousOrders > 0}
            <div class="text-xs text-text-muted mt-1">
              vs {kpiData.previousOrders.toLocaleString()}
            </div>
          {/if}
        </div>

        <div class="bg-surface rounded-lg p-4 border border-border/50">
          <div class="text-xs font-medium text-text-secondary uppercase tracking-wide">Avg Order Value</div>
          <div class="text-lg font-bold text-text-primary mt-1">
            Rp {kpiData.avgOrderValue.toLocaleString('id-ID', { maximumFractionDigits: 0 })}
          </div>
          {#if kpiData.previousAvgOrderValue > 0}
            <div class="text-xs text-text-muted mt-1">
              vs Rp {kpiData.previousAvgOrderValue.toLocaleString('id-ID', { maximumFractionDigits: 0 })}
            </div>
          {/if}
        </div>

        <div class="bg-surface rounded-lg p-4 border border-border/50">
          <div class="text-xs font-medium text-text-secondary uppercase tracking-wide">Revenue per Day</div>
          <div class="text-lg font-bold text-text-primary mt-1">
            Rp {kpiData.revenuePerDay.toLocaleString('id-ID')}
          </div>
          {#if kpiData.previousRevenuePerDay > 0}
            <div class="text-xs text-text-muted mt-1">
              vs Rp {kpiData.previousRevenuePerDay.toLocaleString('id-ID')}
            </div>
          {/if}
        </div>

        <div class="bg-surface rounded-lg p-4 border border-border/50">
          <div class="text-xs font-medium text-text-secondary uppercase tracking-wide">
            vs Previous Period
            {#if kpiData.isPartial}
              <span class="ml-1 text-[10px] bg-warning/20 text-warning px-1.5 py-0.5 rounded">
                Partial
              </span>
            {/if}
          </div>
          <div class="flex items-center mt-1">
            <span class={`text-lg font-bold ${
              kpiData.comparisonType === 'new' ? 'text-success' :
              kpiData.comparisonType === 'zero' ? 'text-text-secondary' :
              kpiData.percentChange > 0 ? 'text-success' : 'text-danger'
            }`}>
              {kpiData.comparisonType === 'new' ? 'NEW' :
               kpiData.comparisonType === 'zero' ? '±0%' :
               kpiData.percentChange >= 0 ? '+' + kpiData.percentChange.toFixed(1) + '%' :
               kpiData.percentChange.toFixed(1) + '%'}
            </span>
            <span class={`ml-2 ${
              kpiData.comparisonType === 'new' ? 'text-success' :
              kpiData.comparisonType === 'zero' ? 'text-text-secondary' :
              kpiData.percentChange > 0 ? 'text-success' : 'text-danger'
            }`}>
              {kpiData.comparisonType === 'new' ? '🚀' :
               kpiData.comparisonType === 'zero' ? '—' :
               kpiData.percentChange > 0 ? '↗' : '↘'}
            </span>
          </div>
          {#if kpiData.periodInfo}
            <div class="text-xs text-text-muted mt-1">
              {kpiData.periodInfo.current_period.start} to {kpiData.periodInfo.current_period.end}
            </div>
          {/if}
        </div>
      {/if}
    </div>

    <div class="h-64 relative">
      {#if loading}
        <div class="absolute inset-0 flex items-center justify-center rounded-xl border border-dashed border-primary/30 bg-primary-subtle/10 shadow-glow-primary-sm overflow-hidden">
          <div class="absolute inset-0 bg-linear-to-r from-transparent via-primary-subtle/20 to-transparent animate-shimmer" style="background-size: 200% 100%;"></div>
        </div>
      {:else}
        <canvas use:chart={chartConfig}></canvas>
      {/if}
    </div>
  </div>

  <!-- Sales table -->
  <div class="card p-0 overflow-hidden">
    <div class="px-4 py-3 border-b border-border flex items-center justify-between">
      <p class="text-sm font-semibold text-text-primary">Transaction History</p>
      {#if !loading}
        <span class="badge badge-muted">{total} records</span>
      {/if}
    </div>

    {#if loading}
      <div class="divide-y divide-border">
        {#each { length: 5 } as _}
          <div class="flex items-center gap-4 px-4 py-3.5">
            <Skeleton width="w-32" height="h-4" />
            <Skeleton width="w-24" height="h-4" />
            <Skeleton width="w-20" height="h-6" rounded="rounded-full" class="ml-auto" />
            <Skeleton width="w-28" height="h-4" />
          </div>
        {/each}
      </div>
    {:else if salesData.length === 0}
      <div class="px-4 py-12 text-center">
        <div class="empty-state-icon bg-surface w-20 h-20 mx-auto">
          <Receipt size={32} class="text-text-muted" />
        </div>
        <p class="text-text-primary font-semibold mt-4">No transactions found</p>
        <p class="text-text-muted text-sm mt-1">Try adjusting the date range</p>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table>
          <thead class="sticky top-0 bg-bg-secondary z-10 shadow-sm">
            <tr>
              <th>Invoice</th>
              <th>Date</th>
              <th>Items</th>
              <th>Payment</th>
              <th class="text-right">Total (Rp)</th>
            </tr>
          </thead>
          <tbody>
            {#each salesData as sale (sale.id)}
              <tr class="border-t border-border hover:bg-surface-hover/50 transition-colors">
                <td>
                  <button
                    class="font-mono text-sm font-medium text-white hover:text-primary-light transition-colors flex items-center gap-1.5 group underline decoration-border-strong underline-offset-4 hover:decoration-primary-light cursor-pointer"
                    onclick={() => openTransactionDetails(sale)}
                  >
                    <Eye size={14} class="opacity-70 group-hover:opacity-100 transition-opacity" />
                    {sale.invoice_number}
                  </button>
                </td>
                <td class="text-sm text-text-secondary">
                  {formatDateTime(new Date(sale.created_at))}
                </td>
                <td class="text-sm text-text-secondary">
                  {sale.items?.length || 0} items
                </td>
                <td>
                  <span class="text-sm text-text-secondary capitalize">
                    {sale.payment_method || '—'}
                  </span>
                </td>
                <td class="text-right text-sm font-semibold text-text-primary">
                  {(sale.total_amount || 0).toLocaleString('id-ID')}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

      <div class="p-4 bg-surface-subtle/30">
        <Pagination 
          {total} 
          {limit} 
          {offset} 
          onPageChange={handlePageChange} 
        />
      </div>
    {/if}
  </div>

  <!-- Transaction Details Modal -->
  <Modal bind:open={showTransactionModal} title="Transaction Details" size="md">
    {#if selectedTransaction}
      <div class="space-y-4">
        <div>
          <p class="text-sm font-medium text-text-secondary">Invoice Number</p>
          <p class="text-text-primary">{selectedTransaction.invoice_number}</p>
        </div>
        <div>
          <p class="text-sm font-medium text-text-secondary">Date & Time</p>
          <p class="text-text-primary">{formatDateTime(new Date(selectedTransaction.created_at))}</p>
        </div>
        <div>
          <p class="text-sm font-medium text-text-secondary">Payment Method</p>
          <p class="text-text-primary capitalize">{selectedTransaction.payment_method || '—'}</p>
        </div>
        <div>
          <p class="text-sm font-medium text-text-secondary">Status</p>
          <Badge variant={statusVariant(selectedTransaction.status)} class="mt-1">
            {selectedTransaction.status || 'completed'}
          </Badge>
        </div>
        {#if selectedTransaction.items && selectedTransaction.items.length > 0}
          <div>
            <p class="text-sm font-medium text-text-secondary mb-2 block">Items</p>
            <div class="space-y-2">
              {#each selectedTransaction.items as item}
                <div class="flex justify-between items-center py-2 px-3 bg-surface rounded-md border border-border">
                  <div>
                    <p class="text-sm font-medium text-text-primary">{item.name}</p>
                    <p class="text-xs text-text-secondary">Qty: {item.quantity}</p>
                  </div>
                  <p class="text-sm text-text-primary">{(item.unit_price * item.quantity).toLocaleString('id-ID')}</p>
                </div>
              {/each}
            </div>
          </div>
        {/if}
        <div class="border-t border-border pt-4">
          <div class="flex justify-between items-center">
            <span class="text-sm font-medium text-text-secondary">Total Amount</span>
            <span class="text-lg font-semibold text-text-primary">Rp {(selectedTransaction.total_amount || 0).toLocaleString('id-ID')}</span>
          </div>
        </div>
      </div>
    {/if}
  </Modal>
</div>

<style>
  /* Style the calendar picker indicator (WebKit browsers) */
  input[type="date"]::-webkit-calendar-picker-indicator {
    filter: invert(1) brightness(2); /* Make it white */
    cursor: pointer;
  }

  /* Firefox fallback */
  input[type="date"]::-moz-calendar-picker-indicator {
    filter: invert(1) brightness(2);
    cursor: pointer;
  }
</style>