import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockFetch = vi.fn();
vi.mock('$shared/api/http-client', () => ({
  apiFetch: (...args: unknown[]) => mockFetch(...args),
}));

describe('reporting-service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('getAvailableYears returns years list', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ data: [2024, 2025, 2026] }),
    });

    const { getAvailableYears } = await import('../reporting-service');
    const result = await getAvailableYears();

    expect(mockFetch).toHaveBeenCalledWith('/api/dashboard/years');
    expect(result).toEqual([2024, 2025, 2026]);
  });

  it('getAvailableYears returns empty on error', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network error'));

    const { getAvailableYears } = await import('../reporting-service');
    const result = await getAvailableYears();

    expect(result).toEqual([]);
  });

  it('getChartData builds correct URL with prev params', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ current: [{ total: 100 }], previous: [{ total: 80 }] }),
    });

    const { getChartData } = await import('../reporting-service');
    const result = await getChartData('/api/dashboard/chart', '2026-06-01', '2026-06-22', '2026-05-25', '2026-06-15');

    const url = mockFetch.mock.calls[0][0] as string;
    expect(url).toContain('/api/dashboard/chart?');
    expect(url).toContain('startDate=2026-06-01');
    expect(url).toContain('endDate=2026-06-22');
    expect(url).toContain('prevStart=2026-05-25');
    expect(url).toContain('prevEnd=2026-06-15');
    expect(result?.current).toHaveLength(1);
    expect(result?.previous).toHaveLength(1);
  });

  it('getChartData works without prev params', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ data: [{ total: 50 }], previous: [] }),
    });

    const { getChartData } = await import('../reporting-service');
    const result = await getChartData('/api/dashboard/chart/monthly', '2026-01-01', '2026-12-31');

    const url = mockFetch.mock.calls[0][0] as string;
    expect(url).not.toContain('prevStart');
    expect(result?.current).toHaveLength(1);
  });

  it('getChartData returns null on error', async () => {
    mockFetch.mockResolvedValueOnce({ ok: false });

    const { getChartData } = await import('../reporting-service');
    const result = await getChartData('/api/dashboard/chart', '2026-01-01', '2026-01-31');

    expect(result).toBeNull();
  });

  it('getComparison fetches comparison data', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ data: { current_revenue: 100000 }, meta: { is_partial: false } }),
    });

    const { getComparison } = await import('../reporting-service');
    const result = await getComparison('daily', 'completed', '2026-06-22');

    expect(mockFetch).toHaveBeenCalledWith('/api/dashboard/comparison?period=daily&mode=completed&date=2026-06-22');
    expect(result?.data.current_revenue).toBe(100000);
    expect(result?.meta.is_partial).toBe(false);
  });

  it('getComparison returns null on error', async () => {
    mockFetch.mockResolvedValueOnce({ ok: false });

    const { getComparison } = await import('../reporting-service');
    const result = await getComparison('daily', 'completed', '2026-06-22');

    expect(result).toBeNull();
  });

  it('exportDashboard sends POST with form data', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      blob: () => Promise.resolve(new Blob()),
    });

    const { exportDashboard } = await import('../reporting-service');
    const result = await exportDashboard('daily', 'completed', '2026-06-22');

    expect(mockFetch).toHaveBeenCalledWith('/api/dashboard/export', {
      method: 'POST',
      body: expect.any(FormData),
    });
    expect(result).toBeInstanceOf(Blob);
  });

  it('exportDashboard returns null on fail', async () => {
    mockFetch.mockResolvedValueOnce({ ok: false });

    const { exportDashboard } = await import('../reporting-service');
    const result = await exportDashboard('daily', 'completed', '2026-06-22');

    expect(result).toBeNull();
  });
});
