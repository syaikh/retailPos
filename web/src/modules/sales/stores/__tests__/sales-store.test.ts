import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockGetSalesHistory = vi.fn();

vi.mock('../../services/sales-service', () => ({
  getSalesHistory: (...args: unknown[]) => mockGetSalesHistory(...args),
}));

describe('sales-store', () => {
  let store: ReturnType<typeof import('../sales-store.svelte').useSalesStore>;

  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
  });

  it('returns expected API shape', async () => {
    const { useSalesStore } = await import('../sales-store.svelte');
    store = useSalesStore();
    expect(store).toHaveProperty('salesData');
    expect(store).toHaveProperty('total');
    expect(store).toHaveProperty('loading');
    expect(store).toHaveProperty('searchQuery');
    expect(store).toHaveProperty('sortBy');
    expect(store).toHaveProperty('sortDir');
    expect(store).toHaveProperty('pageSize');
    expect(store).toHaveProperty('limit');
    expect(store).toHaveProperty('offset');
    expect(store).toHaveProperty('cashierId');
    expect(store).toHaveProperty('load');
  });

  it('loads sales successfully', async () => {
    mockGetSalesHistory.mockResolvedValueOnce({ data: [{ id: 1 }], total: 1 });
    const { useSalesStore } = await import('../sales-store.svelte');
    store = useSalesStore();
    store.pageSize = 20;
    store.page = 0;
    await store.load({ limit: 20, offset: 0, sortBy: 'created_at', sortDir: 'DESC', startDate: '2026-01-01', endDate: '2026-06-29' });
    expect(mockGetSalesHistory).toHaveBeenCalled();
    expect(store.salesData).toHaveLength(1);
    expect(store.total).toBe(1);
    expect(store.loading).toBe(false);
  });

  it('sets loading false on error', async () => {
    mockGetSalesHistory.mockRejectedValueOnce(new Error('Network error'));
    const { useSalesStore } = await import('../sales-store.svelte');
    store = useSalesStore();
    store.pageSize = 20;
    store.page = 0;
    await store.load({ limit: 20, offset: 0, startDate: '2026-01-01', endDate: '2026-06-29' });
    expect(store.loading).toBe(false);
    expect(store.salesData).toEqual([]);
    expect(store.total).toBe(0);
  });

  it('provides currentFilters derived from state', async () => {
    const { useSalesStore } = await import('../sales-store.svelte');
    store = useSalesStore();
    store.startDate = '2026-01-01';
    store.endDate = '2026-06-29';
    store.pageSize = 50;
    store.page = 2;
    store.sortBy = 'invoice_number';
    store.sortDir = 'asc';
    store.searchQuery = 'test';
    store.cashierId = 5;
    const filters = store.currentFilters;
    expect(filters.limit).toBe(50);
    expect(filters.offset).toBe(100);
    expect(filters.sortBy).toBe('invoice_number');
    expect(filters.sortDir).toBe('asc');
    expect(filters.search).toBe('test');
    expect(filters.cashierId).toBe(5);
  });

  it('sets cashierId to undefined when null', async () => {
    const { useSalesStore } = await import('../sales-store.svelte');
    store = useSalesStore();
    store.cashierId = null;
    const filters = store.currentFilters;
    expect(filters.cashierId).toBeUndefined();
  });

  it('maps limit/offset to pageSize/page', async () => {
    const { useSalesStore } = await import('../sales-store.svelte');
    store = useSalesStore();
    store.limit = 50;
    store.offset = 100;
    expect(store.pageSize).toBe(50);
    expect(store.page).toBe(2);
  });
});
