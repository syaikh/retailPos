import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'ReceiptPrintOverlay.svelte'), 'utf-8');
}

describe('ReceiptPrintOverlay.svelte source-structure guards', () => {
  const src = getSource();

  it('imports printReceipt store', () => {
    expect(src).toContain("import { printReceipt } from '$shared/stores/printReceipt.svelte'");
  });

  it('imports settingsStore for dynamic receipt branding', () => {
    expect(src).toContain("import { settingsStore } from '$shared/stores/settings.svelte'");
  });

  it('uses settingsStore for receipt store name', () => {
    expect(src).toContain('settingsStore.storeName');
  });

  it('imports formatDateTimeInJakarta utility', () => {
    expect(src).toContain("import { formatDateTimeInJakarta } from '$shared/utils/jakartaTime'");
  });

  it('renders conditional on printReceipt store', () => {
    expect(src).toContain('{#if $printReceipt}');
  });

  it('displays invoice number', () => {
    expect(src).toContain('{$printReceipt.invoice_number}');
  });

  it('displays created_at with Jakarta time formatting', () => {
    expect(src).toContain('formatDateTimeInJakarta($printReceipt.created_at || new Date().toISOString())');
  });

  it('conditionally renders customer name', () => {
    expect(src).toContain('{#if $printReceipt.customer_name}');
    expect(src).toContain('{$printReceipt.customer_name}');
  });

  it('renders items list with quantity and price', () => {
    expect(src).toContain('{#each $printReceipt.items as item}');
    expect(src).toContain('item.unit_price * item.quantity');
  });

  it('conditionally renders DPP and PPN', () => {
    expect(src).toContain('$printReceipt.subtotal_dpp');
    expect(src).toContain('$printReceipt.tax');
  });

  it('conditionally renders receipt header from settingsStore', () => {
    expect(src).toContain('settingsStore.receiptHeader');
  });

  it('conditionally renders receipt footer from settingsStore or defaults', () => {
    expect(src).toContain('settingsStore.receiptFooter');
  });

  it('displays total amount', () => {
    expect(src).toContain('{$printReceipt.total_amount');
  });

  it('displays payment section with method breakdown', () => {
    expect(src).toContain('{labels.payment}');
    expect(src).toContain('{#if $printReceipt.payments');
    expect(src).toContain('{#each $printReceipt.payments as p}');
    expect(src).toContain('p.amount.toLocaleString');
  });

  it('has Indonesian footer text', () => {
    expect(src).toContain('{labels.receiptThanks}');
    expect(src).toContain('{labels.receiptNoReturn}');
  });
});
