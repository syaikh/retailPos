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

  it('reuses TransactionFilters with export disabled', () => {
    expect(src).toContain("import TransactionFilters from './TransactionFilters.svelte'");
    expect(src).toContain('showExport={false}');
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

  it('hides payment-method and amount-range filters (receipt lookup needs only search + date range)', () => {
    expect(src).toContain('showPaymentMethods={false}');
    expect(src).toContain('showAmountRange={false}');
    // The shared filter's payment/amount widgets must not be wired in for lookup.
    expect(src).not.toContain('labels.allMethods');
    expect(src).not.toContain('labels.minLabel');
    expect(src).not.toContain('labels.maxLabel');
  });

  it('keeps the invoice search bar and date-range widgets', () => {
    expect(src).toContain('bind:searchQuery');
    expect(src).toContain('bind:selectedDateRange');
    expect(src).toContain('searchPlaceholder={labels.searchByInvoiceNumber}');
  });

  it('does not expose cost or customer columns', () => {
    expect(src).not.toContain('labels.customerLabel');
    expect(src).not.toContain('sale.customer_name');
    expect(src).not.toContain('sale.items');
  });
});
