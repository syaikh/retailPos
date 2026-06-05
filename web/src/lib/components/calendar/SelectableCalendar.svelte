<script lang="ts">
  import { type DateValue, CalendarDate } from "@internationalized/date";
  import { cn, getThemeStyle, type Theme } from "./utils";
  import { getTodayJakartaDate, getCompletedDaysInCurrentWeek, getJakartaDayOfWeek } from '$lib/utils/jakartaTime';

  type SelectionMode = "day" | "week";

  interface Props {
    value?: { start: DateValue; end: DateValue } | null;
    onValueChange?: (value: { start: DateValue; end: DateValue } | null) => void;
    mode?: SelectionMode;
    minValue?: DateValue;
    maxValue?: DateValue;
    theme?: Theme;
    class?: string;
  }

  let {
    value = null,
    onValueChange,
    mode = "week",
    minValue,
    maxValue,
    theme = {},
    class: className,
  }: Props = $props();

  let hoverDate: DateValue | null = $state(null);
  let displayMonth = $state(new CalendarDate(
    new Date().getFullYear(),
    new Date().getMonth() + 1,
    1
  ));

  // Use Jakarta timezone for today to match backend calculations
  const today = new CalendarDate(
    getTodayJakartaDate().year,
    getTodayJakartaDate().month,
    getTodayJakartaDate().day
  );

  // Get day of week for threshold calculation (0=Sunday, 1=Monday, ..., 6=Saturday)
  const jakartaDayOfWeek = getJakartaDayOfWeek();

  const daysOfWeek = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];

  const getWeekRange = (date: DateValue): { start: DateValue; end: DateValue } => {
    const jsDate = new Date(date.year, date.month - 1, date.day);
    const dayOfWeek = jsDate.getDay();
    const daysToMonday = dayOfWeek === 0 ? 6 : dayOfWeek - 1;
    const weekStart = date.subtract({ days: daysToMonday });
    const weekEnd = weekStart.add({ days: 6 });
    return { start: weekStart, end: weekEnd };
  };

  const isSelectedStartDate = (date: DateValue): boolean => {
    if (!value) return false;
    const rangeStart = value.start ?? value;
    return date.compare(rangeStart) === 0;
  };

  const isSelectedEndDate = (date: DateValue): boolean => {
    if (!value) return false;
    const rangeEnd = value.end ?? value;
    return date.compare(rangeEnd) === 0;
  };

  const getHoverRange = (): { start: DateValue | null; end: DateValue | null } => {
    if (!hoverDate) return { start: null, end: null };
    let range: { start: DateValue; end: DateValue };
    if (mode === "week") {
      range = getWeekRange(hoverDate);
    } else {
      range = { start: hoverDate, end: hoverDate };
    }
    // Constrain to minValue/maxValue for partial week highlighting
    if (minValue && range.start.compare(minValue) < 0) {
      range.start = minValue;
    }
    if (maxValue && range.end.compare(maxValue) > 0) {
      range.end = maxValue;
    }
    return range;
  };

  const getDayClass = (date: DateValue, currentMonth: { year: number; month: number }) => {
    const selected = isDateInSelectedRange(date);
    const hover = isDateInHoverRange(date);
    const todayFlag = isToday(date);
    const inCurrentMonth = date.year === currentMonth.year && date.month === currentMonth.month;
    const disabled = isDateDisabled(date);
    const hoverRange = getHoverRange();
    const isHoverStart = hoverRange.start && date.compare(hoverRange.start) === 0;
    const isHoverEnd = hoverRange.end && date.compare(hoverRange.end) === 0;

    return cn(
      "relative w-11 h-11 text-center text-sm",
      selected && "bg-[var(--calendar-selected)] text-[var(--calendar-selected-text)] rounded-none",
      selected && isSelectedStartDate(date) && "rounded-l-md",
      selected && isSelectedEndDate(date) && "rounded-r-md",
      hover && !selected && !disabled && "bg-[var(--calendar-hover)] text-[var(--calendar-text)] rounded-none",
      isHoverStart && !selected && !disabled && "rounded-l-md",
      isHoverEnd && !selected && !disabled && "rounded-r-md",
      disabled && "text-[var(--calendar-muted)] opacity-40 cursor-not-allowed rounded-md bg-[var(--calendar-disabled-bg)]",
      todayFlag && mode === "week" && !selected && "text-[var(--calendar-muted)] opacity-40 rounded-md bg-[var(--calendar-disabled-bg)]",
      // Today flag styling: only apply if NOT disabled (to avoid overriding disabled styling)
      todayFlag && !disabled && !selected && !hover && mode !== "week" && "ring-2 ring-[var(--calendar-today-border)] ring-offset-1",
      !disabled && !selected && !hover && !todayFlag && !inCurrentMonth && "text-[var(--calendar-muted)] opacity-60",
      !disabled && !selected && !hover && !todayFlag && inCurrentMonth && "text-[var(--calendar-text)]"
    );
  };

  const isToday = (date: DateValue): boolean => {
    return date.year === today.year && date.month === today.month && date.day === today.day;
  };

  const isDateInSelectedRange = (date: DateValue): boolean => {
    if (!value) return false;
    // Handle both range object and single CalendarDate
    let rangeStart = value.start ?? value;
    let rangeEnd = value.end ?? value;
    // Constrain to minValue/maxValue for partial week support
    if (mode === "week" && minValue && rangeStart.compare(minValue) < 0) {
      rangeStart = minValue;
    }
    if (mode === "week" && maxValue && rangeEnd.compare(maxValue) > 0) {
      rangeEnd = maxValue;
    }
    if (mode === "day") {
      return date.compare(rangeStart) === 0 && date.compare(rangeEnd) === 0;
    }
    return date.compare(rangeStart) >= 0 && date.compare(rangeEnd) <= 0;
  };

  const isDateInHoverRange = (date: DateValue): boolean => {
    if (!hoverDate) return false;
    let range = mode === "week" ? getWeekRange(hoverDate) : { start: hoverDate, end: hoverDate };
    // For week mode, check if hovering over a disabled (current partial) week
    const isCurrentWeekHover = mode === "week" && 
      range.start.compare(today) <= 0 && range.end.compare(today) >= 0 &&
      jakartaDayOfWeek >= 1 && jakartaDayOfWeek <= 3; // Mon, Tue, Wed - before threshold
    
    // Don't apply hover effect on disabled dates
    if (isCurrentWeekHover) return false;
    // Constrain hover range to minValue/maxValue for partial week support
    if (mode === "week" && minValue && range.start.compare(minValue) < 0) {
      range = { start: minValue, end: range.end };
    }
    if (mode === "week" && maxValue && range.end.compare(maxValue) > 0) {
      range = { start: range.start, end: maxValue };
    }
    return date.compare(range.start) >= 0 && date.compare(range.end) <= 0;
  };

  const isDateDisabled = (date: DateValue): boolean => {
    if (minValue && date.compare(minValue) < 0) return true;
    
    // For week mode, check 3-day threshold rule
    if (maxValue && mode === "week") {
      const weekStart = getWeekRange(date).start;
      const weekEnd = getWeekRange(date).end;
      
      // Check if this week is the current week (contains today or overlaps with today)
      const isCurrentWeek = weekStart.compare(today) <= 0 && weekEnd.compare(today) >= 0;
      
      // If this is the current week, check if 3+ days have passed (threshold rule)
      // Monday(1), Tuesday(2), Wednesday(3) -> 0, 1, 2 completed days -> DISABLED
      // Thursday(4), Friday(5), Saturday(6), Sunday(0) -> 3+ days completed -> SELECTABLE
      if (isCurrentWeek) {
        // Disable if dayOfWeek is Mon(1), Tue(2), or Wed(3) - less than 3 full days completed
        if (jakartaDayOfWeek >= 1 && jakartaDayOfWeek <= 3) {
          return true;
        }
      }
      
      // For non-current weeks, check if week extends beyond maxValue
      if (weekEnd.compare(maxValue) > 0) return true;
    } else if (maxValue && mode !== "week") {
      // For day mode, disable if date is after maxValue
      if (date.compare(maxValue) > 0) return true;
    }
    return false;
  };

  const handleDayClick = (date: DateValue) => {
    if (isDateDisabled(date)) return;
    let range = mode === "week" ? getWeekRange(date) : { start: date, end: date };
    // Constrain to maxValue for partial week support
    if (mode === "week" && maxValue && range.end.compare(maxValue) > 0) {
      range = { start: range.start, end: maxValue };
    }
    onValueChange?.(range);
  };

  const handleMouseEnter = (date: DateValue) => {
    hoverDate = date;
  };

  const handleMouseLeave = () => {
    hoverDate = null;
  };

  // Get current month dates with padding (max 5 rows = 35 days)
  function getMonthDates(year: number, month: number) {
    const firstDay = new Date(year, month - 1, 1);
    const lastDay = new Date(year, month, 0);
    const days: DateValue[] = [];
    const maxDays = 35;

    // Add previous month days
    const firstDayOfWeek = firstDay.getDay() || 7;
    const prevMonthEnd = new Date(year, month - 1, 0).getDate();
    for (let i = firstDayOfWeek - 1; i > 0 && days.length < maxDays; i--) {
      days.push(new CalendarDate(year, month - 1, prevMonthEnd - i + 1));
    }

    // Add current month days
    for (let i = 1; i <= lastDay.getDate() && days.length < maxDays; i++) {
      days.push(new CalendarDate(year, month, i));
    }

    // Add next month days to fill 35 days max
    const daysAdded = days.length;
    const nextMonth = month + 1;
    for (let i = 1; i <= maxDays - daysAdded; i++) {
      days.push(new CalendarDate(year, nextMonth, i));
    }

    return days;
  }

  const monthDates = $derived(getMonthDates(displayMonth.year, displayMonth.month));

  // Check if there are any selectable dates in the current month view
  const hasSelectableDates = $derived(
    monthDates.some(date => !isDateDisabled(date))
  );
