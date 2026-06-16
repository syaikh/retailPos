<script lang="ts">
  import { type DateValue, CalendarDate } from "@internationalized/date";
  import { cn, getThemeStyle, type Theme } from "./utils";
  import { getTodayJakartaDate } from "$lib/utils/jakartaTime";

  interface Props {
    value?: { start: DateValue; end: DateValue } | null;
    onValueChange?: (value: { start: DateValue; end: DateValue } | null) => void;
    minValue?: DateValue;
    maxValue?: DateValue;
    theme?: Theme;
    class?: string;
    availableYears?: number[];
  }

  let {
    value = null,
    onValueChange,
    minValue,
    maxValue,
    theme = {},
    class: className,
    availableYears,
  }: Props = $props();

  let hoverYear: number | null = $state(null);

  const jakartaToday = getTodayJakartaDate();
  const today = new CalendarDate(jakartaToday.year, jakartaToday.month, jakartaToday.day);

const effectiveMaxValue = $derived(maxValue ?? new CalendarDate(today.year, 12, 31));
  // Calendar shows 16 years in 4x4 grid, ending at maxValue.year (or 2030)
  // Years without data are disabled but still shown
  const maxYear = $derived(Math.min(effectiveMaxValue.year, 2030));
  const yearStart = $derived(Math.max(1900, maxYear - 15));
  const years = $derived(
    Array.from({ length: 16 }, (_, i) => yearStart + i)
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
    // Disable year if availableYears is explicitly set and year is not in the list
    // If availableYears is empty/undefined, don't disable (allows all years)
    if (availableYears && availableYears.length > 0 && !availableYears.includes(year)) return true;

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
      <span class="text-sm font-medium text-[var(--calendar-text)]">{yearStart} - {maxYear}</span>
    </div>

    <div class="grid grid-cols-4 gap-2 flex-1">
      {#each years as year}
        {@const disabled = isYearDisabled(year)}
        <button
          class={getYearClass(year)}
          {disabled}
          onmouseenter={() => !disabled && (hoverYear = year)}
          onmouseleave={() => (hoverYear = null)}
          onclick={(e) => { e.stopPropagation(); handleYearClick(year); }}
        >
          {year}
        </button>
      {/each}
    </div>
  </div>
</div>