<script lang="ts">
  import { Button, Input, SearchBar, Dropdown } from '$shared/ui';
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
    sliderMin = $bindable<number | null>(null),
    sliderMax = $bindable<number | null>(null),
    selectedDateRange = $bindable('last30d'),
    paymentMethodOptions = [] as { code: string; name: string }[],
    onexportcsv = () => {},
    onexportxlsx = () => {},
  } = $props();

  let editStartDate = $state('');
  let editEndDate = $state('');

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

  const canApplyCustom = $derived(
    editStartDate.length > 0 && editEndDate.length > 0 && (editStartDate !== startDate || editEndDate !== endDate)
  );

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

  function togglePaymentMethod(code: string) {
    if (selectedPaymentMethods.includes(code)) {
      selectedPaymentMethods = selectedPaymentMethods.filter(c => c !== code);
    } else {
      selectedPaymentMethods = [...selectedPaymentMethods, code];
    }
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
    if (selectedPaymentMethods.length > 0) {
      params.set('paymentMethods', selectedPaymentMethods.join(','));
    }
    if (sliderMin !== null && sliderMin > 0) {
      params.set('minTotal', sliderMin.toString());
    }
    if (sliderMax !== null && sliderMax < SLIDER_MAX_BOUND) {
      params.set('maxTotal', sliderMax.toString());
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
      <SearchBar bind:value={searchQuery} placeholder="Search by invoice, product, or customer..." />
    </div>

    <Dropdown menu={false} placement="bottom-start">
      {#snippet trigger({ toggle })}
        <Button
          variant="secondary"
          class="flex items-center gap-2 min-w-40 h-[38px]"
          onclick={toggle}
        >
          <span class="text-sm truncate flex-1 text-left text-text-secondary">
            {selectedPaymentMethods.length > 0
              ? `${paymentMethodName(selectedPaymentMethods[0])}${selectedPaymentMethods.length > 1 ? ` +${selectedPaymentMethods.length - 1}` : ''}`
              : 'All methods'}
          </span>
          <ChevronDown size={14} class="opacity-60 shrink-0" />
        </Button>
      {/snippet}
      {#snippet content({ close })}
        <div class="p-2 min-w-44 max-h-56 overflow-y-auto">
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
              onclick={() => { selectedPaymentMethods = []; close(); }}
            >
              Clear selection
            </button>
          {/if}
        </div>
      {/snippet}
    </Dropdown>

    <div class="flex items-center gap-1.5">
      <div class="flex items-center gap-1 bg-surface-default border border-border rounded-lg px-2.5 h-[38px] {amountError ? 'border-danger' : ''}">
        <span class="text-xs text-text-muted font-medium shrink-0">Rp</span>
        <input
          type="text"
          inputmode="numeric"
          value={minDisplay}
          placeholder="Min"
          class="w-20 bg-transparent text-sm text-right text-text-primary outline-none placeholder:text-text-muted"
          oninput={handleMinInput}
          onblur={handleMinInput}
        />
        <span class="text-text-muted text-xs shrink-0">—</span>
        <input
          type="text"
          inputmode="numeric"
          value={maxDisplay}
          placeholder="Max"
          class="w-20 bg-transparent text-sm text-right text-text-primary outline-none placeholder:text-text-muted"
          oninput={handleMaxInput}
          onblur={handleMaxInput}
        />
      </div>
      {#if amountError}
        <span class="text-xs text-danger whitespace-nowrap">{amountError}</span>
      {/if}
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
        <div
          role="dialog"
          class="absolute right-0 top-full mt-1.5 z-50 bg-surface-default border border-border rounded-lg shadow-xl min-w-72 date-picker-container"
          onkeydown={handleDatePickerKeydown}
        >
          <div class="p-4 space-y-4">
            <div>
              <p class="text-xs font-medium text-text-muted uppercase tracking-wider mb-2">Preset Ranges</p>
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
              <p class="text-xs font-medium text-text-muted uppercase tracking-wider mb-3">Custom Range</p>
              <div class="flex gap-3">
                <div class="flex-1">
                  <label for="txn-start-date" class="block text-xs text-text-secondary mb-1">Start Date</label>
                  <Input id="txn-start-date" type="date" bind:value={editStartDate} class="w-full" min={currentYearStart} max={editEndDate || getTodayInJakarta()} />
                </div>
                <div class="flex-1">
                  <label for="txn-end-date" class="block text-xs text-text-secondary mb-1">End Date</label>
                  <Input id="txn-end-date" type="date" bind:value={editEndDate} class="w-full" min={editStartDate || currentYearStart} max={getTodayInJakarta()} />
                </div>
              </div>
            </div>
          </div>

          <div class="flex items-center justify-end gap-2 px-4 py-3 border-t border-border bg-surface-subtle/50 rounded-b-lg">
            <Button variant="ghost" size="sm" onclick={cancelCustomRange}>
              Cancel
            </Button>
            <Button variant="primary" size="sm" disabled={!canApplyCustom} onclick={applyCustomRange}>
              Apply
            </Button>
          </div>
        </div>
      {/if}
    </div>

    <Dropdown items={[
      { label: 'Export to CSV', icon: FileSpreadsheet, iconClass: 'text-success-light', onclick: exportCsv },
      { label: 'Export to Excel', icon: FileSpreadsheet, iconClass: 'text-info-light', onclick: exportExcel },
    ]}>
      {#snippet trigger({ toggle })}
        <Button
          variant="primary"
          class="flex items-center gap-2 h-[38px]"
          onclick={toggle}
        >
          <Download size={15} />
          Export
          <ChevronDown size={14} class="transition-transform duration-300" />
        </Button>
      {/snippet}
    </Dropdown>
  </div>
</div>
