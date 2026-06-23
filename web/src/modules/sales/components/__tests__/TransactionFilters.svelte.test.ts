import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'TransactionFilters.svelte'), 'utf-8');
}

describe('TransactionFilters.svelte source-structure guards', () => {
  const src = getSource();

  it('uses $props()', () => {
    expect(src).toContain('= $props()');
  });

  it('uses $bindable() for props', () => {
    expect(src).toContain('$bindable(');
  });

  it('imports Jakarta time utilities', () => {
    expect(src).toContain("import { getTodayInJakarta, getDateNDaysAgoInJakarta, formatJakartaDateStr } from '$shared/utils/jakartaTime'");
  });

  it('imports SearchBar and Input', () => {
    expect(src).toContain("import { Button, Input, SearchBar } from '$shared/ui'");
  });

  it('imports lucide icons', () => {
    expect(src).toContain("import { CalendarDays, ChevronDown, Download, FileSpreadsheet } from 'lucide-svelte'");
  });

  it('imports getAuthToken', () => {
    expect(src).toContain("import { getAuthToken } from '$modules/auth'");
  });

  it('imports toast', () => {
    expect(src).toContain("import { toast } from '$shared/stores/toast.svelte'");
  });

  it('defines SLIDER_MAX_BOUND', () => {
    expect(src).toContain('SLIDER_MAX_BOUND');
  });

  it('defines datePresets', () => {
    expect(src).toContain('datePresets');
  });

  it('has dateRangeLabel derived', () => {
    expect(src).toContain('const dateRangeLabel = $derived');
  });

  it('has isFiltered derived', () => {
    expect(src).toContain('const isFiltered = $derived');
  });

  it('has hasPendingChanges derived', () => {
    expect(src).toContain('const hasPendingChanges = $derived');
  });

  it('has amountError derived', () => {
    expect(src).toContain('const amountError = $derived');
  });

  it('has minDisplay derived', () => {
    expect(src).toContain('const minDisplay = $derived');
  });

  it('has maxDisplay derived', () => {
    expect(src).toContain('const maxDisplay = $derived');
  });

  it('has handleMinInput and handleMaxInput', () => {
    expect(src).toContain('function handleMinInput');
    expect(src).toContain('function handleMaxInput');
  });

  it('has sanitizeSearch function', () => {
    expect(src).toContain('function sanitizeSearch');
  });

  it('has applyDatePreset function', () => {
    expect(src).toContain('function applyDatePreset');
  });

  it('has togglePaymentMethod function', () => {
    expect(src).toContain('function togglePaymentMethod');
  });

  it('has applyFilters function', () => {
    expect(src).toContain('function applyFilters');
  });

  it('has cancelFilters function', () => {
    expect(src).toContain('function cancelFilters');
  });

  it('has resetFilters function', () => {
    expect(src).toContain('function resetFilters');
  });

  it('has paymentMethodName function', () => {
    expect(src).toContain('function paymentMethodName');
  });

  it('has buildExportUrl function', () => {
    expect(src).toContain('function buildExportUrl');
  });

  it('has downloadExport function', () => {
    expect(src).toContain('async function downloadExport');
  });

  it('has exportCsv and exportExcel functions', () => {
    expect(src).toContain('function exportCsv');
    expect(src).toContain('function exportExcel');
  });

  it('has applyCustomRange function', () => {
    expect(src).toContain('function applyCustomRange');
  });

  it('renders SearchBar component', () => {
    expect(src).toContain('<SearchBar');
  });

  it('renders payment dropdown', () => {
    expect(src).toContain('showPaymentDropdown');
  });

  it('renders export dropdown', () => {
    expect(src).toContain('showExportDropdown');
  });

  it('renders date picker', () => {
    expect(src).toContain('showDatePicker');
  });

  it('renders Apply and Reset buttons', () => {
    expect(src).toContain('applyFilters');
    expect(src).toContain('resetFilters');
  });
});
