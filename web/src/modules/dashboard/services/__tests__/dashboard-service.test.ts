import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockFetch = vi.fn();
vi.mock('$shared/api/http-client', () => ({
  apiFetch: (...args: unknown[]) => mockFetch(...args),
}));

describe('dashboard-service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('getLiveStats returns parsed data', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({
        data: { todays_revenue: 500000, todays_sales: 10, total_products: 200, low_stock_count: 3 },
      }),
    });

    const { getLiveStats } = await import('../dashboard-service');
    const result = await getLiveStats();

    expect(mockFetch).toHaveBeenCalledWith('/api/dashboard/live');
    expect(result.todaysRevenue).toBe(500000);
    expect(result.todaysSales).toBe(10);
    expect(result.totalProducts).toBe(200);
    expect(result.lowStockCount).toBe(3);
  });

  it('getLiveStats returns defaults on API error', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network error'));

    const { getLiveStats } = await import('../dashboard-service');
    const result = await getLiveStats();

    expect(result.todaysRevenue).toBe(0);
    expect(result.todaysSales).toBe(0);
  });

  it('getLiveStats returns defaults on non-ok response', async () => {
    mockFetch.mockResolvedValueOnce({ ok: false });

    const { getLiveStats } = await import('../dashboard-service');
    const result = await getLiveStats();

    expect(result.todaysRevenue).toBe(0);
  });

  it('getLiveStats handles missing data field', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({}),
    });

    const { getLiveStats } = await import('../dashboard-service');
    const result = await getLiveStats();

    expect(result.todaysRevenue).toBe(0);
  });
});
