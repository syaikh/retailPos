import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

const mockGetCustomers = vi.fn();

vi.mock('../../services/customer-service', () => ({
  getCustomers: (...args: unknown[]) => mockGetCustomers(...args),
}));

describe('customer-store', () => {
  let store: ReturnType<typeof import('../customer-store.svelte').useCustomerStore>;

  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('returns expected API shape', async () => {
    const { useCustomerStore } = await import('../customer-store.svelte');
    store = useCustomerStore();
    expect(store).toHaveProperty('customers');
    expect(store).toHaveProperty('total');
    expect(store).toHaveProperty('loading');
    expect(store).toHaveProperty('limit');
    expect(store).toHaveProperty('offset');
    expect(store).toHaveProperty('searchQuery');
    expect(store).toHaveProperty('statusFilter');
    expect(store).toHaveProperty('selectedIds');
    expect(store).toHaveProperty('load');
    expect(store).toHaveProperty('clearSelection');
  });

  it('loads customers successfully', async () => {
    mockGetCustomers.mockResolvedValueOnce({ data: [{ id: 1, name: 'John' }], total: 1 });
    const { useCustomerStore } = await import('../customer-store.svelte');
    store = useCustomerStore();
    await store.load();
    expect(mockGetCustomers).toHaveBeenCalled();
    expect(store.customers).toHaveLength(1);
    expect(store.total).toBe(1);
    expect(store.loading).toBe(false);
  });

  it('loads customers with filters', async () => {
    mockGetCustomers.mockResolvedValueOnce({ data: [], total: 0 });
    const { useCustomerStore } = await import('../customer-store.svelte');
    store = useCustomerStore();
    store.searchQuery = 'test';
    await store.load();
    expect(mockGetCustomers).toHaveBeenCalledWith({ limit: 20, offset: 0, search: 'test' });
  });

  it('sets loading false on error', async () => {
    mockGetCustomers.mockRejectedValueOnce(new Error('Network error'));
    const { useCustomerStore } = await import('../customer-store.svelte');
    store = useCustomerStore();
    await store.load();
    expect(store.loading).toBe(false);
    expect(store.customers).toEqual([]);
    expect(store.total).toBe(0);
  });

  it('sets active status filter', async () => {
    mockGetCustomers.mockResolvedValueOnce({ data: [], total: 0 });
    const { useCustomerStore } = await import('../customer-store.svelte');
    store = useCustomerStore();
    store.statusFilter = 'active';
    await store.load();
    expect(mockGetCustomers).toHaveBeenCalledWith({ limit: 20, offset: 0, isActive: 'true' });
  });

  it('sets inactive status filter', async () => {
    mockGetCustomers.mockResolvedValueOnce({ data: [], total: 0 });
    const { useCustomerStore } = await import('../customer-store.svelte');
    store = useCustomerStore();
    store.statusFilter = 'inactive';
    await store.load();
    expect(mockGetCustomers).toHaveBeenCalledWith({ limit: 20, offset: 0, isActive: 'false' });
  });

  it('clears selection', async () => {
    const { useCustomerStore } = await import('../customer-store.svelte');
    store = useCustomerStore();
    store.selectedIds = new Set([1, 2, 3]);
    store.clearSelection();
    expect(store.selectedIds).toEqual(new Set());
  });
});