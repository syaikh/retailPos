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

  it('imports extracted child components', () => {
    expect(src).toContain("import TransactionFilters from './TransactionFilters.svelte'");
    expect(src).toContain("import TransactionTable from './TransactionTable.svelte'");
    expect(src).toContain("import TransactionDrawer from './TransactionDrawer.svelte'");
  });

  it('renders child components in template', () => {
    expect(src).toContain('<TransactionFilters');
    expect(src).toContain('<TransactionTable');
    expect(src).toContain('<TransactionDrawer');
  });

  it('imports apiFetch instead of apiClient', () => {
    expect(src).toContain("import { apiFetch } from '$shared/api/http-client'");
  });

  it('imports Jakarta time utilities', () => {
    expect(src).toContain("import { getTodayInJakarta, getDateNDaysAgoInJakarta } from '$shared/utils/jakartaTime'");
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

  it('has fetchSales function', () => {
    expect(src).toContain('async function fetchSales');
  });

  it('has toggleSort and handlePageChange', () => {
    expect(src).toContain('function toggleSort');
    expect(src).toContain('function handlePageChange');
  });

  it('has openTransactionDetails and closeTransactionDrawer', () => {
    expect(src).toContain('function openTransactionDetails');
    expect(src).toContain('function closeTransactionDrawer');
  });

  it('has handleKeydown and handleWindowClick', () => {
    expect(src).toContain('function handleKeydown');
    expect(src).toContain('function handleWindowClick');
  });

  it('has sanitizeSearch function', () => {
    expect(src).toContain('function sanitizeSearch');
  });

  it('imports debounce', () => {
    expect(src).toContain("import { debounce } from '$shared/utils/debounce'");
  });

  it('has onMount lifecycle', () => {
    expect(src).toContain('onMount(');
  });

  it('has SLIDER_MAX_BOUND constant', () => {
    expect(src).toContain('SLIDER_MAX_BOUND');
  });
});
