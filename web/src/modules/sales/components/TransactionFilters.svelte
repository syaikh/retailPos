<script lang="ts">
  import { fly } from 'svelte/transition';
  import { Button, Input, SearchBar } from '$shared/ui';
  import { CalendarDays, ChevronDown, Download, FileSpreadsheet } from 'lucide-svelte';
  import { getTodayInJakarta, getDateNDaysAgoInJakarta, formatJakartaDateStr } from '$shared/utils/jakartaTime';
  import { getAuthToken } from '$modules/auth';
  import { toast } from '$shared/stores/toast.svelte';

  const SLIDER_MAX_BOUND = 50000000;

  const datePresets = [
    { label: 'Today', days: 0 },
    { label: 'Yesterday', days: 1 },
    { label: 'Last 7 Days', days: 7 },
    { label: 'Last 30 Days', days: 30 },
    { label: 'This Month', days: 'month' as const },
    { label: 'This Year', days: 'year' as const },
  ];

  let {
    searchQuery = $bindable(''),
    startDate = $bindable(),
    endDate = $bindable(),
    selectedPaymentMethods = $bindable<string[]>([]),
    showDatePicker = $bindable(false),
    showPaymentDropdown = $bindable(false),
    showExportDropdown = $bindable(false),
    sliderMin = $bindable<number | null>(null),
    sliderMax = $bindable<number | null>(null),
    appliedPaymentMethods = $bindable<string[]>([]),
    appliedSliderMin = $bindable<number | null>(null),
    appliedSliderMax = $bindable<number | null>(null),
    loading = false,
    isSearching = $bindable(false),
    selectedDateRange = $bindable('last30d'),
    paymentMethodOptions = [] as { code: string; name: string }[],
    onsearch = () => {},
    onapplyfilters = () => {},
    oncancelfilters = () => {},
    onresetfilters = () => {},
    onapplydaterange = () => {},
    onexportcsv = () => {},
    onexportxlsx = () => {},
  } = $props();

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
    onapplydaterange();
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
    onapplyfilters();
  }

  function cancelFilters() {
    selectedPaymentMethods = [...appliedPaymentMethods];
    sliderMin = appliedSliderMin;
    sliderMax = appliedSliderMax;
    oncancelfilters();
  }

  function resetFilters() {
    selectedPaymentMethods = [];
    sliderMin = null;
    sliderMax = null;
    appliedPaymentMethods = [];
    appliedSliderMin = null;
    appliedSliderMax = null;
    onresetfilters();
  }

  function paymentMethodName(code: string): string {
    return paymentMethodOptions.find(p => p.code === code)?.name || code;
  }

  function buildExportUrl(format: string): string {
    const params = new URLSearchParams({
      format,
      startDate,
      endDate,
      search: sanitizeSearch(searchQuery),
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
    onexportcsv();
  }

  function exportExcel() {
    downloadExport('xlsx');
    showExportDropdown = false;
    onexportxlsx();
  }

  function applyCustomRange() {
    selectedDateRange = 'custom';
    showDatePicker = false;
    onapplydaterange();
  }
</script>

<div class="card p-4 space-y-3">
  <div class="flex items-center justify-between gap-4">
    <div class="min-w-0 flex-1">
      <SearchBar bind:value={searchQuery} placeholder="Search by invoice, product, or customer..." oninput={onsearch} loading={isSearching} />
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
