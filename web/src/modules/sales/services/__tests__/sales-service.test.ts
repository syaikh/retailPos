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

  it('exportSales returns null when no token', async () => {
    vi.mocked(await import('$modules/auth')).getAuthToken = vi.fn(() => null);

    const { exportSales } = await import('../sales-service');
    const result = await exportSales('csv', {
      startDate: '2026-06-01', endDate: '2026-06-22', limit: 20, offset: 0,
    });

    expect(result).toBeNull();
  });

  it('exportSales returns null on non-ok response', async () => {
    globalThis.fetch = vi.fn().mockResolvedValueOnce({ ok: false });

    const { exportSales } = await import('../sales-service');
    const result = await exportSales('csv', {
      startDate: '2026-06-01', endDate: '2026-06-22', limit: 20, offset: 0,
    });

    expect(result).toBeNull();
  });

  it('getSalesHistory returns empty on non-ok response', async () => {
    mockFetch.mockResolvedValueOnce({ ok: false });

    const { getSalesHistory } = await import('../sales-service');
    const result = await getSalesHistory({
      startDate: '2026-06-01', endDate: '2026-06-22', limit: 20, offset: 0,
    });

    expect(result.data).toEqual([]);
    expect(result.total).toBe(0);
  });

  it('getPaymentMethods returns active payment methods', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ data: [{ code: 'cash', name: 'Cash', is_active: true }, { code: 'qris', name: 'QRIS', is_active: false }] }),
    });

    const { getPaymentMethods } = await import('../sales-service');
    const result = await getPaymentMethods();

    expect(result).toHaveLength(1);
    expect(result[0].code).toBe('cash');
  });

  it('getPaymentMethods returns empty array on error', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network error'));

    const { getPaymentMethods } = await import('../sales-service');
    const result = await getPaymentMethods();

    expect(result).toEqual([]);
  });

  it('getPaymentMethods returns empty array on non-ok', async () => {
    mockFetch.mockResolvedValueOnce({ ok: false });

    const { getPaymentMethods } = await import('../sales-service');
    const result = await getPaymentMethods();

    expect(result).toEqual([]);
  });

  it('getSaleById returns sale data', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ data: { id: 1, invoice_number: 'INV-001', total_amount: 100000 } }),
    });

    const { getSaleById } = await import('../sales-service');
    const result = await getSaleById(1);

    expect(result).not.toBeNull();
    expect(result?.id).toBe(1);
  });

  it('getSaleById returns null on error', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Not found'));

    const { getSaleById } = await import('../sales-service');
    const result = await getSaleById(1);

    expect(result).toBeNull();
  });

  it('getSaleById returns null on non-ok', async () => {
    mockFetch.mockResolvedValueOnce({ ok: false });

    const { getSaleById } = await import('../sales-service');
    const result = await getSaleById(1);

    expect(result).toBeNull();
  });

  it('getSaleById unwraps the data envelope and returns the sale', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ data: { id: 42, invoice_number: 'INV-042', total_amount: 75000 } }),
    });

    const { getSaleById } = await import('../sales-service');
    const result = await getSaleById(42);

    expect(result).not.toBeNull();
    expect(result?.id).toBe(42);
    expect(result?.invoice_number).toBe('INV-042');
  });

  it('getSaleById returns null when the data envelope is missing', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ status: 'ok' }),
    });

    const { getSaleById } = await import('../sales-service');
    const result = await getSaleById(1);

    expect(result).toBeNull();
  });

  it('getSalesLookup builds the lookup endpoint query and returns redacted summary', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () =>
        Promise.resolve({
          data: [
            {
              id: 1,
              invoice_number: 'INV-LOOKUP-1',
              cashier_id: 2,
              cashier_name: 'kasir2',
              total_amount: 50000,
              status: 'completed',
            },
          ],
          total: 1,
        }),
    });

    const { getSalesLookup } = await import('../sales-service');
    const result = await getSalesLookup({
      startDate: '2026-06-01',
      endDate: '2026-06-22',
      limit: 20,
      offset: 0,
    });

    expect(mockFetch).toHaveBeenCalled();
    const url = mockFetch.mock.calls[0][0] as string;
    expect(url).toContain('/api/sales/lookup');
    expect(url).toContain('start_date=2026-06-01');
    expect(url).toContain('end_date=2026-06-22');
    expect(result.data).toHaveLength(1);
    expect(result.data[0].invoice_number).toBe('INV-LOOKUP-1');
    expect(result.data[0].cashier_name).toBe('kasir2');
    expect(result.total).toBe(1);
  });

  it('getSalesLookup returns empty on non-ok response', async () => {
    mockFetch.mockResolvedValueOnce({ ok: false });

    const { getSalesLookup } = await import('../sales-service');
    const result = await getSalesLookup({
      startDate: '2026-06-01',
      endDate: '2026-06-22',
      limit: 20,
      offset: 0,
    });

    expect(result.data).toEqual([]);
    expect(result.total).toBe(0);
  });
});
