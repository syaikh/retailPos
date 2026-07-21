import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockFetch = vi.fn();
vi.mock('$shared/api/http-client', () => ({
  apiFetch: (...args: unknown[]) => mockFetch(...args),
}));

vi.mock('$modules/auth', () => ({
  getAuthToken: () => 'mock-token',
}));

describe('sales-service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('getSalesHistory builds query params and returns data', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ data: [{ id: 1, invoice_number: 'INV-001' }], total: 1 }),
    });

    const { getSalesHistory } = await import('../sales-service');
    const result = await getSalesHistory({
      startDate: '2026-06-01',
      endDate: '2026-06-22',
      limit: 20,
      offset: 0,
    });

    expect(mockFetch).toHaveBeenCalled();
    const url = mockFetch.mock.calls[0][0] as string;
    expect(url).toContain('start_date=2026-06-01');
    expect(url).toContain('end_date=2026-06-22');
    expect(url).toContain('limit=20');
    expect(url).toContain('offset=0');
    expect(result.data).toHaveLength(1);
    expect(result.total).toBe(1);
  });

  it('getSalesHistory includes paymentMethods filter', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ data: [], total: 0 }),
    });

    const { getSalesHistory } = await import('../sales-service');
    await getSalesHistory({
      startDate: '2026-06-01',
      endDate: '2026-06-22',
      limit: 20,
      offset: 0,
      paymentMethods: ['cash', 'qris'],
    });

    const url = mockFetch.mock.calls[0][0] as string;
    expect(url).toContain('payment_methods=cash%2Cqris');
  });

  it('getSalesHistory includes minTotal/maxTotal filter', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ data: [], total: 0 }),
    });

    const { getSalesHistory } = await import('../sales-service');
    await getSalesHistory({
      startDate: '2026-06-01',
      endDate: '2026-06-22',
      limit: 20,
      offset: 0,
      minTotal: 10000,
      maxTotal: 500000,
    });

    const url = mockFetch.mock.calls[0][0] as string;
    expect(url).toContain('min_total=10000');
    expect(url).toContain('max_total=500000');
  });

  it('getSalesHistory includes cashierId filter', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ data: [], total: 0 }),
    });

    const { getSalesHistory } = await import('../sales-service');
    await getSalesHistory({
      startDate: '2026-06-01',
      endDate: '2026-06-22',
      limit: 20,
      offset: 0,
      cashierId: 5,
    });

    const url = mockFetch.mock.calls[0][0] as string;
    expect(url).toContain('cashier_id=5');
  });

  it('exportSales constructs export URL with auth header', async () => {
    globalThis.fetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      blob: () => Promise.resolve(new Blob()),
    });

    const { exportSales } = await import('../sales-service');
    const result = await exportSales('csv', {
      startDate: '2026-06-01',
      endDate: '2026-06-22',
      limit: 20,
      offset: 0,
    });

    expect(globalThis.fetch).toHaveBeenCalled();
    const callArgs = (globalThis.fetch as any).mock.calls[0];
    expect(callArgs[0]).toContain('/api/sales/export');
    expect(callArgs[0]).toContain('format=csv');
    expect(callArgs[1]?.headers?.Authorization).toBe('Bearer mock-token');
    expect(result).toBeInstanceOf(Blob);
  });
});
