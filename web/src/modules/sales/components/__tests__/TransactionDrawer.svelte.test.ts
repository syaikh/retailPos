import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'TransactionDrawer.svelte'), 'utf-8');
}

describe('TransactionDrawer.svelte source-structure guards', () => {
  const src = getSource();

  it('uses $props()', () => {
    expect(src).toContain('= $props()');
  });

  it('imports Badge, Button, and Drawer', () => {
    expect(src).toContain("import { Badge, Button, Drawer } from '$shared/ui'");
  });

  it('imports Printer, Download icons', () => {
    expect(src).toContain("import { Printer, Download } from 'lucide-svelte'");
  });

  it('imports formatDateTimeInJakarta', () => {
    expect(src).toContain("import { formatDateTimeInJakarta } from '$shared/utils/jakartaTime'");
  });

  it('imports printReceipt store', () => {
    expect(src).toContain("import { printReceipt as printReceiptStore } from '$shared/stores/printReceipt.svelte'");
  });

  it('imports downloadInvoice', () => {
    expect(src).toContain("import { downloadInvoice } from '$modules/sales/lib/invoicePdf'");
  });

  it('imports toast', () => {
    expect(src).toContain("import { toast } from '$shared/stores/toast.svelte'");
  });

  it('has statusVariant function', () => {
    expect(src).toContain('function statusVariant');
  });

  it('has getPaymentMethodVariant function', () => {
    expect(src).toContain('function getPaymentMethodVariant');
  });

  it('has formatDateTime function', () => {
    expect(src).toContain('const formatDateTime');
  });

  it('has printTransactionReceipt function', () => {
    expect(src).toContain('function printTransactionReceipt');
  });

  it('has downloadInvoiceHandler function', () => {
    expect(src).toContain('async function downloadInvoiceHandler');
  });

  it('renders Print Receipt button', () => {
    expect(src).toContain('Print Receipt');
  });

  it('renders Download Invoice button', () => {
    expect(src).toContain('Download Invoice');
  });

  it('renders Close button', () => {
    expect(src).toContain('Close');
  });

  it('renders transaction details heading', () => {
    expect(src).toContain('Transaction Details');
  });

  it('renders items table', () => {
    expect(src).toContain('Description');
    expect(src).toContain('Qty');
    expect(src).toContain('Price');
    expect(src).toContain('Subtotal');
  });

});
