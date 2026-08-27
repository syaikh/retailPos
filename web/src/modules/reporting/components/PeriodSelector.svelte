<script>
  import { onMount } from 'svelte';
  import { fly } from 'svelte/transition';
  import { Button, Dropdown } from '$shared/ui';
  import { SelectableCalendar, MonthlyCalendar, YearCalendar } from '$modules/reporting/components/calendar';
  import { CalendarDate } from '@internationalized/date';
  import {
    BarChart3, CalendarDays, ChevronDown, Download, FileSpreadsheet, Clock
  } from 'lucide-svelte';
  import { formatDate, getFirstOfMonthNAgoInJakarta } from '$modules/reporting/lib/reporting-utils';
  import { labels } from '$shared/i18n';
  import { getTodayInJakarta, getDateNDaysAgoInJakarta, getJakartaDayOfWeek } from '$shared/utils/jakartaTime';

  let {
    selectedPeriodType = $bindable('realtime'),
    activePeriodType = $bindable('realtime'),
    dropdownOpen = $bindable(false),
    hoveredOption = $bindable(null),
    selectedDailyDate = $bindable(null),
    selectedWeeklyRange = $bindable(null),
    selectedMonthlyRange = $bindable(null),
    selectedYearlyRange = $bindable(null),
    dailySelectionMade = $bindable(false),
    weeklySelectionMade = $bindable(false),
    monthlySelectionMade = $bindable(false),
    yearlySelectionMade = $bindable(false),
    availableYears = [],
    currentTimeHour = '00:00',
    timezoneString = 'GMT+07',
    onfetchsaleswithrange = (start, end) => {},
    onexportexcel = () => {},
    onexportpdf = () => {},
  } = $props();

  let yesterdayDate = $derived(
    new CalendarDate(
      parseInt(getDateNDaysAgoInJakarta(1).split('-')[0]),
      parseInt(getDateNDaysAgoInJakarta(1).split('-')[1]),
      parseInt(getDateNDaysAgoInJakarta(1).split('-')[2])
    )
  );

  let selectedYear = $state(parseInt(getTodayInJakarta().split('-')[0]));

  const calendarTheme = {
    bg: 'transparent',
    text: 'var(--color-text-primary)',
    muted: 'var(--color-text-muted)',
    border: 'var(--color-border-strong)',
    hover: 'var(--color-primary-subtle)',
    selected: 'var(--color-primary)',
    selectedText: 'var(--color-text-inverted)',
    todayBorder: 'var(--color-info-default)',
    radius: '8px'
  };

  const periodOptions = $derived([
    { value: 'realtime', label: labels.periodRealtime, icon: Clock, description: labels.descRealtime },
    { value: 'yesterday', label: labels.yesterday, icon: CalendarDays, description: labels.descYesterday },
    { value: '7days', label: labels.period7Days, icon: CalendarDays, description: labels.desc7Days },
    { value: '30days', label: labels.period30Days, icon: CalendarDays, description: labels.desc30Days },
    { type: 'separator', label: labels.periodDaily },
    { value: 'daily', label: labels.periodDaily, icon: CalendarDays, description: labels.descDaily },
    { type: 'separator', label: labels.periodExtended },
    { value: 'weekly', label: labels.periodWeekly, icon: CalendarDays, description: labels.descWeekly },
    { value: 'monthly', label: labels.periodMonthly, icon: CalendarDays, description: labels.descMonthly },
    { value: 'yearly', label: labels.periodYearly, icon: CalendarDays, description: labels.descYearly },
  ]);

  function getPeriodDateRange(periodType) {
    const today = getTodayInJakarta();
    const daysAgo = getDateNDaysAgoInJakarta;

    switch (periodType) {
      case 'realtime':
        return { start: today, end: today };
      case 'yesterday':
        return { start: daysAgo(1), end: daysAgo(1) };
      case '7days':
        return { start: daysAgo(7), end: daysAgo(1) };
      case '30days':
        return { start: daysAgo(30), end: daysAgo(1) };
      case 'daily': {
        if (!dailySelectionMade || !selectedDailyDate) {
          return { start: today, end: today };
        }
        const d = selectedDailyDate.start ?? selectedDailyDate;
        const y = d.year;
        const m = String(d.month).padStart(2, '0');
        const day = String(d.day).padStart(2, '0');
        const dateStr = `${y}-${m}-${day}`;
        return { start: dateStr, end: dateStr };
      }
      case 'weekly':
        if (weeklySelectionMade && selectedWeeklyRange) {
          const start = selectedWeeklyRange.start;
          const end = selectedWeeklyRange.end;
          let endStr = `${end.year}-${String(end.month).padStart(2, '0')}-${String(end.day).padStart(2, '0')}`;
          const yesterday = getDateNDaysAgoInJakarta(1).split('-').map(Number);
          const yesterdayDate = new CalendarDate(yesterday[0], yesterday[1], yesterday[2]);
          if (end.compare(yesterdayDate) > 0 && start.compare(yesterdayDate) <= 0) {
            endStr = `${yesterday[0]}-${String(yesterday[1]).padStart(2, '0')}-${String(yesterday[2]).padStart(2, '0')}`;
          }
          return {
            start: `${start.year}-${String(start.month).padStart(2, '0')}-${String(start.day).padStart(2, '0')}`,
            end: endStr
          };
        }
        {
          const dayOfWeek = getJakartaDayOfWeek();
          const mondayOffset = dayOfWeek === 0 ? 6 : dayOfWeek - 1;
          const monday = getDateNDaysAgoInJakarta(mondayOffset);
          const yesterday = getDateNDaysAgoInJakarta(1);
          return { start: monday, end: yesterday };
        }
      case 'monthly':
        if (monthlySelectionMade && selectedMonthlyRange) {
          const start = selectedMonthlyRange.start;
          const end = selectedMonthlyRange.end;
          let endStr = `${end.year}-${String(end.month).padStart(2, '0')}-${String(end.day).padStart(2, '0')}`;
          const todayJakarta = getTodayInJakarta().split('-').map(Number);
          if (start.year === todayJakarta[0] && start.month === todayJakarta[1]) {
            const yesterday = getDateNDaysAgoInJakarta(1).split('-');
            endStr = `${yesterday[0]}-${yesterday[1]}-${yesterday[2]}`;
          }
          return {
            start: `${start.year}-${String(start.month).padStart(2, '0')}-${String(start.day).padStart(2, '0')}`,
            end: endStr
          };
        }
        const todayForMonthly = getTodayInJakarta().split('-').map(Number);
        const lastMonthStart = getFirstOfMonthNAgoInJakarta(1);
        const lastMonthEnd = getDateNDaysAgoInJakarta(1);
        return { start: lastMonthStart, end: lastMonthEnd };
      case 'yearly':
        if (yearlySelectionMade && selectedYearlyRange) {
          const start = selectedYearlyRange.start;
          let endMonth = 12;
          let endDay = 31;
          const todayJakarta = getTodayInJakarta().split('-').map(Number);
          const currentYear = todayJakarta[0];
          const currentMonth = todayJakarta[1];
          if (start.year === currentYear) {
            if (currentMonth === 1) {
              return { start: `${start.year}-01-01`, end: `${start.year}-01-01` };
            }
            endMonth = currentMonth - 1;
            const lastDayOfPrevMonth = new Date(currentYear, currentMonth - 1, 0).getDate();
            endDay = lastDayOfPrevMonth;
          }
          return {
            start: `${start.year}-01-01`,
            end: `${start.year}-${String(endMonth).padStart(2, '0')}-${String(endDay).padStart(2, '0')}`
          };
        }
        return { start: `${selectedYear}-01-01`, end: `${selectedYear}-12-31` };
      default:
        return { start: daysAgo(8), end: daysAgo(1) };
    }
  }

  function getPeriodDescription() {
    const range = getPeriodDateRange(selectedPeriodType);
    const start = formatDate(range.start);
    const end = formatDate(range.end);

    switch (selectedPeriodType) {
      case 'realtime':
        return labels.periodDescRealtime.replace('{time}', currentTimeHour);
      case 'yesterday':
        return labels.periodDescYesterday.replace('{date}', start);
      case '7days':
        return labels.periodDesc7Days.replace('{start}', start).replace('{end}', end);
      case '30days':
        return labels.periodDesc30Days.replace('{start}', start).replace('{end}', end);
      case 'daily':
        return labels.periodDescDaily.replace('{date}', start);
      case 'weekly':
        return labels.periodDescWeekly.replace('{start}', start).replace('{end}', end);
      case 'monthly':
        return labels.periodDescMonthly.replace('{start}', start).replace('{end}', end);
      case 'yearly':
        return labels.periodDescYearly.replace('{start}', start).replace('{end}', end);
      default:
        return `${start} - ${end}`;
    }
  }

  function setPeriod(periodType) {
    if (periodType === 'daily' || periodType === 'monthly' || periodType === 'yearly') {
      return;
    }

    if (periodType === 'weekly') {
      selectedPeriodType = 'weekly';
      activePeriodType = 'weekly';
      dropdownOpen = false;
      weeklySelectionMade = false;
      const range = getPeriodDateRange('weekly');
      const [sy, sm, sd] = range.start.split('-').map(Number);
      const [ey, em, ed] = range.end.split('-').map(Number);
      selectedWeeklyRange = {
        start: new CalendarDate(sy, sm, sd),
        end: new CalendarDate(ey, em, ed)
      };
      weeklySelectionMade = true;
      onfetchsaleswithrange(range.start, range.end);
      return;
    }

    selectedPeriodType = periodType;
    activePeriodType = periodType;
    dropdownOpen = false;
    const range = getPeriodDateRange(periodType);
    onfetchsaleswithrange(range.start, range.end);
  }

  onMount(() => {
    setPeriod(selectedPeriodType);
  });
