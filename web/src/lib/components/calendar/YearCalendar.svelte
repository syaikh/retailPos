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

  let hoverYear: number | null = $state(null);

  const today = new CalendarDate(
    new Date().getFullYear(),
    new Date().getMonth() + 1,
    new Date().getDate()
  );

const effectiveMaxValue = $derived(maxValue ?? new CalendarDate(today.year, 12, 31));
  // Calendar shows years from (current-15) to 2030, placing current year in last row
  const currentYearBasedStart = $derived(Math.max(minValue?.year ?? 1900, today.year - 15));
  const maxYear = $derived(Math.min(effectiveMaxValue.year, 2030));
  let centerYear = $state(0);
  $effect(() => {
    if (centerYear === 0) centerYear = today.year;
  });
  // Generate years array ensuring current year is in last row
  const years = $derived(
    Array.from({ length: maxYear - currentYearBasedStart + 1 }, (_, i) => currentYearBasedStart + i)
  );

  const getYearRange = (year: number): { start: DateValue; end: DateValue } => {
    return {
      start: new CalendarDate(year, 1, 1),
      end: new CalendarDate(year, 12, 31),
    };
  };

  const isYearSelected = (year: number): boolean => {
    if (!value) return false;
    const yearStart = new CalendarDate(year, 1, 1);
    const yearEnd = new CalendarDate(year, 12, 31);
    // Check if year overlaps with selected range (partial selection support)
    return yearStart.compare(value.end) <= 0 && yearEnd.compare(value.start) >= 0;
  };

  const isYearDisabled = (year: number): boolean => {
    const yearStart = new CalendarDate(year, 1, 1);
    const yearEnd = new CalendarDate(year, 12, 31);

    if (minValue) {
      if (yearEnd.year < minValue.year) return true;
    }
    if (effectiveMaxValue) {
      if (yearStart.year > effectiveMaxValue.year) return true;
    }
    return false;
  };

  const isCurrentYear = (year: number): boolean => {
    return year === today.year;
  };

  const isYearInHover = (year: number): boolean => {
    return hoverYear === year;
  };

  const handleYearClick = (year: number) => {
    if (isYearDisabled(year)) return;
    const range = getYearRange(year);
    onValueChange?.(range);
  };

  const getYearClass = (year: number) => {
    const selected = isYearSelected(year);
    const hover = isYearInHover(year);
    const current = isCurrentYear(year);
    const disabled = isYearDisabled(year);

    return cn(
      "w-14 h-11 flex items-center justify-center text-sm rounded transition-colors",
      // Selected takes priority - always show with selected text regardless of disabled
      selected && "bg-[var(--calendar-selected)] text-[var(--calendar-selected-text)]",
      // Then disabled (not selected) - grey out with rounded corners
      disabled && !selected && "text-[var(--calendar-muted)] opacity-40 cursor-not-allowed rounded-md bg-[var(--calendar-disabled-bg)]",
      !disabled && !selected && "text-[var(--calendar-text)]",
      // Then hover (not selected or disabled)
      hover && !selected && !disabled && "bg-[var(--calendar-hover)]",
      // No outline for current year
    );
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
        aria-label="Previous decade"
        onclick={() => centerYear = Math.max(currentYearBasedStart, centerYear - 10)}
        disabled={centerYear - 10 <= currentYearBasedStart}
      >
        <span class="text-xs">‹</span>
      </button>
      <span class="text-sm font-medium text-[var(--calendar-text)]">{currentYearBasedStart} - {maxYear}</span>
      <button
        class="inline-flex items-center justify-center rounded-md p-1 text-[var(--calendar-text)] hover:bg-[var(--calendar-hover)] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        aria-label="Next decade"
        onclick={() => centerYear = Math.min(maxYear, centerYear + 10)}
        disabled={centerYear + 10 > maxYear}
      >
        <span class="text-xs">›</span>
      </button>
    </div>

    <div class="grid grid-cols-5 gap-2 flex-1">
      {#each years as year}
        {@const disabled = isYearDisabled(year)}
        <button
          class={getYearClass(year)}
          {disabled}
          onmouseenter={() => !disabled && (hoverYear = year)}
          onmouseleave={() => (hoverYear = null)}
          onclick={() => handleYearClick(year)}
        >
          {year}
        </button>
      {/each}
    </div>
  </div>
</div>