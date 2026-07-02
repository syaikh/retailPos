<script lang="ts">
  import { Button, Input, SearchBar, Dropdown } from '$shared/ui';
  import { Search, RefreshCw, X, Download, FileSpreadsheet, CalendarDays, ChevronDown, List, Tag } from 'lucide-svelte';
  import { getTodayInJakarta, getDateNDaysAgoInJakarta, formatJakartaDateStr, JAKARTA_OFFSET_MS } from '$shared/utils/jakartaTime';
  import { getAuthToken } from '$modules/auth';
  import { toast } from '$shared/stores/toast.svelte';

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
  };

  const ALL_ACTIONS = [
    { id: 'all', label: 'All Actions' },
    { id: 'create', label: 'Create' },
    { id: 'update', label: 'Update' },
    { id: 'delete', label: 'Delete' },
    { id: 'login', label: 'Login' },
    { id: 'logout', label: 'Logout' },
  ];

  let availableActionFilters = $derived(
    selectedResource === 'all' ? ALL_ACTIONS : (actionsMap[selectedResource] || ['all']).map((id) => ALL_ACTIONS.find((a) => a.id === id) || { id, label: id })
  );

  const actionFilters = [
    { id: 'all', label: 'All' },
    { id: 'create', label: 'Create' },
    { id: 'update', label: 'Update' },
    { id: 'delete', label: 'Delete' },
    { id: 'login', label: 'Login' },
    { id: 'logout', label: 'Logout' },
  ];

  function isActionDisabled(actionId: string): boolean {
    if (selectedResource === 'all') return false;
    if (actionId === 'all') return false;
    const relevant = actionsMap[selectedResource];
    if (!relevant) return true;
    return !relevant.includes(actionId);
  }

  const resourceFilters = [
    { id: 'all', label: 'All Resources' },
    { id: 'auth', label: 'Auth' },
    { id: 'user', label: 'User' },
    { id: 'role', label: 'Role' },
    { id: 'product', label: 'Product' },
    { id: 'sale', label: 'Sale' },
    { id: 'category', label: 'Category' },
    { id: 'brand', label: 'Brand' },
  ];

  const dateRanges = [
    { id: '24h', label: 'Last 24 Hours' },
    { id: '7d', label: 'Last 7 Days' },
    { id: '30d', label: 'Last 30 Days' },
    { id: '90d', label: 'Last 90 Days' },
    { id: 'custom', label: 'Custom Range' },
  ];

  const datePresets = [
    { label: 'Last 24 Hours', rangeId: '24h' },
    { label: 'Last 7 Days', rangeId: '7d' },
    { label: 'Last 30 Days', rangeId: '30d' },
    { label: 'Last 90 Days', rangeId: '90d' },
  ];

  const dateRangeLabel = $derived.by(() => {
    if (selectedDateRange === 'custom') {
      return `${formatJakartaDateStr(customStartDate)} – ${formatJakartaDateStr(customEndDate)}`;
    }
    return dateRanges.find(d => d.id === selectedDateRange)?.label || 'Last 24 Hours';
  });

  let resourceLabel = $derived(resourceFilters.find(f => f.id === selectedResource)?.label || 'All');
  let actionLabel = $derived(availableActionFilters.find(f => f.id === selectedAction)?.label || 'All Actions');

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

  function jakartaDateToUTC(dateStr) {
    const [y, m, d] = dateStr.split('-').map(Number);
    return Date.UTC(y, m - 1, d, 0, 0, 0, 0) - JAKARTA_OFFSET_MS;
  }

  function getJakartaMidnightMs(dateStr: string): number {
    const [y, m, d] = dateStr.split('-').map(Number);
    return Date.UTC(y, m - 1, d, 0, 0, 0, 0) - JAKARTA_OFFSET_MS;
  }

  function getDateRange(range) {
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

  function clearFilter(type) {
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
      });
    }
    if (selectedAction !== 'all') {
      filters.push({
        type: 'action',
        label: actionFilters.find((f) => f.id === selectedAction)?.label || selectedAction,
      });
    }
    if (selectedDateRange !== '24h') {
      if (selectedDateRange === 'custom') {
        filters.push({ type: 'date', label: `${formatJakartaDateStr(customStartDate)} – ${formatJakartaDateStr(customEndDate)}` });
      } else {
        const dr = dateRanges.find((d) => d.id === selectedDateRange);
        if (dr) filters.push({ type: 'date', label: dr.label });
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
    if (!token) { toast.error('Session expired'); return; }

    const res = await fetch(buildExportUrl(format), {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (!res.ok) { toast.error('Export failed'); return; }

    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `audit-logs-${today}.${format === 'csv' ? 'csv' : 'xlsx'}`;
    a.click();
    URL.revokeObjectURL(url);
    toast.success(`Audit logs exported to ${format.toUpperCase()}`);
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
      <SearchBar bind:value={searchQuery} placeholder="Search by actor, role, action, entity, or IP..." inputClass="h-10" />
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
                    onclick={() => applyDatePreset(preset.rangeId)}
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
                  <label for="audit-start-date" class="block text-xs text-text-secondary mb-1">Start Date</label>
                  <Input id="audit-start-date" type="date" bind:value={editStartDate} class="w-full" min={ninetyDaysAgo} max={editEndDate || today} />
                </div>
                <div class="flex-1">
                  <label for="audit-end-date" class="block text-xs text-text-secondary mb-1">End Date</label>
                  <Input id="audit-end-date" type="date" bind:value={editEndDate} class="w-full" min={editStartDate || ninetyDaysAgo} max={today} />
                </div>
              </div>
            </div>
          </div>

          <div class="flex items-center justify-end gap-2 px-4 py-3 border-t border-border bg-surface-subtle/50 rounded-b-lg">
            <Button variant="ghost" size="sm" onclick={cancelCustomDateRange}>
              Cancel
            </Button>
            <Button variant="primary" size="sm" disabled={!canApplyCustom} onclick={applyCustomDateRange}>
              Apply
            </Button>
          </div>
        </div>
      {/if}
    </div>

    <Dropdown placement="bottom-start">
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
        {#each resourceFilters as f}
          <button
            class="w-full text-left px-4 py-2 text-sm transition-colors {selectedResource === f.id ? 'text-primary-light bg-primary-subtle/30 font-medium' : 'text-text-secondary hover:bg-surface-hover'}"
            onclick={() => { selectedResource = f.id; selectedAction = 'all'; close(); }}
          >
            {f.label}
          </button>
        {/each}
      {/snippet}
    </Dropdown>

    <Dropdown placement="bottom-start">
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
            class="w-full text-left px-4 py-2 text-sm transition-colors {selectedAction === f.id ? 'text-primary-light bg-primary-subtle/30 font-medium' : 'text-text-secondary hover:bg-surface-hover'}"
            onclick={() => { selectedAction = f.id; close(); }}
          >
            {f.label}
          </button>
        {/each}
      {/snippet}
    </Dropdown>

    <Button title="Refresh" variant="secondary" class="px-3 h-10" onclick={onrefresh}>
      <RefreshCw size={16} class={loading ? 'animate-spin' : ''} />
    </Button>
    <Dropdown items={[
      { label: 'Export to CSV', icon: FileSpreadsheet, iconClass: 'text-success-light', onclick: exportToCsv },
      { label: 'Export to Excel', icon: FileSpreadsheet, iconClass: 'text-info-light', onclick: exportToExcel },
    ]}>
      {#snippet trigger({ toggle })}
        <Button
          variant="primary"
          class="flex items-center gap-2 transition-all duration-300 h-10"
          onclick={toggle}
        >
          <Download size={15} />
          Export
          <ChevronDown size={14} class="transition-transform duration-300" />
        </Button>
      {/snippet}
    </Dropdown>
  </div>

  <div class="filter-chips-wrapper" class:is-open={activeFilters.length > 0}>
    <div class="filter-chips-inner">
      <div class="flex flex-wrap items-center gap-2 pt-3 mt-3 border-t border-border/50">
        {#each activeFilters as filter}
          {@const FilterIcon = getFilterIcon(filter.type)}
          <div class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-primary-subtle/20 border border-primary-subtle/30 rounded-full text-sm text-text-secondary">
            <FilterIcon size={13} class="text-primary-light shrink-0" />
            <span class="font-medium truncate max-w-[180px]">{filter.label}</span>
            <button
              class="w-4 h-4 rounded-full flex items-center justify-center text-text-muted hover:text-text-primary hover:bg-surface-hover transition-colors"
              title={`Clear ${filter.label}`}
              onclick={() => clearFilter(filter.type)}
              aria-label={`Clear ${filter.label} filter`}
            >
              <X size={12} />
            </button>
          </div>
        {/each}
        <button
          class="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm font-semibold text-text-primary bg-primary-subtle/40 border border-primary-subtle/50 rounded-full transition-colors hover:bg-primary-subtle/60"
          onclick={clearAllFilters}
        >
          Clear all
          <X size={12} />
        </button>
      </div>
    </div>
  </div>
</div>

<style>
  .filter-chips-wrapper {
    display: grid;
    grid-template-rows: 0fr;
    opacity: 0;
    transition: grid-template-rows 0.2s ease-out, opacity 0.2s ease-out;
  }

  .filter-chips-wrapper.is-open {
    grid-template-rows: 1fr;
    opacity: 1;
  }

  .filter-chips-inner {
    overflow: hidden;
  }
</style>