</script>

<div class={cn("inline-block w-72", className)}>
  <div
    class="p-4 rounded-lg w-full min-h-80"
    style={getThemeStyle(theme)}
  >
    <div class="flex items-center justify-between mb-2">
      <button
        class="inline-flex items-center justify-center rounded-md p-1 text-[var(--calendar-text)] hover:bg-[var(--calendar-hover)] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        aria-label="Previous month"
onclick={(e) => { e.stopPropagation(); const prev = displayMonth.subtract({ months: 1 }); displayMonth = new CalendarDate(prev.year, prev.month, 1); }}
        disabled={displayMonth.year <= (minValue?.year ?? 1900) && displayMonth.month <= (minValue?.month ?? 1)}
      >
        <span class="text-xs">‹</span>
      </button>
      <span class="text-sm font-medium text-[var(--calendar-text)]">
        {displayMonth.year} {new Date(displayMonth.year, displayMonth.month - 1, 1).toLocaleString('en-US', { month: 'short' })}
      </span>
      <button
        class="inline-flex items-center justify-center rounded-md p-1 text-[var(--calendar-text)] hover:bg-[var(--calendar-hover)] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        aria-label="Next month"
onclick={(e) => { e.stopPropagation(); const next = displayMonth.add({ months: 1 }); displayMonth = new CalendarDate(next.year, next.month, 1); }}
        disabled={maxValue !== undefined && displayMonth.compare(maxValue) >= 0}
      >
        <span class="text-xs">›</span>
      </button>
    </div>

    <div class="grid grid-cols-7 gap-0.5 mb-2">
      {#each daysOfWeek as day}
        <div class="w-11 h-11 text-center text-xs font-medium text-[var(--calendar-muted)]">
          {day}
        </div>
      {/each}
    </div>

    {#if !hasSelectableDates}
      <div class="text-xs text-center py-4 text-[var(--calendar-muted)]">
        No selectable dates in this month.<br/>Click ‹ to view previous months.
      </div>
    {:else}
      <div class="grid grid-cols-7 gap-0.5" role="group" onmouseleave={handleMouseLeave}>
        {#each monthDates as date}
          {@const todayFlag = isToday(date)}
          {@const disabledFlag = isDateDisabled(date)}
          <button
            class={getDayClass(date, { year: displayMonth.year, month: displayMonth.month })}
            disabled={disabledFlag}
            onmouseenter={() => handleMouseEnter(date)}
            onclick={(e) => { e.stopPropagation(); handleDayClick(date); }}
          >
            <span class={cn(todayFlag && !disabledFlag && "font-bold")}>{date.day}</span>
          </button>
        {/each}
      </div>
    {/if}
  </div>
</div>