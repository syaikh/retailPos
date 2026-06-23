import { describe, it, expect } from 'vitest';

describe('invoicePdf', () => {
  it('returns false for null transaction', async () => {
    const { downloadInvoice } = await import('../invoicePdf');
    const result = await downloadInvoice(null as any, (d: Date) => d.toISOString());
    expect(result).toBe(false);
  });

  it('exports downloadInvoice function', async () => {
    const mod = await import('../invoicePdf');
    expect(typeof mod.downloadInvoice).toBe('function');
  });
});
