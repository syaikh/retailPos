import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'PeriodSelector.svelte'), 'utf-8');
}

describe('PeriodSelector.svelte source-structure guards', () => {
  const src = getSource();

  it('uses $props() with $bindable for period state', () => {
    expect(src).toContain('$props()');
    expect(src).toContain('$bindable(');
  });

  it('imports calendar components', () => {
    expect(src).toContain("import { SelectableCalendar, MonthlyCalendar, YearCalendar }");
  });

  it('imports CalendarDate from @internationalized/date', () => {
    expect(src).toContain("import { CalendarDate } from '@internationalized/date'");
  });

  it('imports lucide icons', () => {
    expect(src).toContain("BarChart3");
    expect(src).toContain("CalendarDays");
    expect(src).toContain("ChevronDown");
    expect(src).toContain("Download");
    expect(src).toContain("FileSpreadsheet");
  });

  it('imports fly from svelte/transition', () => {
    expect(src).toContain("import { fly } from 'svelte/transition'");
  });

  it('imports Button from $shared/ui', () => {
    expect(src).toContain("import { Button } from '$shared/ui'");
  });

  it('has getPeriodDescription function', () => {
    expect(src).toContain('function getPeriodDescription');
  });

  it('has setPeriod function', () => {
    expect(src).toContain('function setPeriod');
  });

  it('has periodOptions constant', () => {
    expect(src).toContain('const periodOptions');
  });

  it('has getPeriodDateRange function', () => {
    expect(src).toContain('function getPeriodDateRange');
  });

  it('has export button with callbacks', () => {
    expect(src).toContain('onexportexcel');
    expect(src).toContain('onexportpdf');
  });

  it('has svelte:window for closing dropdowns', () => {
    expect(src).toContain('<svelte:window');
  });

  it('has calendar detail sections', () => {
    expect(src).toContain("mode=\"day\"");
    expect(src).toContain("mode=\"week\"");
    expect(src).toContain("MonthlyCalendar");
    expect(src).toContain("YearCalendar");
  });

  it('has onfetchsaleswithrange callback', () => {
    expect(src).toContain('onfetchsaleswithrange');
  });

  it('has yesterdayDate derived', () => {
    expect(src).toContain('let yesterdayDate = $derived');
  });
});
