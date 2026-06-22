<script lang="ts">
  import { onMount } from 'svelte';
  import { fly } from 'svelte/transition';
  import { apiFetch } from '$lib/api/client';
  import { getAuthToken } from '$lib/stores/auth';
  import { toast } from '$lib/stores/toast';
  import { printReceipt as printReceiptStore } from '$lib/stores/printReceipt';
  import { getTodayInJakarta, getDateNDaysAgoInJakarta, formatDateTimeInJakarta, formatJakartaDateStr } from '$lib/utils/jakartaTime';
  import { debounce } from '$lib/utils/debounce';
  import Button from '$lib/components/ui/Button.svelte';
  import SearchBar from '$lib/components/ui/SearchBar.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Skeleton from '$lib/components/ui/Skeleton.svelte';
  import Pagination from '$lib/components/ui/Pagination.svelte';
  import { Printer, Download, FileSpreadsheet, Banknote, X, CalendarDays, ChevronDown } from 'lucide-svelte';

  let loading = $state(true);
  let salesData = $state([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let searchQuery = $state('');
  let isSearching = $state(false);

  let startDate = $state(getDateNDaysAgoInJakarta(30));
  let endDate = $state(getTodayInJakarta());

  let showTransactionDrawer = $state(false);
  let selectedTransaction = $state(null);

  let paymentMethodOptions: { code: string; name: string }[] = $state([]);
  let selectedPaymentMethods: string[] = $state([]);
  let showPaymentDropdown = $state(false);
  let showExportDropdown = $state(false);

  const SLIDER_MAX_BOUND = 50000000;

  // Draft state — bound to UI controls, only applied on explicit Apply
  let sliderMin = $state<number | null>(null);
  let sliderMax = $state<number | null>(null);

  // Applied state — source of truth for chips and API calls
  let appliedPaymentMethods: string[] = $state([]);
  let appliedSliderMin = $state<number | null>(null);
  let appliedSliderMax = $state<number | null>(null);

  let sortBy = $state('created_at');
  let sortDir = $state('DESC');

  let showDatePicker = $state(false);
  let selectedDateRange = $state('last30d');

  const datePresets = [
    { label: 'Today', days: 0 },
    { label: 'Yesterday', days: 1 },
    { label: 'Last 7 Days', days: 7 },
    { label: 'Last 30 Days', days: 30 },
    { label: 'This Month', days: 'month' as const },
    { label: 'This Year', days: 'year' as const },
  ];

  const currentYearStart = $derived(getTodayInJakarta().slice(0, 4) + '-01-01');

  const dateRangeLabel = $derived.by(() => {
    if (selectedDateRange === 'custom') {
      return `Custom: ${formatJakartaDateStr(startDate)} – ${formatJakartaDateStr(endDate)}`;
    }
    const preset = datePresets.find(p => {
      if (p.label === 'Yesterday') return selectedDateRange === 'yesterday';
      if (typeof p.days === 'number') {
        if (p.days === 0) return selectedDateRange === 'today';
        return selectedDateRange === `last${p.days}d`;
      }
      if (p.days === 'month') return selectedDateRange === 'thisMonth';
      if (p.days === 'year') return selectedDateRange === 'thisYear';
      return false;
    });
    return preset?.label || 'Last 30 Days';
  });

  const isFiltered = $derived(
    appliedPaymentMethods.length > 0 ||
    (appliedSliderMin !== null && appliedSliderMin > 0) ||
    (appliedSliderMax !== null && appliedSliderMax < SLIDER_MAX_BOUND)
  );

  const hasPendingChanges = $derived(
    (sliderMin ?? 0) !== (appliedSliderMin ?? 0) ||
    (sliderMax ?? SLIDER_MAX_BOUND) !== (appliedSliderMax ?? SLIDER_MAX_BOUND) ||
    selectedPaymentMethods.length !== appliedPaymentMethods.length ||
    (selectedPaymentMethods.length > 0 && selectedPaymentMethods.some(c => !appliedPaymentMethods.includes(c))) ||
    (appliedPaymentMethods.length > 0 && appliedPaymentMethods.some(c => !selectedPaymentMethods.includes(c)))
  );

  const amountError = $derived.by(() => {
    if (sliderMin !== null && sliderMin < 0) return 'Min cannot be negative';
    if (sliderMin !== null && sliderMin > SLIDER_MAX_BOUND) return `Min exceeds max (${SLIDER_MAX_BOUND.toLocaleString('id-ID')})`;
    if (sliderMax !== null && sliderMax < 0) return 'Max cannot be negative';
    if (sliderMax !== null && sliderMax > SLIDER_MAX_BOUND) return `Max exceeds max (${SLIDER_MAX_BOUND.toLocaleString('id-ID')})`;
    if (sliderMin !== null && sliderMax !== null && sliderMin > sliderMax) return 'Min cannot exceed Max';
    return '';
  });

  const minDisplay = $derived(sliderMin !== null ? sliderMin.toLocaleString('id-ID') : '');
  const maxDisplay = $derived(sliderMax !== null ? sliderMax.toLocaleString('id-ID') : '');

  function handleMinInput(e: Event) {
    const input = e.target as HTMLInputElement;
    const digits = input.value.replace(/\D/g, '');
    sliderMin = digits === '' ? null : parseInt(digits, 10);
    const formatted = sliderMin !== null ? sliderMin.toLocaleString('id-ID') : '';
    if (input.value !== formatted) {
      input.value = formatted;
      input.setSelectionRange(formatted.length, formatted.length);
    }
  }

  function handleMaxInput(e: Event) {
    const input = e.target as HTMLInputElement;
    const digits = input.value.replace(/\D/g, '');
    sliderMax = digits === '' ? null : parseInt(digits, 10);
    const formatted = sliderMax !== null ? sliderMax.toLocaleString('id-ID') : '';
    if (input.value !== formatted) {
      input.value = formatted;
      input.setSelectionRange(formatted.length, formatted.length);
    }
  }

  function sanitizeSearch(q: string): string {
    let s = q.trim();
    if (/^INV-/i.test(s)) s = s.slice(4).trim();
    return s;
  }

  const statusVariant = (s: string) =>
    s === 'completed' ? 'success' : s === 'refunded' ? 'danger' : 'warning';

  function getPaymentMethodVariant(method = '') {
    if (!method) return 'muted';
    const m = method.toLowerCase();
    if (m === 'cash') return 'success';
    if (m === 'qris' || m === 'e_wallet') return 'default';
    if (m === 'card') return 'primary';
    if (m === 'transfer') return 'muted';
    return 'muted';
  }

  const formatDateTime = (date: Date) => {
    const isoStr = date instanceof Date ? date.toISOString() : String(date);
    return formatDateTimeInJakarta(isoStr);
  };

  function applyDatePreset(days: number | 'month' | 'year') {
    if (days === 'year') {
      const today = getTodayInJakarta();
      startDate = today.slice(0, 4) + '-01-01';
      endDate = today;
      selectedDateRange = 'thisYear';
    } else if (days === 'month') {
      const todayJakarta = getTodayInJakarta();
      const parts = todayJakarta.split('-').map(Number);
      startDate = `${parts[0]}-${String(parts[1]).padStart(2, '0')}-01`;
      endDate = todayJakarta;
      selectedDateRange = 'thisMonth';
    } else if (days <= 1) {
      startDate = getDateNDaysAgoInJakarta(days);
      endDate = startDate;
      selectedDateRange = days === 0 ? 'today' : 'yesterday';
    } else {
      startDate = getDateNDaysAgoInJakarta(days);
      endDate = getTodayInJakarta();
      selectedDateRange = `last${days}d`;
    }
    showDatePicker = false;
    offset = 0;
    fetchSales(false);
  }

  function togglePaymentMethod(code: string) {
    if (selectedPaymentMethods.includes(code)) {
      selectedPaymentMethods = selectedPaymentMethods.filter(c => c !== code);
    } else {
      selectedPaymentMethods = [...selectedPaymentMethods, code];
    }
  }

  function applyFilters() {
    appliedPaymentMethods = [...selectedPaymentMethods];
    appliedSliderMin = sliderMin;
    appliedSliderMax = sliderMax;
    offset = 0;
    fetchSales(false);
  }

  function cancelFilters() {
    selectedPaymentMethods = [...appliedPaymentMethods];
    sliderMin = appliedSliderMin;
    sliderMax = appliedSliderMax;
  }

  function resetFilters() {
    selectedPaymentMethods = [];
    sliderMin = null;
    sliderMax = null;
    appliedPaymentMethods = [];
    appliedSliderMin = null;
    appliedSliderMax = null;
    offset = 0;
    fetchSales(false);
  }

  function paymentMethodName(code: string): string {
    return paymentMethodOptions.find(p => p.code === code)?.name || code;
  }

  async function fetchSales(isSearch = false) {
    try {
      if (!isSearch) loading = true;
      const params = new URLSearchParams({
        startDate,
        endDate,
        limit: limit.toString(),
        offset: offset.toString(),
        search: sanitizeSearch(searchQuery),
        sortBy,
        sortDir,
      });
      if (appliedPaymentMethods.length > 0) {
        params.set('paymentMethods', appliedPaymentMethods.join(','));
      }
      if (appliedSliderMin !== null && appliedSliderMin > 0) {
        params.set('minTotal', appliedSliderMin.toString());
      }
      if (appliedSliderMax !== null && appliedSliderMax < SLIDER_MAX_BOUND) {
        params.set('maxTotal', appliedSliderMax.toString());
      }
      const res = await apiFetch(`/api/sales?${params.toString()}`);
      if (res.ok) {
        const data = await res.json();
        salesData = data.data || [];
        total = data.total || 0;
      }
    } catch {
      toast.error('Failed to load transactions');
    } finally {
      if (!isSearch) loading = false;
      isSearching = false;
    }
  }

  const doSearch = debounce(() => {
    offset = 0;
    isSearching = true;
    fetchSales(true);
  }, 300);

  function handlePageChange(newOffset: number, newLimit: number) {
    offset = newOffset;
    limit = newLimit;
    fetchSales(false);
  }

  function toggleSort(column: string) {
    if (sortBy === column) {
      sortDir = sortDir === 'ASC' ? 'DESC' : 'ASC';
    } else {
      sortBy = column;
      sortDir = 'ASC';
    }
    offset = 0;
    fetchSales(false);
  }

  function openTransactionDetails(transaction: any) {
    selectedTransaction = transaction;
    showTransactionDrawer = true;
  }

  function closeTransactionDrawer() {
    showTransactionDrawer = false;
  }

  function printTransactionReceipt() {
    if (!selectedTransaction || !selectedTransaction.items) return;
    const taxAmount = selectedTransaction.tax || 0;
    printReceiptStore.set({
      invoice_number: selectedTransaction.invoice_number,
      created_at: selectedTransaction.created_at,
      items: selectedTransaction.items.map((item: any) => ({
        name: item.name,
        quantity: item.quantity,
        unit_price: item.unit_price,
      })),
      total_amount: selectedTransaction.total_amount,
      subtotal_dpp: selectedTransaction.total_amount - taxAmount,
      tax: taxAmount,
      paymentMethod: selectedTransaction.payment_method || '—',
      cashReceived: selectedTransaction.cash_received || selectedTransaction.total_amount,
      changeDue: selectedTransaction.change_due || 0,
      customer_name: selectedTransaction.customer_name || undefined,
    });
    setTimeout(() => window.print(), 300);
  }

  async function downloadInvoice() {
    if (!selectedTransaction) return;
    try {
      const { jsPDF } = await import('jspdf');
      const { default: autoTable } = await import('jspdf-autotable');

      const doc = new jsPDF();
      doc.setFontSize(18);
      doc.text('INVOICE', 20, 20);
      doc.setFontSize(10);
      doc.text(`Invoice #: ${selectedTransaction.invoice_number}`, 20, 30);
      doc.text(`Date: ${formatDateTime(new Date(selectedTransaction.created_at))}`, 20, 36);
      doc.text(`Payment: ${selectedTransaction.payment_method || '—'}`, 20, 42);
      if (selectedTransaction.customer_name) {
        doc.text(`Customer: ${selectedTransaction.customer_name}`, 20, 48);
      }

      const itemRows = (selectedTransaction.items || []).map((item: any) => [
        item.name,
        item.quantity.toString(),
        `Rp ${(item.unit_price || 0).toLocaleString('id-ID')}`,
        `Rp ${(item.unit_price * item.quantity).toLocaleString('id-ID')}`,
      ]);

      autoTable(doc, {
        startY: 58,
        head: [['Item', 'Qty', 'Price', 'Subtotal']],
        body: itemRows,
        theme: 'grid',
        styles: { fontSize: 9 },
        headStyles: { fillColor: [124, 58, 237] },
      });

      const finalY = doc.lastAutoTable.finalY + 10;
      const taxAmt = selectedTransaction.tax || 0;
      if (taxAmt > 0) {
        doc.setFontSize(10);
        doc.text(`Subtotal (DPP): Rp ${((selectedTransaction.total_amount || 0) - taxAmt).toLocaleString('id-ID')}`, 20, finalY);
        doc.text(`PPN 11%: Rp ${taxAmt.toLocaleString('id-ID')}`, 20, finalY + 6);
        doc.setFontSize(12);
        doc.text(`Total: Rp ${(selectedTransaction.total_amount || 0).toLocaleString('id-ID')}`, 20, finalY + 14);
      } else {
        doc.setFontSize(12);
        doc.text(`Total: Rp ${(selectedTransaction.total_amount || 0).toLocaleString('id-ID')}`, 20, finalY);
      }

      doc.save(`invoice-${selectedTransaction.invoice_number}.pdf`);
      toast.success('Invoice downloaded');
    } catch {
      toast.error('Failed to download invoice');
    }
  }

  function buildExportUrl(format: string): string {
    const params = new URLSearchParams({
      format,
      startDate,
      endDate,
      search: sanitizeSearch(searchQuery),
      sortBy,
      sortDir,
    });
    if (appliedPaymentMethods.length > 0) {
      params.set('paymentMethods', appliedPaymentMethods.join(','));
    }
    if (appliedSliderMin !== null && appliedSliderMin > 0) {
      params.set('minTotal', appliedSliderMin.toString());
    }
    if (appliedSliderMax !== null && appliedSliderMax < SLIDER_MAX_BOUND) {
      params.set('maxTotal', appliedSliderMax.toString());
    }
    return `/api/sales/export?${params.toString()}`;
  }

  async function downloadExport(format: string) {
    const token = getAuthToken();
    if (!token) { toast.error('Session expired'); return; }

    const res = await fetch(buildExportUrl(format), {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (!res.ok) { toast.error('Export failed'); return; }

    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `transactions-${getTodayInJakarta()}.${format === 'csv' ? 'csv' : 'xlsx'}`;
    a.click();
    URL.revokeObjectURL(url);
  }

  function exportCsv() {
    downloadExport('csv');
    showExportDropdown = false;
  }

  function exportExcel() {
    downloadExport('xlsx');
    showExportDropdown = false;
  }

  function applyCustomRange() {
    selectedDateRange = 'custom';
    showDatePicker = false;
    offset = 0;
    fetchSales(false);
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      showTransactionDrawer = false;
      showDatePicker = false;
      showPaymentDropdown = false;
      showExportDropdown = false;
    }
  }

  function handleWindowClick(e: MouseEvent) {
    const target = e.target as HTMLElement;
    if (showDatePicker && !target.closest('.date-picker-container') && !target.closest('.date-picker-trigger')) {
      showDatePicker = false;
    }
    if (showPaymentDropdown && !target.closest('.payment-dropdown-container')) {
      showPaymentDropdown = false;
    }
    if (showExportDropdown && !target.closest('.export-dropdown-container')) {
      showExportDropdown = false;
    }
  }

  onMount(async () => {
    await fetchSales(false);
    const res = await apiFetch('/api/payment-methods');
    if (res.ok) {
      const data = await res.json();
      paymentMethodOptions = (data.data || data || []).filter((m: any) => m.is_active !== false);
    }
  });
</script>

<svelte:window onkeydown={handleKeydown} onclick={handleWindowClick} />

<div class="space-y-5">
  <div class="card p-4 space-y-3">
    <div class="flex items-center justify-between gap-4">
      <div class="min-w-0 flex-1">
        <SearchBar bind:value={searchQuery} placeholder="Search by invoice, product, or customer..." oninput={doSearch} loading={isSearching} />
      </div>
      <div class="flex items-center gap-2 shrink-0">
        <div class="relative">
          <Button
            variant="secondary"
            class="flex items-center gap-2 min-w-44 date-picker-trigger"
            onclick={() => showDatePicker = !showDatePicker}
          >
            <CalendarDays size={16} class="text-text-secondary shrink-0" />
            <span class="text-sm font-medium truncate flex-1 text-left text-text-secondary">{dateRangeLabel}</span>
            <ChevronDown size={14} class="opacity-60 shrink-0" />
          </Button>
          {#if showDatePicker}
            <div class="absolute right-0 top-full mt-1.5 z-50 bg-surface-default border border-border rounded-lg shadow-xl p-3 min-w-64 date-picker-container">
              <div class="flex flex-wrap gap-1 mb-3">
                {#each datePresets as preset}
                  <Button
                    variant="ghost"
                    size="xs"
                    onclick={() => applyDatePreset(preset.days)}
                  >
                    {preset.label}
                  </Button>
                {/each}
              </div>
              <div class="flex items-center gap-2 text-xs">
                <Input type="date" bind:value={startDate} class="w-full" min={currentYearStart} max={endDate} />
                <span class="text-text-muted">—</span>
                <Input type="date" bind:value={endDate} class="w-full" min={startDate} max={getTodayInJakarta()} />
              </div>
              <div class="flex justify-end mt-2">
                <Button
                  variant="primary"
                  size="xs"
                  onclick={applyCustomRange}
                >
                  Apply
                </Button>
              </div>
            </div>
          {/if}
        </div>
        <div class="relative export-dropdown-container">
          <Button
            variant="primary"
            class="flex items-center gap-2 transition-all duration-300"
            onclick={() => showExportDropdown = !showExportDropdown}
          >
            <Download size={15} />
            Export
            <ChevronDown size={14} class="transition-transform duration-300 {showExportDropdown ? 'rotate-180' : ''}" />
          </Button>
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
                onclick={() => { showExportDropdown = false; exportCsv(); }}
              >
                <FileSpreadsheet size={16} class="text-success-light" />
                Export to CSV
              </button>
              <button
                class="flex items-center gap-3 px-3 py-2 text-sm font-medium text-text-secondary hover:text-text-primary hover:bg-surface-hover rounded-xl transition-all duration-200 active:scale-[0.98] w-full text-left"
                role="menuitem"
                onclick={() => { showExportDropdown = false; exportExcel(); }}
              >
                <FileSpreadsheet size={16} class="text-info-light" />
                Export to Excel
              </button>
            </div>
          {/if}
        </div>
      </div>
    </div>

    <div class="pt-1">
      <div class="flex flex-wrap items-end gap-3">
        <div class="relative payment-dropdown-container">
          <p class="text-xs font-medium text-text-secondary mb-1.5">Payment</p>
          <Button
            variant="secondary"
            class="flex items-center gap-2 min-w-44"
            onclick={() => showPaymentDropdown = !showPaymentDropdown}
          >
            <span class="text-sm truncate flex-1 text-left text-text-secondary">
              {selectedPaymentMethods.length > 0
                ? `${paymentMethodName(selectedPaymentMethods[0])}${selectedPaymentMethods.length > 1 ? ` +${selectedPaymentMethods.length - 1}` : ''}`
                : 'All methods'}
            </span>
            <ChevronDown size={14} class="opacity-60 shrink-0" />
          </Button>
          {#if showPaymentDropdown}
            <div class="absolute left-0 top-full mt-1.5 z-50 bg-surface-default border border-border rounded-lg shadow-xl p-2 min-w-44 max-h-56 overflow-y-auto">
              {#each paymentMethodOptions as pm}
                <label class="flex items-center gap-2 px-2 py-1.5 rounded hover:bg-surface-hover cursor-pointer text-xs">
                  <input
                    type="checkbox"
                    checked={selectedPaymentMethods.includes(pm.code)}
                    onchange={() => togglePaymentMethod(pm.code)}
                    class="accent-primary"
                  />
                  {pm.name}
                </label>
              {/each}
              {#if selectedPaymentMethods.length > 0}
                <button
                  class="w-full text-left px-2 py-1.5 mt-1 text-xs text-primary hover:bg-surface-hover rounded"
                  onclick={() => { selectedPaymentMethods = []; }}
                >
                  Clear selection
                </button>
              {/if}
            </div>
          {/if}
        </div>

        <div class="w-44">
          <p class="text-xs font-medium text-text-secondary mb-1.5">Min Total Trx (Rp)</p>
          <Input
            type="text"
            inputmode="numeric"
            value={minDisplay}
            placeholder="0"
            class="py-2 w-full text-right {amountError ? 'border-danger focus:border-danger focus:ring-danger/20' : ''}"
            oninput={handleMinInput}
          />
        </div>

        <div class="w-44">
          <p class="text-xs font-medium text-text-secondary mb-1.5">Max Total Trx (Rp)</p>
          <Input
            type="text"
            inputmode="numeric"
            value={maxDisplay}
            placeholder="∞"
            class="py-2 w-full text-right {amountError ? 'border-danger focus:border-danger focus:ring-danger/20' : ''}"
            oninput={handleMaxInput}
          />
        </div>

        {#if amountError}
          <div class="flex items-end h-[42px]">
            <p class="text-xs text-danger">{amountError}</p>
          </div>
        {/if}

        <div class="flex-1 min-w-0"></div>

        <div class="flex items-end gap-2">
          <Button variant="secondary" disabled={!isFiltered} onclick={resetFilters}>
            Reset
          </Button>
          {#if hasPendingChanges}
            <Button variant="secondary" onclick={cancelFilters}>
              Cancel
            </Button>
          {/if}
          <Button variant="primary" disabled={!hasPendingChanges || !!amountError} onclick={applyFilters}>
            Apply
          </Button>
        </div>
      </div>
    </div>
  </div>

  <div class="card p-0 overflow-hidden">
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
        <div class="empty-state-icon bg-surface w-20 h-20 mx-auto flex justify-center">
          <Banknote size={32} class="text-text-muted" />
        </div>
        <p class="text-text-primary font-semibold mt-4">No transactions found</p>
        <p class="text-text-muted text-sm mt-1">Try adjusting the search or date range</p>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full">
          <thead class="bg-muted/50">
            <tr>
              <th class="text-left p-4 font-semibold">
                <button type="button" class="flex items-center gap-1 hover:text-primary transition-colors" onclick={() => toggleSort('invoice_number')}>
                  INVOICE {#if sortBy === 'invoice_number'}<span>{sortDir === 'ASC' ? '▲' : '▼'}</span>{/if}
                </button>
              </th>
              <th class="text-left p-4 font-semibold">
                <button type="button" class="flex items-center gap-1 hover:text-primary transition-colors" onclick={() => toggleSort('created_at')}>
                  DATE {#if sortBy === 'created_at'}<span>{sortDir === 'ASC' ? '▲' : '▼'}</span>{/if}
                </button>
              </th>
              <th class="text-left p-4 font-semibold w-[30%]">CUSTOMER</th>
              <th class="text-left p-4 font-semibold">ITEMS</th>
              <th class="text-left p-4 font-semibold">
                <button type="button" class="flex items-center gap-1 hover:text-primary transition-colors" onclick={() => toggleSort('payment_method')}>
                  PAYMENT {#if sortBy === 'payment_method'}<span>{sortDir === 'ASC' ? '▲' : '▼'}</span>{/if}
                </button>
              </th>
              <th class="text-right p-4 font-semibold">
                <button type="button" class="flex items-center gap-1 hover:text-primary transition-colors justify-end w-full" onclick={() => toggleSort('total_amount')}>
                  TOTAL (RP) {#if sortBy === 'total_amount'}<span>{sortDir === 'ASC' ? '▲' : '▼'}</span>{/if}
                </button>
              </th>
            </tr>
          </thead>
          <tbody>
            {#each salesData as sale (sale.id)}
              <tr
                class="border-t border-border hover:bg-surface-hover/50 transition-colors cursor-pointer"
                onclick={() => openTransactionDetails(sale)}
                onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); openTransactionDetails(sale); } }}
                tabindex="0"
                role="button"
              >
                  <td class="p-4">
                    <span class="text-sm font-medium text-text-primary">
                      {sale.invoice_number}
                    </span>
                </td>
                <td class="p-4 text-sm text-text-secondary">
                  {formatDateTime(new Date(sale.created_at))}
                </td>
                <td class="p-4 text-sm text-text-secondary">
                  {sale.customer_name || '—'}
                </td>
                <td class="p-4 text-sm text-text-secondary">
                  {sale.items?.length || 0} items
                </td>
                <td class="p-4">
                  <Badge variant={getPaymentMethodVariant(sale.payment_method)} class="text-xs px-2.5 py-0.5">
                    {sale.payment_method || '—'}
                  </Badge>
                </td>
                <td class="p-4 text-right text-sm font-semibold text-text-primary">
                  {(sale.total_amount || 0).toLocaleString('id-ID')}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

      <div class="p-4 bg-surface-subtle/30 border-t border-border/50">
        <Pagination
          {total}
          {limit}
          {offset}
          onPageChange={handlePageChange}
        />
      </div>
    {/if}
  </div>
</div>

{#if showTransactionDrawer && selectedTransaction}
  <div class="fixed inset-0 bg-black/60 z-50" onclick={closeTransactionDrawer} aria-hidden="true"></div>
  <div
    class="fixed inset-y-0 right-0 w-[520px] max-w-full bg-surface-default border-l border-border shadow-2xl z-[55] flex flex-col"
    transition:fly={{ x: 520, duration: 300, easing: t => t * (2 - t) }}
    role="dialog" aria-modal="true" aria-labelledby="transaction-details-heading" tabindex="-1"
    onkeydown={(e) => { if (e.key === 'Escape') { e.preventDefault(); closeTransactionDrawer(); } }}
  >
    <div class="flex items-center justify-between px-6 py-5 border-b border-border shrink-0">
      <div class="flex items-center gap-3">
        <h2 id="transaction-details-heading" class="text-lg font-bold text-text-primary">Transaction Details</h2>
        <span class="inline-flex items-center px-2.5 py-0.5 text-xs font-medium rounded-full {statusVariant(selectedTransaction.status) === 'success' ? 'bg-success/20 text-success' : statusVariant(selectedTransaction.status) === 'warning' ? 'bg-warning/20 text-warning' : 'bg-info/20 text-info'}">
          {selectedTransaction.status || 'completed'}
        </span>
      </div>
      <button
        class="p-2 rounded-lg text-text-muted hover:bg-surface-hover hover:text-text-secondary transition-colors"
        onclick={closeTransactionDrawer}
        onkeydown={(e) => { if (e.key === 'Enter' || e.key === 'Escape' || e.key === ' ') { e.preventDefault(); closeTransactionDrawer(); } }}
        title="Close detail" aria-label="Close detail panel"
      >
        <X size={18} />
      </button>
    </div>

    <div class="flex-1 overflow-y-auto px-6 py-4 space-y-5">
      <div class="grid grid-cols-2 gap-x-8 gap-y-4">
        <div class="space-y-3">
          <div>
            <p class="text-xs font-medium text-text-muted uppercase tracking-wide">Invoice Number</p>
            <p class="text-sm font-semibold text-text-primary font-mono">{selectedTransaction.invoice_number}</p>
          </div>
          <div>
            <p class="text-xs font-medium text-text-muted uppercase tracking-wide">Date & Time</p>
            <p class="text-sm text-text-primary">{formatDateTime(new Date(selectedTransaction.created_at))}</p>
          </div>
          <div>
            <p class="text-xs font-medium text-text-muted uppercase tracking-wide">Customer</p>
            <p class="text-sm text-text-primary">{selectedTransaction.customer_name || 'Walk-in / General'}</p>
          </div>
        </div>
        <div class="space-y-3">
          <div>
            <p class="text-xs font-medium text-text-muted uppercase tracking-wide">Payment Method</p>
            <div class="mt-1">
              <span class="inline-flex items-center px-2.5 py-1 text-xs font-medium rounded-full {getPaymentMethodVariant(selectedTransaction.payment_method) === 'success' ? 'bg-success/20 text-success' : getPaymentMethodVariant(selectedTransaction.payment_method) === 'warning' ? 'bg-warning/20 text-warning' : 'bg-primary/20 text-primary'}">
                {selectedTransaction.payment_method || '—'}
              </span>
            </div>
          </div>
        </div>
      </div>

      {#if selectedTransaction.items && selectedTransaction.items.length > 0}
        <div>
          <p class="text-sm font-semibold text-text-secondary mb-3">Items</p>
          <div class="border border-border rounded-lg">
            <div class="max-h-80 overflow-y-auto">
              <table class="w-full text-sm">
                <thead class="sticky top-0 bg-surface-subtle z-10">
                  <tr>
                    <th class="text-left py-3 px-4 font-semibold text-text-primary">Description</th>
                    <th class="text-center py-3 px-4 font-semibold text-text-primary w-20">Qty</th>
                    <th class="text-right py-3 px-4 font-semibold text-text-primary w-28">Price</th>
                    <th class="text-right py-3 px-4 font-semibold text-text-primary w-32">Subtotal</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-border">
                  {#each selectedTransaction.items as item}
                    <tr class="hover:bg-surface/50">
                      <td class="py-3 px-4 text-text-primary">{item.name}</td>
                      <td class="py-3 px-4 text-center text-text-secondary">{item.quantity}</td>
                      <td class="py-3 px-4 text-right text-text-secondary">{(item.unit_price || 0).toLocaleString('id-ID')}</td>
                      <td class="py-3 px-4 text-right font-medium text-text-primary">{(item.unit_price * item.quantity).toLocaleString('id-ID')}</td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
            <div class="bg-surface-subtle/50 border-t border-border">
              {#if selectedTransaction.tax && selectedTransaction.tax > 0}
                <div class="flex justify-between items-center py-2 px-4 text-sm">
                  <span class="text-text-muted">Subtotal (DPP)</span>
                  <span class="text-text-secondary">{((selectedTransaction.total_amount || 0) - selectedTransaction.tax).toLocaleString('id-ID')}</span>
                </div>
                <div class="flex justify-between items-center py-2 px-4 text-sm border-t border-border/50">
                  <span class="text-text-muted">PPN 11%</span>
                  <span class="text-text-secondary">{(selectedTransaction.tax || 0).toLocaleString('id-ID')}</span>
                </div>
              {/if}
              <div class="flex justify-between items-center py-3 px-4 border-t border-border/50">
                <span class="font-bold text-text-primary">TOTAL</span>
                <span class="font-bold text-lg text-text-primary">Rp {(selectedTransaction.total_amount || 0).toLocaleString('id-ID')}</span>
              </div>
            </div>
          </div>
        </div>
      {/if}
    </div>

    <div class="absolute bottom-0 left-0 right-0 p-4 bg-surface-default border-t border-border/50">
      <div class="grid grid-cols-[auto_1fr_1fr] gap-3">
        <Button variant="secondary" class="rounded-xl px-4 h-11 text-sm font-semibold whitespace-nowrap" onclick={closeTransactionDrawer}>
          Close
        </Button>
        <Button variant="secondary" class="rounded-xl px-4 h-11 text-sm font-semibold flex items-center gap-1.5 whitespace-nowrap" onclick={printTransactionReceipt}>
          <Printer size={15} class="mr-1.5" />
          Print Receipt
        </Button>
        <Button variant="primary" class="rounded-xl px-4 h-11 text-sm font-semibold text-white shadow-glow-primary-sm flex items-center gap-1.5 whitespace-nowrap" onclick={downloadInvoice}>
          <Download size={15} class="mr-1.5" />
          Download Invoice
        </Button>
      </div>
    </div>
  </div>
{/if}

<style>
  :global(input[type="date"]::-webkit-calendar-picker-indicator) {
    filter: invert(1);
    cursor: pointer;
  }
</style>
