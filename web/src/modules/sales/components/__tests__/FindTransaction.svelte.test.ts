import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'FindTransaction.svelte'), 'utf-8');
}

describe('FindTransaction.svelte source-structure guards', () => {
  const src = getSource();

  it('calls the cross-cashier lookup service', () => {
    expect(src).toContain("import { getSalesLookup } from '../services/sales-service'");
    expect(src).toContain('getSalesLookup(');
  });

  it('uses a single SearchBar for invoice-only lookup (no default table load)', () => {
    expect(src).toContain('SearchBar');
    expect(src).toContain('onsubmit={runSearch}');
    expect(src).toContain('function runSearch');
    // No auto-load effect driving a default store-wide list on mount.
    expect(src).not.toContain('bind:selectedDateRange');
    expect(src).not.toContain('showExport={false}');
    expect(src).not.toContain('TransactionFilters');
  });

  it('surfaces the redaction notice', () => {
    expect(src).toContain('labels.lookupRedactedNotice');
  });

  it('renders a redacted summary table (no items/cost/payment/status columns)', () => {
    expect(src).toContain('labels.cashierLabel');
    expect(src).toContain('sale.invoice_number');
    // Redacted: the full transaction table with customer/item columns is NOT used.
    expect(src).not.toContain('<TransactionTable');
  });

  it('makes invoice lookup date-independent (widened window)', () => {
    expect(src).toContain("'2000-01-01'");
    expect(src).toContain('getTodayInJakarta');
    expect(src).not.toContain('getDateNDaysAgoInJakarta');
  });

  it('shows a search hint before a query and a no-results state after', () => {
    expect(src).toContain('labels.findTransactionHint');
    expect(src).toContain('noResultsFor');
    expect(src).toContain('!searchQuery.trim()');
  });

  it('re-runs the lookup when a column is sorted (no auto-load effect)', () => {
    expect(src).toContain('function toggleSort');
    // The removed auto-load $effect must be replaced by an explicit reload.
    expect(src).toContain('runSearch();');
  });

  it('does not expose cost or customer columns', () => {
    expect(src).not.toContain('labels.customerLabel');
    expect(src).not.toContain('sale.customer_name');
    expect(src).not.toContain('sale.items');
  });
});
