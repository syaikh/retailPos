<script lang="ts">
  import { Button, Input, SearchBar, Dropdown } from '$shared/ui';
  import { CalendarDays, ChevronDown, Download, FileSpreadsheet, X } from 'lucide-svelte';
  import { getTodayInJakarta, getDateNDaysAgoInJakarta, formatJakartaDateStr } from '$shared/utils/jakartaTime';
  import { getAuthToken } from '$modules/auth';
  import { toast } from '$shared/stores/toast.svelte';
  import { labels, t } from '$shared/i18n';

  const SLIDER_MAX_BOUND = 50000000;

  const datePresets = $derived([
    { label: labels.today, days: 0 },
    { label: labels.yesterday, days: 1 },
    { label: labels.last7Days, days: 7 },
    { label: labels.last30Days, days: 30 },
    { label: labels.thisMonth, days: 'month' as const },
    { label: labels.thisYear, days: 'year' as const },
  ]);

  let {
    searchQuery = $bindable(''),
    startDate = $bindable(),
    endDate = $bindable(),
    selectedPaymentMethods = $bindable<string[]>([]),
    showDatePicker = $bindable(false),
    sliderMin = $bindable<number | null>(null),
    sliderMax = $bindable<number | null>(null),
    selectedDateRange = $bindable('last30d'),
    paymentMethodOptions = [] as { code: string; name: string }[],
    onexportcsv = () => {},
    onexportxlsx = () => {},
  } = $props();

  let editStartDate = $state('');
  let editEndDate = $state('');
  let dropdownOpen = $state(false);
  let pendingPaymentMethods = $state<string[]>([]);

  const currentYearStart = $derived(getTodayInJakarta().slice(0, 4) + '-01-01');

  const dateRangeLabel = $derived.by(() => {
    if (selectedDateRange === 'custom') {
      return t('customDateRange', {
        start: formatJakartaDateStr(startDate),
        end: formatJakartaDateStr(endDate),
      });
    }
    const preset = datePresets.find(p => {
      if (typeof p.days === 'number') {
        if (p.days === 0) return selectedDateRange === 'today';
        if (p.days === 1) return selectedDateRange === 'yesterday';
        return selectedDateRange === `last${p.days}d`;
      }
      if (p.days === 'month') return selectedDateRange === 'thisMonth';
      if (p.days === 'year') return selectedDateRange === 'thisYear';
      return false;
    });
    return preset?.label || labels.last30Days;
  });

  const amountError = $derived.by(() => {
    if (sliderMin !== null && sliderMin < 0) return labels.errorMinCannotBeNegative;
    if (sliderMin !== null && sliderMin > SLIDER_MAX_BOUND)
      return t('errorMinExceedsMax', { max: SLIDER_MAX_BOUND.toLocaleString('id-ID') });
    if (sliderMax !== null && sliderMax < 0) return labels.errorMaxCannotBeNegative;
    if (sliderMax !== null && sliderMax > SLIDER_MAX_BOUND)
      return t('errorMaxExceedsMax', { max: SLIDER_MAX_BOUND.toLocaleString('id-ID') });
    if (sliderMin !== null && sliderMax !== null && sliderMin > sliderMax) return labels.errorMinCannotExceedMax;
    return '';
  });

  const minDisplay = $derived(sliderMin !== null ? sliderMin.toLocaleString('id-ID') : '');
  const maxDisplay = $derived(sliderMax !== null ? sliderMax.toLocaleString('id-ID') : '');

  const canApplyCustom = $derived(
    editStartDate.length > 0 && editEndDate.length > 0 && (editStartDate !== startDate || editEndDate !== endDate)
  );

  $effect(() => {
    if (dropdownOpen) {
      pendingPaymentMethods = [...selectedPaymentMethods];
    }
  });

  $effect(() => {
    if (!showDatePicker) return;

    const handleClickOutside = (e: MouseEvent) => {
      const target = e.target as HTMLElement;
      if (!target.closest('.date-picker-container') && !target.closest('.date-picker-trigger')) {
        showDatePicker = false;
      }
    };

    const frame = requestAnimationFrame(() => {
      document.addEventListener('click', handleClickOutside, true);
    });

    return () => {
      cancelAnimationFrame(frame);
      document.removeEventListener('click', handleClickOutside, true);
    };
  });

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

  function handleMinBlur() {
    if (amountError) toast.error(amountError);
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

  function handleMaxBlur() {
    if (amountError) toast.error(amountError);
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
  }

  function openDatePicker() {
    editStartDate = startDate;
    editEndDate = endDate;
    showDatePicker = true;
  }

  function applyCustomRange() {
    if (!canApplyCustom) return;
    startDate = editStartDate;
    endDate = editEndDate;
    selectedDateRange = 'custom';
    showDatePicker = false;
  }

  function cancelCustomRange() {
    editStartDate = '';
    editEndDate = '';
    showDatePicker = false;
  }

  function togglePendingPaymentMethod(code: string) {
    if (pendingPaymentMethods.includes(code)) {
      pendingPaymentMethods = pendingPaymentMethods.filter(c => c !== code);
    } else {
      pendingPaymentMethods = [...pendingPaymentMethods, code];
    }
  }

  function paymentMethodName(code: string): string {
    return paymentMethodOptions.find(p => p.code === code)?.name || code;
  }

  function buildExportUrl(format: string): string {
    const params = new URLSearchParams({
      format,
      start_date: startDate,
      end_date: endDate,
      search: sanitizeSearch(searchQuery),
    });
    if (selectedPaymentMethods.length > 0) {
      params.set('payment_methods', selectedPaymentMethods.join(','));
    }
    if (sliderMin !== null && sliderMin > 0) {
      params.set('min_total', sliderMin.toString());
    }
    if (sliderMax !== null && sliderMax < SLIDER_MAX_BOUND) {
      params.set('max_total', sliderMax.toString());
    }
    return `/api/sales/export?${params.toString()}`;
  }

  async function downloadExport(format: string) {
    const token = getAuthToken();
    if (!token) { toast.error(labels.toastSessionExpired); return; }

    const res = await fetch(buildExportUrl(format), {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (!res.ok) { toast.error(labels.toastExportFailed); return; }

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
    onexportcsv();
  }

  function exportExcel() {
    downloadExport('xlsx');
    onexportxlsx();
  }

  function handleDatePickerKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      cancelCustomRange();
    } else if (e.key === 'Enter' && canApplyCustom) {
      applyCustomRange();
    }
  }
</script>

<div class="card p-3">
  <div class="flex flex-wrap items-center gap-3">
    <div class="min-w-0 flex-[2_1_200px]">
      <SearchBar bind:value={searchQuery} placeholder={labels.searchByInvoiceProductCustomer} />
    </div>

    <Dropdown menu={false} placement="bottom-start" bind:open={dropdownOpen}>
      {#snippet trigger({ toggle })}
        <Button
          variant="secondary"
          class="flex items-center gap-2 min-w-40 h-[38px]"
          onclick={toggle}
        >
          <span class="text-sm truncate flex-1 text-left text-text-secondary">
            {selectedPaymentMethods.length > 0
              ? `${paymentMethodName(selectedPaymentMethods[0])}${selectedPaymentMethods.length > 1 ? ` +${selectedPaymentMethods.length - 1}` : ''}`
              : labels.allMethods}
          </span>
          <ChevronDown size={14} class="opacity-60 shrink-0" />
        </Button>
      {/snippet}
      {#snippet content({ close })}
        <div class="p-2 min-w-44">
          <div class="max-h-56 overflow-y-auto">
            {#each paymentMethodOptions as pm}
              <label class="flex items-center gap-2 px-2 py-1.5 rounded hover:bg-surface-hover cursor-pointer text-xs">
                <input
                  type="checkbox"
                  checked={pendingPaymentMethods.includes(pm.code)}
                  onchange={() => togglePendingPaymentMethod(pm.code)}
                  class="accent-primary"
                />
                {pm.name}
              </label>
            {/each}
          </div>
          <div class="flex items-center gap-2 px-0 pt-2 mt-2 border-t border-border">
            <button
              class="flex-1 px-3 py-1.5 text-xs font-semibold text-text-muted hover:bg-surface-hover rounded"
              onclick={() => { selectedPaymentMethods = []; pendingPaymentMethods = []; close(); }}
            >
              {labels.clear}
            </button>
            <button
              class="flex-1 px-3 py-1.5 text-xs font-semibold text-white bg-primary-default hover:bg-primary-hover rounded"
              onclick={() => { selectedPaymentMethods = pendingPaymentMethods; close(); }}
            >
              {labels.apply}
            </button>
          </div>
        </div>
      {/snippet}
    </Dropdown>

    <div class="flex items-center gap-1.5">
      <div class="flex items-center gap-1 bg-surface-default border border-border rounded-lg px-2.5 h-[38px] {amountError ? 'border-danger' : ''}">
        <span class="text-xs text-text-muted font-medium shrink-0">{labels.currencySymbol}</span>
        <input
          type="text"
          inputmode="numeric"
          value={minDisplay}
          placeholder={labels.minLabel}
          class="w-20 bg-transparent text-sm text-right text-text-primary outline-none placeholder:text-text-muted"
          oninput={handleMinInput}
          onblur={handleMinBlur}
        />
        <span class="text-text-muted text-xs shrink-0">—</span>
        <input
          type="text"
          inputmode="numeric"
          value={maxDisplay}
          placeholder={labels.maxLabel}
          class="w-20 bg-transparent text-sm text-right text-text-primary outline-none placeholder:text-text-muted"
          oninput={handleMaxInput}
          onblur={handleMaxBlur}
        />
        <button
          class="p-1 -mr-1.5 text-text-muted hover:text-danger hover:bg-danger-subtle rounded transition-colors {sliderMin !== null || sliderMax !== null ? '' : 'invisible'}"
          onclick={() => { sliderMin = null; sliderMax = null; }}
          title={labels.filterJumlah}
          aria-label={labels.filterJumlah}
        >
          <X size={14} />
        </button>
      </div>
    </div>

    <div class="relative shrink-0">
      <Button
        variant="secondary"
        class="flex items-center gap-2 min-w-40 h-[38px] date-picker-trigger"
        onclick={openDatePicker}
      >
        <CalendarDays size={16} class="text-text-secondary shrink-0" />
        <span class="text-sm font-medium truncate flex-1 text-left text-text-secondary">{dateRangeLabel}</span>
        <ChevronDown size={14} class="opacity-60 shrink-0" />
      </Button>
      {#if showDatePicker}
        <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
        <div
          role="dialog"
          tabindex="0"
          class="absolute right-0 top-full mt-1.5 z-50 bg-surface-default border border-border rounded-lg shadow-xl min-w-72 date-picker-container"
          onkeydown={handleDatePickerKeydown}
        >
          <div class="p-4 space-y-4">
            <div>
              <p class="text-xs font-medium text-text-muted uppercase tracking-wider mb-2">{labels.presetRanges}</p>
              <div class="flex flex-wrap gap-1.5">
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
            </div>

            <hr class="border-border" />

            <div>
              <p class="text-xs font-medium text-text-muted uppercase tracking-wider mb-3">{labels.customRange}</p>
              <div class="flex gap-3">
                <div class="flex-1">
                  <label for="txn-start-date" class="block text-xs text-text-secondary mb-1">{labels.startDateLabel}</label>
                  <Input id="txn-start-date" type="date" bind:value={editStartDate} class="w-full" min={currentYearStart} max={editEndDate || getTodayInJakarta()} />
                </div>
                <div class="flex-1">
                  <label for="txn-end-date" class="block text-xs text-text-secondary mb-1">{labels.endDateLabel}</label>
                  <Input id="txn-end-date" type="date" bind:value={editEndDate} class="w-full" min={editStartDate || currentYearStart} max={getTodayInJakarta()} />
                </div>
              </div>
            </div>
          </div>

          <div class="flex items-center justify-end gap-2 px-4 py-3 border-t border-border bg-surface-subtle/50 rounded-b-lg">
            <Button variant="ghost" size="sm" onclick={cancelCustomRange}>
              {labels.cancel}
            </Button>
            <Button variant="primary" size="sm" disabled={!canApplyCustom} onclick={applyCustomRange}>
              {labels.apply}
            </Button>
          </div>
        </div>
      {/if}
    </div>

    <Dropdown items={[
      { label: labels.exportCSV, icon: FileSpreadsheet, iconClass: 'text-success-light', onclick: exportCsv },
      { label: labels.exportExcel, icon: FileSpreadsheet, iconClass: 'text-info-light', onclick: exportExcel },
    ]}>
      {#snippet trigger({ toggle })}
        <Button
          variant="primary"
          class="flex items-center gap-2 h-[38px]"
          onclick={toggle}
        >
          <Download size={15} />
          {labels.export}
          <ChevronDown size={14} class="transition-transform duration-300" />
        </Button>
      {/snippet}
    </Dropdown>
  </div>
</div>
