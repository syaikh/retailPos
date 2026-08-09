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

  it('imports i18n labels', () => {
    expect(src).toContain("import { labels, t } from '$shared/i18n'");
  });

  it('defines SLIDER_MAX_BOUND', () => {
    expect(src).toContain('SLIDER_MAX_BOUND');
  });

  it('defines datePresets with localized labels', () => {
    expect(src).toContain('datePresets');
    expect(src).toContain('labels.today');
    expect(src).toContain('labels.yesterday');
    expect(src).toContain('labels.last7Days');
    expect(src).toContain('labels.last30Days');
    expect(src).toContain('labels.thisMonth');
    expect(src).toContain('labels.thisYear');
  });

  it('has dateRangeLabel derived', () => {
    expect(src).toContain('const dateRangeLabel = $derived');
    expect(src).toContain("t('customDateRange'");
  });

  it('has amountError derived with localized messages', () => {
    expect(src).toContain('const amountError = $derived');
    expect(src).toContain('labels.errorMinCannotBeNegative');
    expect(src).toContain("t('errorMinExceedsMax'");
    expect(src).toContain('labels.errorMaxCannotBeNegative');
    expect(src).toContain("t('errorMaxExceedsMax'");
    expect(src).toContain('labels.errorMinCannotExceedMax');
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
    expect(src).toContain('labels.exportCSV');
    expect(src).toContain('labels.exportExcel');
  });

  it('renders date picker with presets and custom range', () => {
    expect(src).toContain('showDatePicker');
    expect(src).toContain('{labels.presetRanges}');
    expect(src).toContain('{labels.customRange}');
  });

  it('has date picker footer with localized Cancel and Apply', () => {
    expect(src).toContain('{labels.cancel}');
    expect(src).toContain('{labels.apply}');
  });

  it('localizes export toast errors', () => {
    expect(src).toContain('labels.toastSessionExpired');
    expect(src).toContain('labels.toastExportFailed');
  });

  it('localizes amount placeholders and reset', () => {
    expect(src).toContain('placeholder={labels.minLabel}');
    expect(src).toContain('placeholder={labels.maxLabel}');
    expect(src).toContain('labels.filterJumlah');
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