</script>

<svelte:window
  onclick={(e) => {
    const path = e.composedPath?.() || [];
    const inDropdown = path.some(el => {
      const classList = el?.classList;
      return classList && (classList.contains('card-glass') ||
                           el.closest?.('.card-glass') !== null);
    });
    if (!inDropdown) {
      if (dropdownOpen) dropdownOpen = false;
    }
  }}
  onkeydown={(e) => {
    if (e.key === 'Escape' && dropdownOpen) dropdownOpen = false;
  }}
/>

<div class="card p-4 flex flex-wrap items-center gap-4">
  <div class="flex items-center gap-2 text-sm font-medium text-text-secondary">
    <BarChart3 size={16} class="text-white" />
    {labels.period}
  </div>

  <div class="relative">
    <Button
      variant="secondary"
      class="flex items-center gap-2"
      onclick={(e) => { e.stopPropagation(); dropdownOpen = !dropdownOpen; }}
      aria-haspopup="menu"
      aria-expanded={dropdownOpen}
      aria-controls="period-dropdown-menu"
    >
      <CalendarDays size={15} />
      {getPeriodDescription()}
      <ChevronDown
        size={14}
        class="transition-transform duration-300 {dropdownOpen ? 'rotate-180' : ''}"
      />
    </Button>

    {#if dropdownOpen}
      <div
        id="period-dropdown-menu"
        class="absolute left-0 top-full mt-2 card-glass p-2 z-50 min-w-[28rem] flex gap-2"
        role="menu"
        aria-orientation="vertical"
        tabindex="-1"
        transition:fly={{ y: -8, duration: 200 }}
      >
        <div class="flex flex-col gap-1 min-w-32">
          {#each periodOptions as option}
            {#if option.type === 'separator'}
              <div class="px-3 py-1 text-xs font-semibold text-text-muted uppercase tracking-wide">
                {option.label}
              </div>
            {:else}
              {@const isCalendarOption = ['daily', 'weekly', 'monthly', 'yearly'].includes(option.value)}
              <button type="button"
                role="menuitem"
                class="flex items-center gap-2 px-3 py-2 text-sm rounded-lg transition-colors {selectedPeriodType === option.value ? 'bg-primary/20 text-primary-light' : 'text-text-secondary hover:bg-primary/10 hover:text-primary-light'}"
                onclick={() => { if (!isCalendarOption) setPeriod(option.value); }}
                onmouseenter={() => hoveredOption = option}
              >
                <option.icon size={14} />
                {option.label}
              </button>
            {/if}
          {/each}
        </div>

        <div class="flex-1 min-w-80 border-l border-border/50 pl-2">
          <div class="text-xs text-text-secondary mb-2">{labels.details}</div>

          {#if hoveredOption?.value === 'realtime'}
            <div class="text-sm text-text-primary mb-2">{labels.realtimeRevenue}</div>
            <div class="text-xs text-text-muted">{labels.showsRealtime}</div>
          {:else if hoveredOption?.value === 'yesterday'}
            <div class="text-sm text-text-primary mb-2">{labels.yesterdayRevenue}</div>
            <div class="text-xs text-text-muted">{labels.showsYesterday}</div>
          {:else if hoveredOption?.value === '7days'}
            <div class="text-sm text-text-primary mb-2">{labels.days7Revenue}</div>
            <div class="text-xs text-text-muted">{labels.shows7Days}</div>
          {:else if hoveredOption?.value === '30days'}
            <div class="text-sm text-text-primary mb-2">{labels.days30Revenue}</div>
            <div class="text-xs text-text-muted">{labels.shows30Days}</div>
          {:else if hoveredOption?.value === 'daily'}
            <div class="text-sm text-text-primary mb-2">
              <span class="block text-xs text-text-muted mb-2">{labels.selectDate}</span>
              <SelectableCalendar
                mode="day"
                bind:value={selectedDailyDate}
                minValue={new CalendarDate(2023, 6, 16)}
                maxValue={yesterdayDate}
                theme={calendarTheme}
                onValueChange={(val) => {
                  if (val) {
                    selectedDailyDate = val;
                    const d = val.start || val;
                    const y = d.year;
                    const m = String(d.month).padStart(2, '0');
                    const day = String(d.day).padStart(2, '0');
                    const dateStr = `${y}-${m}-${day}`;
                    dailySelectionMade = true;
                    activePeriodType = 'daily';
                    selectedPeriodType = 'daily';
                    dropdownOpen = false;
                    onfetchsaleswithrange(dateStr, dateStr);
                  }
                }}
              />
            </div>
            <div class="text-xs text-text-muted">{labels.showsDaily}</div>
          {:else if hoveredOption?.value === 'weekly'}
            <div class="text-sm text-text-primary mb-2">
              <span class="block text-xs text-text-muted mb-2">{labels.selectWeek}</span>
              <SelectableCalendar
                mode="week"
                bind:value={selectedWeeklyRange}
                minValue={new CalendarDate(2023, 6, 16)}
                maxValue={yesterdayDate}
                theme={calendarTheme}
                onValueChange={(val) => {
                  if (val) {
                    selectedWeeklyRange = val;
                    weeklySelectionMade = true;
                    const start = val.start;
                    const end = val.end;
                    let endStr = `${end.year}-${String(end.month).padStart(2, '0')}-${String(end.day).padStart(2, '0')}`;
                    const yesterday = getDateNDaysAgoInJakarta(1).split('-').map(Number);
                    const yesterdayDate = new CalendarDate(yesterday[0], yesterday[1], yesterday[2]);
                    if (end.compare(yesterdayDate) > 0 && start.compare(yesterdayDate) <= 0) {
                      endStr = `${yesterday[0]}-${yesterday[1]}-${yesterday[2]}`;
                    }
                    activePeriodType = 'weekly';
                    selectedPeriodType = 'weekly';
                    dropdownOpen = false;
                    onfetchsaleswithrange(
                      `${start.year}-${String(start.month).padStart(2, '0')}-${String(start.day).padStart(2, '0')}`,
                      endStr
                    );
                  }
                }}
              />
            </div>
            <div class="text-xs text-text-muted">{labels.showsWeekly}</div>
          {:else if hoveredOption?.value === 'monthly'}
            <div class="text-sm text-text-primary mb-2">
              <span class="block text-xs text-text-muted mb-2">{labels.selectMonth}</span>
              <MonthlyCalendar
                bind:value={selectedMonthlyRange}
                minValue={new CalendarDate(2023, 6, 1)}
                maxValue={yesterdayDate}
                theme={calendarTheme}
                onValueChange={(val) => {
                  if (val) {
                    selectedMonthlyRange = val;
                    monthlySelectionMade = true;
                    const start = val.start;
                    const end = val.end;
                    const startStr = `${start.year}-${String(start.month).padStart(2, '0')}-${String(start.day).padStart(2, '0')}`;
                    let endStr = `${end.year}-${String(end.month).padStart(2, '0')}-${String(end.day).padStart(2, '0')}`;
                    const todayJakarta = getTodayInJakarta().split('-').map(Number);
                    const isCurrentMonth = start.year === todayJakarta[0] && start.month === todayJakarta[1];
                    if (isCurrentMonth) {
                      const yesterday = getDateNDaysAgoInJakarta(1).split('-');
                      endStr = `${yesterday[0]}-${yesterday[1]}-${yesterday[2]}`;
                    }
                    activePeriodType = 'monthly';
                    selectedPeriodType = 'monthly';
                    dropdownOpen = false;
                    onfetchsaleswithrange(startStr, endStr);
                  }
                }}
              />
            </div>
            <div class="text-xs text-text-muted">{labels.showsMonthly}</div>
          {:else if hoveredOption?.value === 'yearly'}
            <div class="text-sm text-text-primary mb-2">
              <span class="block text-xs text-text-muted mb-2">{labels.selectYear}</span>
              <YearCalendar
                bind:value={selectedYearlyRange}
                minValue={availableYears.length > 0 ? new CalendarDate(Math.min(...availableYears), 1, 1) : new CalendarDate(2023, 6, 16)}
                maxValue={yesterdayDate}
                {availableYears}
                theme={calendarTheme}
                onValueChange={(val) => {
                  if (val) {
                    selectedYearlyRange = val;
                    yearlySelectionMade = true;
                    const year = val.start.year;
                    const todayJakarta = getTodayInJakarta();
                    const todayParts = todayJakarta.split('-').map(Number);
                    const currentYear = todayParts[0];
                    const currentMonth = todayParts[1];
                    let endMonth = 12;
                    let endDay = 31;
                    if (year === currentYear) {
                      if (currentMonth === 1) {
                        activePeriodType = 'yearly';
                        selectedPeriodType = 'yearly';
                        dropdownOpen = false;
                        onfetchsaleswithrange(`${year}-01-01`, `${year}-01-01`);
                        return;
                      }
                      endMonth = currentMonth - 1;
                      const lastDayOfPrevMonth = new Date(year, currentMonth - 1, 0).getDate();
                      endDay = lastDayOfPrevMonth;
                    }
                    activePeriodType = 'yearly';
                    selectedPeriodType = 'yearly';
                    dropdownOpen = false;
                    onfetchsaleswithrange(`${year}-01-01`, `${year}-${String(endMonth).padStart(2, '0')}-${String(endDay).padStart(2, '0')}`);
                  }
                }}
              />
            </div>
            <div class="text-xs text-text-muted">{labels.showsYearly}</div>
          {/if}

          <div class="text-xs text-text-muted mt-2">
            {labels.timezone} {timezoneString}
          </div>
        </div>
      </div>
    {/if}
  </div>

  <div class="ml-auto">
    <Dropdown items={[
      { label: labels.exportToExcel, icon: FileSpreadsheet, iconClass: 'text-success-light', onclick: onexportexcel },
      { label: labels.exportToPdf, icon: Download, iconClass: 'text-danger-light', onclick: onexportpdf },
    ]}>
      {#snippet trigger({ toggle })}
        <Button
          variant="primary"
          class="flex items-center gap-2 transition-all duration-300"
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
