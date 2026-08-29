<script lang="ts">
  import { Button, Input, SearchBar, Dropdown, FilterChipBar } from '$shared/ui';
  import { Search, RefreshCw, Download, FileSpreadsheet, CalendarDays, ChevronDown, List, Tag } from 'lucide-svelte';
  import { getTodayInJakarta, getDateNDaysAgoInJakarta, formatJakartaDateStr, JAKARTA_OFFSET_MS } from '$shared/utils/jakartaTime';
  import { getAuthToken } from '$modules/auth';
  import { toast } from '$shared/stores/toast.svelte';
  import { onMount } from 'svelte';
  import { labels, t } from '$shared/i18n';

  let entityTypes = $state<string[]>([]);

  onMount(async () => {
    try {
      const token = getAuthToken();
      if (!token) return;
      const res = await fetch('/api/audit-logs/entity-types', {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const json = await res.json();
        entityTypes = json.data || [];
      } else {
        toast.error(labels.failedToLoad);
      }
    } catch (e) {
      toast.error(labels.failedToLoad);
    }
  });

  let {
    searchQuery = $bindable(''),
    selectedAction = $bindable('all'),
    selectedResource = $bindable('all'),
    selectedDateRange = $bindable('24h'),
    showDatePicker = $bindable(false),
    customStartDate = $bindable(getDateNDaysAgoInJakarta(1)),
    customEndDate = $bindable(getTodayInJakarta()),
    loading = false,
    onrefresh = () => {},
    onexportcsv = () => {},
    onexportxlsx = () => {},
  }: {
    searchQuery?: string;
    selectedAction?: string;
    selectedResource?: string;
    selectedDateRange?: string;
    showDatePicker?: boolean;
    customStartDate?: string;
    customEndDate?: string;
    loading?: boolean;
    onrefresh?: () => void;
    onexportcsv?: () => void;
    onexportxlsx?: () => void;
  } = $props();

  const actionsMap: Record<string, string[]> = {
    all: ['all', 'create', 'update', 'delete', 'login', 'logout'],
    auth: ['all', 'login', 'logout'],
    user: ['all', 'create', 'update', 'delete'],
    role: ['all', 'create', 'update', 'delete'],
    product: ['all', 'create', 'update', 'delete'],
    sale: ['all', 'create', 'update', 'delete'],
    category: ['all', 'create', 'update', 'delete'],
    brand: ['all', 'create', 'update', 'delete'],
    uom: ['all', 'create', 'update', 'delete'],
    customer: ['all', 'create', 'update', 'delete'],
    stock: ['all', 'update'],
    shift: ['all', 'shift_opened', 'shift_closed', 'shift_close_all'],
    payment: ['all', 'payment.created'],
  };

  function getResourceActions(resource: string): string[] {
    return actionsMap[resource] || ['all', 'create', 'update', 'delete'];
  }

  const ALL_ACTIONS = [
    { id: 'all', label: `${labels.all} ${labels.action}` },
    { id: 'create', label: labels.create },
    { id: 'update', label: labels.update },
    { id: 'delete', label: labels.delete },
    { id: 'login', label: labels.login },
    { id: 'logout', label: labels.logout },
    { id: 'shift_opened', label: 'Shift Opened' },
    { id: 'shift_closed', label: 'Shift Closed' },
    { id: 'shift_close_all', label: 'Shift Close All' },
    { id: 'payment.created', label: 'Payment Created' },
  ];

  let availableActionFilters = $derived(
    selectedResource === 'all' ? ALL_ACTIONS : getResourceActions(selectedResource).map((id) => ALL_ACTIONS.find((a) => a.id === id) || { id, label: id })
  );

  const actionFilters = [
    { id: 'all', label: labels.all },
    { id: 'create', label: labels.create },
    { id: 'update', label: labels.update },
    { id: 'delete', label: labels.delete },
    { id: 'login', label: labels.login },
    { id: 'logout', label: labels.logout },
    { id: 'shift_opened', label: 'Shift Opened' },
    { id: 'shift_closed', label: 'Shift Closed' },
    { id: 'shift_close_all', label: 'Shift Close All' },
    { id: 'payment.created', label: 'Payment Created' },
  ];

  function isActionDisabled(actionId: string): boolean {
    if (selectedResource === 'all') return false;
    if (actionId === 'all') return false;
    const relevant = getResourceActions(selectedResource);
    return !relevant.includes(actionId);
  }

  const resourceLabels: Record<string, string> = {
    auth: 'Auth',
    user: labels.user,
    role: labels.role,
    product: labels.product,
    sale: 'Sale',
    category: labels.category,
    brand: labels.brand,
    stock: labels.stock,
    uom: 'UOM',
    customer: labels.customer,
  };

  const resourceFilters = $derived(
    entityTypes.length > 0
      ? [{ id: 'all', label: `${labels.all} ${labels.resource}` }, ...entityTypes.map((t) => ({ id: t, label: resourceLabels[t] || t.charAt(0).toUpperCase() + t.slice(1) }))]
      : [{ id: 'all', label: `${labels.all} ${labels.resource}` }]
  );

  const dateRanges = [
    { id: '24h', label: labels.last24Hours },
    { id: '7d', label: labels.last7Days },
    { id: '30d', label: labels.last30Days },
    { id: '90d', label: labels.last90Days },
    { id: 'custom', label: labels.customRange },
  ];

  const datePresets = [
    { label: labels.last24Hours, rangeId: '24h' },
    { label: labels.last7Days, rangeId: '7d' },
    { label: labels.last30Days, rangeId: '30d' },
    { label: labels.last90Days, rangeId: '90d' },
  ];

  const dateRangeLabel = $derived.by(() => {
    if (selectedDateRange === 'custom') {
      return `${formatJakartaDateStr(customStartDate)} – ${formatJakartaDateStr(customEndDate)}`;
    }
    return dateRanges.find(d => d.id === selectedDateRange)?.label || labels.last24Hours;
  });

  let resourceLabel = $derived(resourceFilters.find(f => f.id === selectedResource)?.label || labels.all);
  let actionLabel = $derived(availableActionFilters.find(f => f.id === selectedAction)?.label || `${labels.all} ${labels.action}`);

  const today = getTodayInJakarta();
  const ninetyDaysAgo = getDateNDaysAgoInJakarta(90);

  let editStartDate = $state('');
  let editEndDate = $state('');

  const canApplyCustom = $derived(
    editStartDate.length > 0 && editEndDate.length > 0 && (editStartDate !== customStartDate || editEndDate !== customEndDate)
  );

  function toggleDatePicker() {
    if (showDatePicker) {
      cancelCustomDateRange();
    } else {
      editStartDate = customStartDate;
      editEndDate = customEndDate;
      showDatePicker = true;
    }
  }

  function applyCustomDateRange() {
    if (!canApplyCustom) return;
    customStartDate = editStartDate;
    customEndDate = editEndDate;
    selectedDateRange = 'custom';
    showDatePicker = false;
  }

  function cancelCustomDateRange() {
    editStartDate = '';
    editEndDate = '';
    showDatePicker = false;
  }

  function handleDatePickerKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      cancelCustomDateRange();
    } else if (e.key === 'Enter' && canApplyCustom) {
      applyCustomDateRange();
    }
  }

  function jakartaDateToUTC(dateStr: string) {
    const [y, m, d] = dateStr.split('-').map(Number);
    return Date.UTC(y, m - 1, d, 0, 0, 0, 0) - JAKARTA_OFFSET_MS;
  }

  function getJakartaMidnightMs(dateStr: string): number {
    const [y, m, d] = dateStr.split('-').map(Number);
    return Date.UTC(y, m - 1, d, 0, 0, 0, 0) - JAKARTA_OFFSET_MS;
  }

  function getDateRange(range: string) {
    switch (range) {
      case '24h': {
        const yesterday = getDateNDaysAgoInJakarta(1);
        const todayJakarta = getTodayInJakarta();
        return {
          start: new Date(getJakartaMidnightMs(yesterday)),
          end: new Date(getJakartaMidnightMs(todayJakarta) + 86400000),
        };
      }
      case '7d': {
        const sevenDaysAgo = getDateNDaysAgoInJakarta(7);
        const todayJakarta = getTodayInJakarta();
        return {
          start: new Date(getJakartaMidnightMs(sevenDaysAgo)),
          end: new Date(getJakartaMidnightMs(todayJakarta) + 86400000),
        };
      }
      case '30d': {
        const thirtyDaysAgo = getDateNDaysAgoInJakarta(30);
        const todayJakarta = getTodayInJakarta();
        return {
          start: new Date(getJakartaMidnightMs(thirtyDaysAgo)),
          end: new Date(getJakartaMidnightMs(todayJakarta) + 86400000),
        };
      }
      case '90d': {
        const ninetyDaysAgo = getDateNDaysAgoInJakarta(90);
        const todayJakarta = getTodayInJakarta();
        return {
          start: new Date(getJakartaMidnightMs(ninetyDaysAgo)),
          end: new Date(getJakartaMidnightMs(todayJakarta) + 86400000),
        };
      }
      case 'custom':
        if (customStartDate && customEndDate) {
          const startMs = jakartaDateToUTC(customStartDate);
          const endMs = jakartaDateToUTC(customEndDate) + 86400000;
          return { start: new Date(startMs), end: new Date(endMs) };
        }
        const fallbackStart = getDateNDaysAgoInJakarta(7);
        const fallbackEnd = getTodayInJakarta();
        return {
          start: new Date(getJakartaMidnightMs(fallbackStart)),
          end: new Date(getJakartaMidnightMs(fallbackEnd) + 86400000),
        };
      default: {
        const defaultStart = getDateNDaysAgoInJakarta(7);
        const defaultEnd = getTodayInJakarta();
        return {
          start: new Date(getJakartaMidnightMs(defaultStart)),
          end: new Date(getJakartaMidnightMs(defaultEnd) + 86400000),
        };
      }
    }
  }

  function applyDatePreset(rangeId: string) {
    selectedDateRange = rangeId;
    showDatePicker = false;
    if (rangeId !== 'custom') {
      const today = getTodayInJakarta();
      switch (rangeId) {
        case '24h':
          customStartDate = getDateNDaysAgoInJakarta(1);
          customEndDate = today;
          break;
        case '7d':
          customStartDate = getDateNDaysAgoInJakarta(7);
          customEndDate = today;
          break;
        case '30d':
          customStartDate = getDateNDaysAgoInJakarta(30);
          customEndDate = today;
          break;
        case '90d':
          customStartDate = getDateNDaysAgoInJakarta(90);
          customEndDate = today;
          break;
      }
    }
  }

  function clearFilter(type: string) {
    if (type === 'entity') selectedResource = 'all';
    if (type === 'action') selectedAction = 'all';
    if (type === 'date') {
      selectedDateRange = '24h';
      customStartDate = getDateNDaysAgoInJakarta(1);
      customEndDate = getTodayInJakarta();
    }
  }

  function clearAllFilters() {
    selectedResource = 'all';
    selectedAction = 'all';
    selectedDateRange = '24h';
    customStartDate = getDateNDaysAgoInJakarta(1);
    customEndDate = getTodayInJakarta();
    searchQuery = '';
  }

  function getFilterIcon(type: string) {
    if (type === 'entity') return Tag;
    if (type === 'action') return List;
    if (type === 'date') return CalendarDays;
    if (type === 'search') return Search;
    return List;
  }

  let activeFilters = $derived.by(() => {
    const filters = [];
    if (selectedResource !== 'all') {
      filters.push({
        type: 'entity',
        label: resourceFilters.find((f) => f.id === selectedResource)?.label || selectedResource,
        icon: getFilterIcon('entity'),
      });
    }
    if (selectedAction !== 'all') {
      filters.push({
        type: 'action',
        label: actionFilters.find((f) => f.id === selectedAction)?.label || selectedAction,
        icon: getFilterIcon('action'),
      });
    }
    if (selectedDateRange !== '24h' && selectedDateRange !== '7d') {
      if (selectedDateRange === 'custom') {
        filters.push({ type: 'date', label: `${formatJakartaDateStr(customStartDate)} – ${formatJakartaDateStr(customEndDate)}`, icon: getFilterIcon('date') });
      } else {
        const dr = dateRanges.find((d) => d.id === selectedDateRange);
        if (dr) filters.push({ type: 'date', label: dr.label, icon: getFilterIcon('date') });
      }
    }
    return filters;
  });

  function buildExportUrl(format: string): string {
    const range = getDateRange(selectedDateRange);
    const params = new URLSearchParams({
      format,
      search: searchQuery,
      start_date: range.start.toISOString(),
      end_date: range.end.toISOString(),
    });
    if (selectedAction !== 'all') params.append('action', selectedAction);
    if (selectedResource !== 'all') params.append('entity_type', selectedResource);
    return `/api/audit-logs/export?${params.toString()}`;
  }

  async function downloadExport(format: string) {
    const token = getAuthToken();
    if (!token) { toast.error(labels.errorOccurred); return; }

    const res = await fetch(buildExportUrl(format), {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (!res.ok) { toast.error(labels.errorOccurred); return; }

    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `audit-logs-${today}.${format === 'csv' ? 'csv' : 'xlsx'}`;
    a.click();
    URL.revokeObjectURL(url);
    toast.success(`${labels.auditLogs} ${labels.export}: ${format.toUpperCase()}`);
  }

  function exportToCsv() {
    downloadExport('csv');
    onexportcsv();
  }

  function exportToExcel() {
    downloadExport('xlsx');
    onexportxlsx();
  }
</script>

<div class="card p-4">
  <div class="flex items-center gap-3">
    <div class="relative flex-1">
      <SearchBar bind:value={searchQuery} placeholder={labels.searchByActor} inputClass="h-10" />
    </div>

    <div class="relative shrink-0 date-picker-container">
      <Button
        variant="secondary"
        class="date-picker-trigger flex items-center gap-2 min-w-44"
        onclick={toggleDatePicker}
      >
        <CalendarDays size={16} class="text-text-secondary shrink-0" />
        <span class="text-sm font-medium truncate flex-1 text-left text-text-secondary">{dateRangeLabel}</span>
        <ChevronDown size={14} class="opacity-60 shrink-0" />
      </Button>
      {#if showDatePicker}
        <div
          role="dialog"
          aria-label={labels.dateRangePicker}
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
                    onclick={() => applyDatePreset(preset.rangeId)}
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
                  <label for="audit-start-date" class="block text-xs text-text-secondary mb-1">{labels.startDateLabel}</label>
                  <Input id="audit-start-date" type="date" bind:value={editStartDate} class="w-full" min={ninetyDaysAgo} max={editEndDate || today} />
                </div>
                <div class="flex-1">
                  <label for="audit-end-date" class="block text-xs text-text-secondary mb-1">{labels.endDateLabel}</label>
                  <Input id="audit-end-date" type="date" bind:value={editEndDate} class="w-full" min={editStartDate || ninetyDaysAgo} max={today} />
                </div>
              </div>
            </div>
          </div>

          <div class="flex items-center justify-end gap-2 px-4 py-3 border-t border-border bg-surface-subtle/50 rounded-b-lg">
            <Button variant="ghost" size="sm" onclick={cancelCustomDateRange}>
              {labels.cancel}
            </Button>
            <Button variant="primary" size="sm" disabled={!canApplyCustom} onclick={applyCustomDateRange}>
              {labels.apply}
            </Button>
          </div>
        </div>
      {/if}
    </div>

    <Dropdown placement="bottom-start" menuClass="min-w-[320px]">
      {#snippet trigger({ toggle })}
        <button
          class="flex items-center gap-2 px-3 h-10 w-32 rounded-xl border border-border bg-surface-default text-text-secondary text-sm hover:border-border-strong hover:bg-surface-hover transition-colors"
          onclick={toggle}
        >
          <span class="flex-1 text-left truncate">{resourceLabel}</span>
          <ChevronDown size={14} class="text-text-muted shrink-0" />
        </button>
      {/snippet}
      {#snippet content({ close })}
        <div class="grid grid-cols-2 gap-1">
          {#each resourceFilters as f}
            <button
              class="w-full text-left px-3 py-1.5 text-sm transition-colors {selectedResource === f.id ? 'text-primary-light bg-primary-subtle/30 font-medium' : 'text-text-secondary hover:bg-surface-hover'}"
              onclick={() => { selectedResource = f.id; selectedAction = 'all'; close(); }}
            >
              {f.label}
            </button>
          {/each}
        </div>
      {/snippet}
    </Dropdown>

    <Dropdown placement="bottom-start" menuClass="w-[140px] min-w-0">
      {#snippet trigger({ toggle })}
        <button
          class="flex items-center gap-2 px-3 h-10 w-[140px] rounded-xl border border-border bg-surface-default text-text-secondary text-sm hover:border-border-strong hover:bg-surface-hover transition-colors"
          onclick={toggle}
        >
          <span class="flex-1 text-left truncate">{actionLabel}</span>
          <ChevronDown size={14} class="text-text-muted shrink-0" />
        </button>
      {/snippet}
      {#snippet content({ close })}
        {#each availableActionFilters as f}
          <button
            class="w-full text-left px-3 py-1.5 text-sm transition-colors {selectedAction === f.id ? 'text-primary-light bg-primary-subtle/30 font-medium' : 'text-text-secondary hover:bg-surface-hover'}"
            onclick={() => { selectedAction = f.id; close(); }}
          >
            {f.label}
          </button>
        {/each}
      {/snippet}
    </Dropdown>

    <Button title={labels.refresh} variant="secondary" class="px-3 h-10" onclick={onrefresh}>
      <RefreshCw size={16} class={loading ? 'animate-spin' : ''} />
    </Button>
    <Dropdown items={[
      { label: labels.exportCSV, icon: FileSpreadsheet, iconClass: 'text-success-light', onclick: exportToCsv },
      { label: labels.exportExcel, icon: FileSpreadsheet, iconClass: 'text-info-light', onclick: exportToExcel },
    ]}>
      {#snippet trigger({ toggle })}
        <Button
          variant="primary"
          class="flex items-center gap-2 transition-all duration-300 h-10"
          onclick={toggle}
        >
          <Download size={15} />
          {labels.export}
          <ChevronDown size={14} class="transition-transform duration-300" />
        </Button>
      {/snippet}
    </Dropdown>
  </div>

  <FilterChipBar chips={activeFilters} onclear={clearFilter} onclearall={clearAllFilters} clearLabel={labels.clearAll} />
</div>
