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
    expect(src).toContain("import { Button, Input, SearchBar, Dropdown } from '$shared/ui'");
  });

  it('imports lucide icons', () => {
    expect(src).toContain("import { CalendarDays, ChevronDown, Download, FileSpreadsheet, X } from 'lucide-svelte'");
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

  it('has amountError derived', () => {
    expect(src).toContain('const amountError = $derived');
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

  it('has togglePendingPaymentMethod function', () => {
    expect(src).toContain('function togglePendingPaymentMethod');
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

  it('has openDatePicker and cancelCustomRange', () => {
    expect(src).toContain('function openDatePicker');
    expect(src).toContain('function cancelCustomRange');
  });

  it('has canApplyCustom derived', () => {
    expect(src).toContain('const canApplyCustom = $derived');
  });

  it('renders SearchBar component', () => {
    expect(src).toContain('<SearchBar');
  });

  it('renders payment dropdown via Dropdown', () => {
    expect(src).toContain('<Dropdown');
  });

  it('renders export via Dropdown', () => {
    expect(src).toContain('Export to CSV');
    expect(src).toContain('Export to Excel');
  });

  it('renders date picker with presets and custom range', () => {
    expect(src).toContain('showDatePicker');
    expect(src).toContain('Preset Ranges');
    expect(src).toContain('Custom Range');
  });

  it('has date picker footer with Cancel and Apply', () => {
    expect(src).toContain('Cancel');
    expect(src).toContain('Apply');
  });

  it('does not have old draft/applied pattern', () => {
    expect(src).not.toContain('appliedPaymentMethods');
    expect(src).not.toContain('appliedSliderMin');
    expect(src).not.toContain('appliedSliderMax');
    expect(src).not.toContain('function applyFilters');
    expect(src).not.toContain('function cancelFilters');
    expect(src).not.toContain('function resetFilters');
  });
});
