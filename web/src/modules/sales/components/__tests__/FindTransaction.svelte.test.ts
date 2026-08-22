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

  it('renders a redacted summary table (no items/cost/payment columns)', () => {
    expect(src).toContain('labels.cashierLabel');
    expect(src).toContain('labels.statusLabel');
    expect(src).toContain('sale.invoice_number');
    // Redacted: the full transaction table with customer/item columns is NOT used.
    expect(src).not.toContain('<TransactionTable');
  });

  it('does not expose cost or customer columns', () => {
    expect(src).not.toContain('labels.customerLabel');
    expect(src).not.toContain('sale.customer_name');
    expect(src).not.toContain('sale.items');
  });
});
