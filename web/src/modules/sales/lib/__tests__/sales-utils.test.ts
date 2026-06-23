import { describe, it, expect } from 'vitest';

describe('sales-utils', () => {
  it('statusVariant returns correct variant for each status', async () => {
    const mod = await import('../sales-utils');
    const { statusVariant } = mod;
    expect(statusVariant('completed')).toBe('success');
    expect(statusVariant('refunded')).toBe('danger');
    expect(statusVariant('pending')).toBe('warning');
    expect(statusVariant('unknown')).toBe('warning');
  });

  it('getPaymentMethodVariant returns correct variant', async () => {
    const mod = await import('../sales-utils');
    const { getPaymentMethodVariant } = mod;
    expect(getPaymentMethodVariant('cash')).toBe('success');
    expect(getPaymentMethodVariant('qris')).toBe('default');
    expect(getPaymentMethodVariant('card')).toBe('primary');
    expect(getPaymentMethodVariant('transfer')).toBe('muted');
    expect(getPaymentMethodVariant('')).toBe('muted');
    expect(getPaymentMethodVariant('unknown')).toBe('muted');
  });

  it('sanitizeSearch strips INV- prefix', async () => {
    const mod = await import('../sales-utils');
    const { sanitizeSearch } = mod;
    expect(sanitizeSearch('INV-00123')).toBe('00123');
    expect(sanitizeSearch('inv-00123')).toBe('00123');
    expect(sanitizeSearch('00123')).toBe('00123');
    expect(sanitizeSearch('  INV-456  ')).toBe('456');
  });
});
