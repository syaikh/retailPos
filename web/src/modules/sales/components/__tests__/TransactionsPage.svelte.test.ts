import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'TransactionsPage.svelte'), 'utf-8');
}

describe('TransactionsPage.svelte source-structure guards', () => {
  const src = getSource();

  it('imports apiFetch instead of apiClient', () => {
    expect(src).toContain("import { apiFetch } from '$shared/api/http-client'");
  });

  it('imports auth token helper', () => {
    expect(src).toContain("import { getAuthToken } from '$modules/auth'");
  });

  it('imports printReceipt store', () => {
    expect(src).toContain("import { printReceipt as printReceiptStore } from '$shared/stores/printReceipt.svelte'");
  });

  it('imports Jakarta time utilities', () => {
    expect(src).toContain("import { getTodayInJakarta, getDateNDaysAgoInJakarta, formatDateTimeInJakarta, formatJakartaDateStr } from '$shared/utils/jakartaTime'");
  });

  it('imports shared UI components', () => {
    expect(src).toContain("import { Badge, Button, Input, Pagination, SearchBar, Skeleton } from '$shared/ui'");
  });

  it('uses $state for salesData, pagination, search, date range', () => {
    expect(src).toContain('let salesData = $state([])');
    expect(src).toContain('let total = $state(0)');
    expect(src).toContain('let searchQuery = $state');
    expect(src).toContain('let startDate = $state');
    expect(src).toContain('let endDate = $state');
  });

  it('uses draft/applied filter pattern for payments and amount range', () => {
    expect(src).toContain('let appliedPaymentMethods');
    expect(src).toContain('let appliedSliderMin');
    expect(src).toContain('let appliedSliderMax');
    expect(src).toContain('let sliderMin');
    expect(src).toContain('let sliderMax');
  });

  it('uses $derived for dateRangeLabel, isFiltered, hasPendingChanges', () => {
    expect(src).toContain('const dateRangeLabel = $derived');
    expect(src).toContain('const isFiltered = $derived');
    expect(src).toContain('const hasPendingChanges = $derived');
  });

  it('has applyFilters and export functions', () => {
    expect(src).toContain('function applyFilters');
    expect(src).toContain('function exportCsv');
    expect(src).toContain('function exportExcel');
  });

  it('renders Pagination component', () => {
    expect(src).toContain('<Pagination');
  });
});
