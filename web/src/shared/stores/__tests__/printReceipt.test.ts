import { describe, it, expect, vi, beforeEach } from 'vitest';

describe('printReceipt store', () => {
  let printReceipt: typeof import('../printReceipt.svelte').printReceipt;

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
    let value: typeof import('../printReceipt.svelte').ReceiptData | null = 'not null';
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
    let value: typeof import('../printReceipt.svelte').ReceiptData | null = null;
    printReceipt.subscribe((v) => { value = v; })();
    expect(value).toEqual(data);
  });

  it('set null clears receipt data', async () => {
    const { printReceipt: p } = await import('../printReceipt.svelte');
    printReceipt = p;
    printReceipt.set({ invoice_number: 'X', created_at: '', items: [], total_amount: 0, paymentMethod: '', cashReceived: 0, changeDue: 0 });
    printReceipt.set(null);
    let value: typeof import('../printReceipt.svelte').ReceiptData | null = 'not null';
    printReceipt.subscribe((v) => { value = v; })();
    expect(value).toBeNull();
  });
});