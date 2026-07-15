import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { ReceiptData } from '../printReceipt.svelte.ts';
import type { printReceipt as PrintReceiptStore } from '../printReceipt.svelte.ts';

describe('printReceipt store', () => {
  let printReceipt: typeof PrintReceiptStore;

  beforeEach(() => {
    vi.resetModules();
  });

  it('returns writable store with set method', async () => {
    const { printReceipt: p } = await import('../printReceipt.svelte');
    printReceipt = p;
    expect(printReceipt).toHaveProperty('set');
    expect(printReceipt).toHaveProperty('subscribe');
    expect(printReceipt).toHaveProperty('update');
  });

  it('initial value is null', async () => {
    const { printReceipt: p } = await import('../printReceipt.svelte');
    printReceipt = p;
    let value: ReceiptData | null = 'not null' as unknown as ReceiptData;
    printReceipt.subscribe((v) => { value = v; })();
    expect(value).toBeNull();
  });

  it('set updates receipt data', async () => {
    const { printReceipt: p } = await import('../printReceipt.svelte');
    printReceipt = p;
    const data = {
      invoice_number: 'INV-001',
      created_at: '2024-01-01',
      items: [],
      total_amount: 1000,
      paymentMethod: 'Cash',
      cashReceived: 1000,
      changeDue: 0,
    };
    printReceipt.set(data);
    let value: ReceiptData | null = null;
    printReceipt.subscribe((v) => { value = v; })();
    expect(value).toEqual(data);
  });

  it('set null clears receipt data', async () => {
    const { printReceipt: p } = await import('../printReceipt.svelte');
    printReceipt = p;
    printReceipt.set({ invoice_number: 'X', created_at: '', items: [], total_amount: 0, paymentMethod: '', cashReceived: 0, changeDue: 0 });
    printReceipt.set(null);
    let value: ReceiptData | null = 'not null' as unknown as ReceiptData;
    printReceipt.subscribe((v) => { value = v; })();
    expect(value).toBeNull();
  });
});