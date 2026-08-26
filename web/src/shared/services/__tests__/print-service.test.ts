import { describe, it, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';
import { printConfig } from '$shared/stores/printConfig.svelte';
import { printReceipt as printReceiptStore } from '$shared/stores/printReceipt.svelte';
import { settingsStore } from '$shared/stores/settings.svelte';
import { printReceipt } from '../print-service';
import type { ReceiptData } from '$shared/stores/printReceipt.svelte';

const sample: ReceiptData = {
  invoice_number: 'INV-1',
  created_at: '2026-08-26 10:00',
  items: [{ name: 'Kopi', quantity: 2, unit_price: 5000 }],
  total_amount: 10000,
  paymentMethod: 'CASH',
  cashReceived: 10000,
  changeDue: 0,
};

const wait = (ms: number) => new Promise((r) => setTimeout(r, ms));

beforeEach(() => {
  printConfig.setMode('preview');
  printConfig.setAgentUrl('http://localhost:9123');
  settingsStore.storeName = 'Toko Test';
  printReceiptStore.set(null);
  vi.restoreAllMocks();
  (window as unknown as { print: () => void }).print = vi.fn();
  (globalThis as unknown as { fetch: unknown }).fetch = vi.fn();
});

describe('printReceipt service', () => {
  it('preview mode renders the overlay and opens the browser print dialog', async () => {
    await printReceipt(sample);
    expect(get(printReceiptStore)?.invoice_number).toBe('INV-1');
    await wait(400);
    expect((window as unknown as { print: () => void }).print).toHaveBeenCalledTimes(1);
  });

  it('silent mode POSTs to the agent and skips the dialog', async () => {
    printConfig.setMode('silent');
    const fetchMock = vi.fn().mockResolvedValue({ ok: true });
    (globalThis as unknown as { fetch: typeof fetch }).fetch = fetchMock as unknown as typeof fetch;

    await printReceipt(sample);

    expect(fetchMock).toHaveBeenCalledWith(
      'http://localhost:9123/print',
      expect.objectContaining({ method: 'POST' })
    );
    expect((window as unknown as { print: () => void }).print).not.toHaveBeenCalled();
    // overlay must stay empty in silent mode
    expect(get(printReceiptStore)).toBeNull();
  });

  it('silent mode falls back to preview when the agent is unreachable', async () => {
    printConfig.setMode('silent');
    (globalThis as unknown as { fetch: typeof fetch }).fetch = vi
      .fn()
      .mockRejectedValue(new Error('conn refused')) as unknown as typeof fetch;

    await printReceipt(sample);
    await wait(400);
    expect((window as unknown as { print: () => void }).print).toHaveBeenCalledTimes(1);
  });

  it('silent mode falls back to preview when the agent returns a non-ok status', async () => {
    printConfig.setMode('silent');
    (globalThis as unknown as { fetch: typeof fetch }).fetch = vi
      .fn()
      .mockResolvedValue({ ok: false, status: 500 }) as unknown as Response;

    await printReceipt(sample);
    await wait(400);
    expect((window as unknown as { print: () => void }).print).toHaveBeenCalledTimes(1);
    expect(get(printReceiptStore)?.invoice_number).toBe('INV-1');
  });

  it('silent mode sends the receipt payload with settings branding to the agent', async () => {
    printConfig.setMode('silent');
    let captured: string | undefined;
    const fetchMock = vi.fn().mockImplementation(async (_url: string, opts: RequestInit) => {
      captured = opts.body as string;
      return { ok: true };
    });
    (globalThis as unknown as { fetch: typeof fetch }).fetch = fetchMock as unknown as typeof fetch;

    await printReceipt(sample);

    expect(captured).toBeDefined();
    const body = JSON.parse(captured as string);
    expect(body.invoice).toBe('INV-1');
    expect(body.data).toEqual(sample);
    expect(body.branding.storeName).toBe('Toko Test');
  });
});
