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

  it('imports Jakarta time utilities', () => {
    expect(src).toContain("import { getTodayInJakarta, getDateNDaysAgoInJakarta } from '$shared/utils/jakartaTime'");
  });

  it('imports useSalesStore from store', () => {
    expect(src).toContain("import { useSalesStore } from '../stores/sales-store.svelte'");
  });

  it('imports auth store and RBAC', () => {
    expect(src).toContain("import { useAuthStore } from '$modules/auth'");
    expect(src).toContain("import { useRBAC } from '$shared/composables/useRBAC.svelte'");
  });

  it('sets cashierId filter for cashier role', () => {
    expect(src).toContain('store.cashierId');
  });

  it('imports createQueryManager', () => {
    expect(src).toContain("import { createQueryManager } from '../lib/query-manager'");
  });

  it('initializes store with default dates', () => {
    expect(src).toContain('store.startDate = ');
    expect(src).toContain('store.endDate = ');
    expect(src).toContain("store.dateRange = 'last30d'");
  });

  it('has toggleSort and handlePageChange', () => {
    expect(src).toContain('function toggleSort');
    expect(src).toContain('function handlePageChange');
  });

  it('has openTransactionDetails and closeTransactionDrawer', () => {
    expect(src).toContain('function openTransactionDetails');
    expect(src).toContain('function closeTransactionDrawer');
  });

  it('has handleKeydown', () => {
    expect(src).toContain('function handleKeydown');
  });

  it('has onMount lifecycle', () => {
    expect(src).toContain('onMount(');
  });

  it('has $effect for filter watching', () => {
    expect(src).toContain('$effect(() => {');
  });

  it('creates query manager', () => {
    expect(src).toContain('const qm = createQueryManager');
  });

  it('calls store.loadPaymentMethods in onMount', () => {
    expect(src).toContain('store.loadPaymentMethods()');
  });

  it('binds filter props to store values', () => {
    expect(src).toContain("bind:searchQuery={store.searchQuery}");
    expect(src).toContain("bind:startDate={store.startDate}");
    expect(src).toContain('bind:showDatePicker');
  });
});
