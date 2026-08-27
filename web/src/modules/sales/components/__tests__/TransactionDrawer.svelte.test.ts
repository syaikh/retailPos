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

  it('imports printReceipt service', () => {
    expect(src).toContain("import { printReceiptWithToast } from '$shared/services/print-service'");
  });

  it('imports downloadInvoice', () => {
    expect(src).toContain("import { downloadInvoice } from '$modules/sales/lib/invoicePdf'");
  });

  it('imports apiClient for sale detail fetch', () => {
    expect(src).toContain("import apiClient from '$shared/api/http-client'");
  });

  it('imports toast', () => {
    expect(src).toContain("import { toast } from '$shared/stores/toast.svelte'");
  });

  it('imports i18n labels', () => {
    expect(src).toContain("import { labels } from '$shared/i18n'");
  });

  it('has detailLoading state for full sale fetch', () => {
    expect(src).toContain('let detailLoading = $state');
  });

  it('renders loading indicator when fetching detail', () => {
    expect(src).toContain('detailLoading');
  });

  it('fetches full sale detail on open', () => {
    expect(src).toContain('apiClient.get');
    expect(src).toContain('selectedTransaction.id');
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
    expect(src).toContain('{labels.printReceipt}');
  });

  it('renders Download Invoice button', () => {
    expect(src).toContain('{labels.downloadInvoice}');
  });

  it('renders Close button', () => {
    expect(src).toContain('{labels.close}');
  });

  it('renders transaction details heading', () => {
    expect(src).toContain('{labels.transactionDetails}');
  });

  it('renders localized metadata fields', () => {
    expect(src).toContain('{labels.invoiceNumber}');
    expect(src).toContain('{labels.dateAndTime}');
    expect(src).toContain('{labels.customer}');
    expect(src).toContain('{labels.paymentMethod}');
    expect(src).toContain('{labels.refLabel}');
    expect(src).toContain('labels.walkInGeneral');
  });

  it('hides the customer field in cross-cashier lookup (redacted) mode', () => {
    // In lookup (foreign-sale) mode the drawer renders a redacted summary, so
    // the customer row must be wrapped and only shown outside lookup mode.
    expect(src).toContain("{#if mode !== 'lookup'}");
    expect(src).toContain('{labels.customer}');
  });

  it('renders items table with localized headers', () => {
    expect(src).toContain('{labels.description}');
    expect(src).toContain('{labels.qty}');
    expect(src).toContain('{labels.price}');
    expect(src).toContain('{labels.subTotal}');
    expect(src).toContain('{labels.items}');
  });

  it('localizes totals, tax and savings labels', () => {
    expect(src).toContain('{labels.totalLabel}');
    expect(src).toContain('{labels.hemat}');
    expect(src).toContain('{labels.subTotal} ({labels.dpp})');
    expect(src).toContain('{labels.ppn}');
    expect(src).toContain('{labels.currencySymbol}');
  });

  it('localizes download toast messages', () => {
    expect(src).toContain('labels.toastInvoiceDownloaded');
    expect(src).toContain('labels.toastFailedToDownloadInvoice');
  });

  it('delegates silent print + failure toast to printReceiptWithToast (no fallback to preview)', () => {
    expect(src).toContain('printReceiptWithToast(');
    expect(src).not.toContain('toast.error(labels.printAgentUnavailable)');
  });

});
