import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

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

  afterEach(() => {
    vi.useRealTimers();
  });

  it('returns expected API shape', async () => {
    const { useSalesStore } = await import('../sales-store.svelte');
    store = useSalesStore();
    expect(store).toHaveProperty('salesData');
    expect(store).toHaveProperty('total');
    expect(store).toHaveProperty('loading');
    expect(store).toHaveProperty('limit');
    expect(store).toHaveProperty('offset');
    expect(store).toHaveProperty('searchQuery');
    expect(store).toHaveProperty('sortBy');
    expect(store).toHaveProperty('sortDir');
    expect(store).toHaveProperty('load');
  });

  it('loads sales successfully', async () => {
    mockGetSalesHistory.mockResolvedValueOnce({ data: [{ id: 1 }], total: 1 });
    const { useSalesStore } = await import('../sales-store.svelte');
    store = useSalesStore();
    await store.load({ limit: 20, offset: 0, sortBy: 'created_at', sortDir: 'DESC' });
    expect(mockGetSalesHistory).toHaveBeenCalled();
    expect(store.salesData).toHaveLength(1);
    expect(store.total).toBe(1);
    expect(store.loading).toBe(false);
  });

  it('sets loading false on error', async () => {
    mockGetSalesHistory.mockRejectedValueOnce(new Error('Network error'));
    const { useSalesStore } = await import('../sales-store.svelte');
    store = useSalesStore();
    await store.load({ limit: 20, offset: 0 });
    expect(store.loading).toBe(false);
    expect(store.salesData).toEqual([]);
    expect(store.total).toBe(0);
  });

  it('updates offset/limit/sort from filters', async () => {
    mockGetSalesHistory.mockResolvedValueOnce({ data: [], total: 0 });
    const { useSalesStore } = await import('../sales-store.svelte');
    store = useSalesStore();
    await store.load({ limit: 50, offset: 100, sortBy: 'invoice_number', sortDir: 'ASC' });
    expect(store.limit).toBe(50);
    expect(store.offset).toBe(100);
    expect(store.sortBy).toBe('invoice_number');
    expect(store.sortDir).toBe('ASC');
  });
});