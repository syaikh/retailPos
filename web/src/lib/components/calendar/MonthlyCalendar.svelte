<script lang="ts">
  import { type DateValue, CalendarDate } from "@internationalized/date";
  import { cn, getThemeStyle, type Theme } from "./utils";

  interface Props {
    value?: { start: DateValue; end: DateValue } | null;
    onValueChange?: (value: { start: DateValue; end: DateValue } | null) => void;
    minValue?: DateValue;
    maxValue?: DateValue;
    theme?: Theme;
    class?: string;
  }

  let {
    value = null,
    onValueChange,
    minValue,
    maxValue,
    theme = {},
    class: className,
  }: Props = $props();

  let year = $state(0);
  let hoverMonth: number | null = $state(null);
  let hoverYear: number = $state(0);

  const today = new CalendarDate(
    new Date().getFullYear(),
    new Date().getMonth() + 1,
    new Date().getDate()
  );

  // Initialize year from maxValue if available
  $effect(() => {
    if (year === 0) {
      year = maxValue?.year ?? today.year;
    }
  });

  const months = [
    "Jan", "Feb", "Mar", "Apr", "May", "Jun",
    "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"
  ];

  const getMonthRange = (year: number, month: number): { start: DateValue; end: DateValue } => {
    const monthStart = new CalendarDate(year, month, 1);
    const nextMonth = monthStart.add({ months: 1 });
    const monthEnd = nextMonth.subtract({ days: 1 });
    return { start: monthStart, end: monthEnd };
  };

  const isMonthSelected = (year: number, month: number): boolean => {
    if (!value) return false;
    const monthStart = new CalendarDate(year, month, 1);
    const nextMonth = monthStart.add({ months: 1 });
    const monthEnd = nextMonth.subtract({ days: 1 });
    // Check if month overlaps with selected range (partial selection support)
    return monthStart.compare(value.end) <= 0 && monthEnd.compare(value.start) >= 0;
  };

  const isMonthInHover = (year: number, month: number): boolean => {
    if (hoverMonth === null) return false;
    return hoverMonth === month && hoverYear === year;
  };

  const isCurrentMonth = (month: number): boolean => {
    return year === today.year && month === today.month;
  };

  const handleMonthClick = (month: number) => {
    if (isMonthDisabled(month)) return;
    const range = getMonthRange(year, month);
    onValueChange?.(range);
  };

  const handleMouseEnter = (month: number) => {
    hoverMonth = month;
    hoverYear = year;
  };

  const handleMouseLeave = () => {
    hoverMonth = null;
    hoverYear = 0;
  };

  const getMonthClass = (yearNum: number, month: number) => {
    const selected = isMonthSelected(yearNum, month);
    const hover = isMonthInHover(yearNum, month);
    const current = isCurrentMonth(month);
    const disabled = isMonthDisabled(month);

    return cn(
      "w-14 h-11 flex items-center justify-center text-sm rounded transition-colors",
      // Selected takes priority - always show with selected text regardless of disabled
      selected && "bg-[var(--calendar-selected)] text-[var(--calendar-selected-text)]",
      // Then disabled (not selected)
      disabled && !selected && "text-[var(--calendar-muted)] opacity-40 cursor-not-allowed",
      !disabled && !selected && "text-[var(--calendar-text)]",
      // Then hover (not selected or disabled)
      hover && !selected && !disabled && "bg-[var(--calendar-hover)]",
      current && !selected && !disabled && "ring-2 ring-[var(--calendar-today-border)] ring-offset-1"
    );
  };

  const isMonthDisabled = (month: number): boolean => {
    const monthStart = new CalendarDate(year, month, 1);
    const monthEnd = new CalendarDate(year, month, 1).add({ months: 1 }).subtract({ days: 1 });

    if (minValue) {
      if (monthEnd.year < minValue.year ||
          (monthEnd.year === minValue.year && monthEnd.month < minValue.month)) {
        return true;
      }
    }
    if (maxValue) {
      // Compare month starts with maxValue month
      // If monthStart is after maxValue's month, disable it
      if (monthStart.year > maxValue.year ||
          (monthStart.year === maxValue.year && monthStart.month > maxValue.month)) {
        return true;
      }
    }
    return false;
  };
</script>

<div class={cn("inline-block w-72", className)}>
  <div
    class="p-4 rounded-lg w-full h-80 flex flex-col"
    style={getThemeStyle(theme)}
  >
    <div class="flex items-center justify-between mb-3">
      <button
        class="inline-flex items-center justify-center rounded-md p-1 text-[var(--calendar-text)] hover:bg-[var(--calendar-hover)] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        aria-label="Previous year"
        onclick={() => year = Math.max(minValue?.year ?? 1900, year - 1)}
        disabled={year <= (minValue?.year ?? 1900)}
      >
        <span class="text-xs">‹</span>
      </button>
      <span class="text-sm font-medium text-[var(--calendar-text)]">{year}</span>
      <button
        class="inline-flex items-center justify-center rounded-md p-1 text-[var(--calendar-text)] hover:bg-[var(--calendar-hover)] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        aria-label="Next year"
        onclick={() => year = Math.min(maxValue?.year ?? 2100, year + 1)}
        disabled={year >= (maxValue?.year ?? 2100)}
      >
        <span class="text-xs">›</span>
      </button>
    </div>

    <div class="grid grid-cols-4 gap-2">
      {#each months as monthName, i}
        {@const month = i + 1}
        {@const disabled = isMonthDisabled(month)}
        <button
          class={getMonthClass(year, month)}
          disabled={disabled}
          onmouseenter={() => !disabled && handleMouseEnter(month)}
          onmouseleave={() => handleMouseLeave()}
          onclick={() => handleMonthClick(month)}
        >
          {monthName}
        </button>
      {/each}
    </div>
  </div>
</div>