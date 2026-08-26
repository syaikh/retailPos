import { describe, it, expect, vi, beforeEach } from 'vitest';
import { printConfig } from '$shared/stores/printConfig.svelte';
import { settingsStore } from '$shared/stores/settings.svelte';
import { toast } from '$shared/stores/toast.svelte';
import { labels } from '$shared/i18n';
import { printReceiptWithToast } from '../print-service';
import type { ReceiptData } from '$shared/stores/printReceipt.svelte';

vi.mock('$shared/stores/toast.svelte', () => ({
  toast: { error: vi.fn(), success: vi.fn(), info: vi.fn(), warning: vi.fn() },
}));

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
  printConfig.setMode('silent');
  printConfig.setAgentUrl('http://localhost:9123');
  settingsStore.storeName = 'Toko Test';
  vi.clearAllMocks();
  (globalThis as unknown as { fetch: unknown }).fetch = vi.fn();
  (window as unknown as { print: () => void }).print = vi.fn();
});

describe('printReceiptWithToast (silent failure notification)', () => {
  it('surfaces a non-blocking error toast when the print agent is unreachable (no fallback to preview)', async () => {
    (globalThis as unknown as { fetch: typeof fetch }).fetch = vi
      .fn()
      .mockRejectedValue(new Error('conn refused')) as unknown as typeof fetch;

    const res = await printReceiptWithToast(sample);

    // silent failure, never falls back to the browser dialog
    expect(res.ok).toBe(false);
    expect(res.mode).toBe('silent');
    expect((window as unknown as { print: () => void }).print).not.toHaveBeenCalled();
    // the Retry/Dismiss notification is shown exactly once
    expect(toast.error).toHaveBeenCalledTimes(1);
    expect(toast.error).toHaveBeenCalledWith(labels.printAgentUnavailable);
  });

  it('does NOT show the error toast when printing succeeds', async () => {
    (globalThis as unknown as { fetch: typeof fetch }).fetch = vi
      .fn()
      .mockResolvedValue({ ok: true }) as unknown as typeof fetch;

    const res = await printReceiptWithToast(sample);

    expect(res.ok).toBe(true);
    expect(toast.error).not.toHaveBeenCalled();
    await wait(400);
    expect((window as unknown as { print: () => void }).print).not.toHaveBeenCalled();
  });
});
