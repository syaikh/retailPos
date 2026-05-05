<script>
  import { onMount } from 'svelte';
  import { fly } from 'svelte/transition';
  import { apiFetch } from '$lib/api/client';
  import { toast } from '$lib/stores/toast';
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
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);

  // Date range
  let startDate = $state(new Date(new Date().setDate(1)).toISOString().slice(0, 10));
  let endDate = $state(new Date().toISOString().slice(0, 10));

  // Export dropdown
  let showExportDropdown = $state(false);

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



  async function fetchSales() {
    try {
      loading = true;
      const params = new URLSearchParams({
        startDate,
        endDate,
        limit: limit.toString(),
        offset: offset.toString()
      });
      const r = await apiFetch(`/api/sales?${params.toString()}`);
      if (r.ok) {
        const data = await r.json();
        salesData = data.data || [];
        total = data.total || 0;
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

  const statusVariant = (s) =>
    s === 'completed' ? 'success' : s === 'refunded' ? 'danger' : 'warning';

  function openTransactionDetails(transaction) {
    selectedTransaction = transaction;
    showTransactionModal = true;
  }

  onMount(fetchSales);
</script>

<div class="space-y-5">

  <!-- Date range filter -->
  <div class="card p-4 flex flex-wrap items-center gap-4" onclick={() => showExportDropdown = false} onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') showExportDropdown = false; }} role="region" tabindex="0">
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
          transition:fly={{ y: -8, duration: 200 }}
        >
          <button 
            class="flex items-center gap-3 px-3 py-2 text-sm font-medium text-text-secondary hover:text-text-primary hover:bg-surface-hover rounded-xl transition-all duration-200 active:scale-[0.98] w-full text-left"
            role="menuitem"
          >
            <FileSpreadsheet size={16} class="text-success-light" />
            Export to Excel
          </button>
          <button 
            class="flex items-center gap-3 px-3 py-2 text-sm font-medium text-text-secondary hover:text-text-primary hover:bg-surface-hover rounded-xl transition-all duration-200 active:scale-[0.98] w-full text-left"
            role="menuitem"
          >
            <Download size={16} class="text-danger-light" />
            Export to PDF
          </button>
        </div>
      {/if}
    </div>
  </div>

  <!-- Chart placeholder -->
  <div class="card p-5">
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-sm font-semibold text-text-primary">Revenue Overview</h3>
      <span class="badge badge-muted">This period</span>
    </div>
    <div class="flex items-center justify-center h-48 rounded-xl border border-dashed border-primary/30 bg-primary-subtle/10 shadow-glow-primary-sm overflow-hidden relative">
      <div class="absolute inset-0 bg-gradient-to-r from-transparent via-primary-subtle/20 to-transparent animate-shimmer" style="background-size: 200% 100%;"></div>
      <div class="text-center">
        <BarChart3 size={36} class="text-text-muted mx-auto mb-2 opacity-40" />
        <p class="text-text-muted text-sm">Chart visualization</p>
        <p class="text-text-muted text-xs mt-1">Connect Chart.js to render data</p>
      </div>
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
              <th>Status</th>
              <th class="text-right">Total</th>
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
                <td>
                  <Badge variant={statusVariant(sale.status)}>
                    {sale.status || 'completed'}
                  </Badge>
                </td>
                <td class="text-right text-sm font-semibold text-text-primary">
                  Rp {(sale.total_amount || 0).toLocaleString('id-ID')}
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
                  <p class="text-sm text-text-primary">{(item.price * item.quantity).toLocaleString('id-ID')}</p>
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