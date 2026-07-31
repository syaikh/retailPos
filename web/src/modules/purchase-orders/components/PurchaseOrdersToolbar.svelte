<script lang="ts">
  import { SearchBar, Button, Input, Dropdown } from '$shared/ui';
  import { Plus, CalendarDays, ChevronDown } from 'lucide-svelte';
  import { getTodayInJakarta, getDateNDaysAgoInJakarta, formatJakartaDateStr } from '$shared/utils/jakartaTime';

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
    statusFilter = $bindable(''),
    startDate = $bindable(''),
    endDate = $bindable(''),
    canCreate = false,
    onsearch = () => {},
    onstatuschange = () => {},
    onstartdatechange = () => {},
    onenddatechange = () => {},
    oncreate = () => {},
  }: {
    searchQuery?: string;
    statusFilter?: string;
    startDate?: string;
    endDate?: string;
    canCreate?: boolean;
    onsearch?: () => void;
    onstatuschange?: () => void;
    onstartdatechange?: () => void;
    onenddatechange?: () => void;
    oncreate?: () => void;
  } = $props();

  let showDatePicker = $state(false);
  let editStartDate = $state('');
  let editEndDate = $state('');
  let selectedDateRange = $state('last30d');

  const statusOptions = [
    { value: 'draft', label: 'Draft' },
    { value: 'confirmed', label: 'Confirmed' },
    { value: 'partial_received', label: 'Partial Received' },
    { value: 'fully_received', label: 'Fully Received' },
    { value: 'cancelled', label: 'Cancelled' },
  ];

  const statusLabel = $derived(
    statusOptions.find(s => s.value === statusFilter)?.label || 'All Status'
  );

  const statusItems = $derived([
    { label: 'All Status', checked: statusFilter === '', onclick: () => { statusFilter = ''; onstatuschange(); } },
    ...statusOptions.map(opt => ({
      label: opt.label,
      checked: statusFilter === opt.value,
      onclick: () => { statusFilter = opt.value; onstatuschange(); },
    })),
  ]);

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

  const canApplyCustom = $derived(
    editStartDate.length > 0 && editEndDate.length > 0 && (editStartDate !== startDate || editEndDate !== endDate)
  );

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
    onstartdatechange();
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
    onstartdatechange();
    onenddatechange();
  }

  function cancelCustomRange() {
    editStartDate = '';
    editEndDate = '';
    showDatePicker = false;
  }

  function handleDatePickerKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      cancelCustomRange();
    } else if (e.key === 'Enter' && canApplyCustom) {
      applyCustomRange();
    }
  }

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
</script>

<div class="card p-3">
  <div class="flex flex-wrap items-center gap-3">
    <div class="min-w-0 flex-[2_1_200px]">
      <SearchBar bind:value={searchQuery} placeholder="Search PO number or supplier..." oninput={onsearch} inputClass="h-10" />
    </div>
    <Dropdown placement="bottom-start" items={statusItems}>
      {#snippet trigger({ toggle })}
        <button
          type="button"
          class="flex items-center gap-2 px-3 h-10 rounded-xl border transition-all duration-200 text-[13px] font-medium whitespace-nowrap {statusFilter !== '' ? 'bg-primary/10 border-primary/30 text-primary-light' : 'bg-surface-default border-border-strong text-text-muted hover:text-text-secondary hover:border-border-strong'}"
          onclick={toggle}
        >
          <span>{statusLabel}</span>
          <ChevronDown size={14} class="text-text-muted shrink-0" />
        </button>
      {/snippet}
    </Dropdown>
    <div class="relative shrink-0">
      <Button
        variant="secondary"
        class="flex items-center gap-2 min-w-40 h-[38px] date-picker-trigger"
        onclick={openDatePicker}
      >
        <CalendarDays size={16} class="text-white shrink-0" />
        <span class="text-sm font-medium truncate flex-1 text-left text-text-secondary">{dateRangeLabel}</span>
        <ChevronDown size={14} class="opacity-60 shrink-0" />
      </Button>
      {#if showDatePicker}
        <div
          role="dialog"
          tabindex="0"
          class="absolute right-0 top-full mt-1.5 z-50 bg-surface-default border border-border rounded-lg shadow-xl min-w-72 date-picker-container"
          onkeydown={handleDatePickerKeydown}
        >
          <div class="p-4 space-y-4">
            <div>
              <p class="text-xs font-medium text-text-muted uppercase tracking-wider mb-2">Preset Ranges</p>
              <div class="flex flex-wrap gap-1.5">
                {#each datePresets as preset}
                  <Button variant="ghost" size="xs" onclick={() => applyDatePreset(preset.days)}>
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
                  <label for="po-start-date" class="block text-xs text-text-secondary mb-1">Start Date</label>
                  <Input id="po-start-date" type="date" bind:value={editStartDate} class="w-full" min={currentYearStart} max={editEndDate || getTodayInJakarta()} />
                </div>
                <div class="flex-1">
                  <label for="po-end-date" class="block text-xs text-text-secondary mb-1">End Date</label>
                  <Input id="po-end-date" type="date" bind:value={editEndDate} class="w-full" min={editStartDate || currentYearStart} max={getTodayInJakarta()} />
                </div>
              </div>
            </div>
          </div>
          <div class="flex items-center justify-end gap-2 px-4 py-3 border-t border-border bg-surface-subtle/50 rounded-b-lg">
            <Button variant="ghost" size="sm" onclick={cancelCustomRange}>Cancel</Button>
            <Button variant="primary" size="sm" disabled={!canApplyCustom} onclick={applyCustomRange}>Apply</Button>
          </div>
        </div>
      {/if}
    </div>
    {#if canCreate}
      <Button variant="primary" class="shrink-0 shadow-glow-primary-sm" onclick={oncreate}>
        <Plus size={18} /> Create PO
      </Button>
    {/if}
  </div>
</div>
