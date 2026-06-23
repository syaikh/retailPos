import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

const mockGetProducts = vi.fn();
const mockGetCategories = vi.fn();
const mockGetBrands = vi.fn();
const mockGetTaxClasses = vi.fn();
const mockGetUnitsOfMeasure = vi.fn();
const mockGetStockThresholds = vi.fn();

vi.mock('../../services/product-service', () => ({
  getProducts: (...args: unknown[]) => mockGetProducts(...args),
  getCategories: (...args: unknown[]) => mockGetCategories(...args),
  getBrands: (...args: unknown[]) => mockGetBrands(...args),
  getTaxClasses: (...args: unknown[]) => mockGetTaxClasses(...args),
  getUnitsOfMeasure: (...args: unknown[]) => mockGetUnitsOfMeasure(...args),
  getStockThresholds: (...args: unknown[]) => mockGetStockThresholds(...args),
}));

describe('product-store', () => {
  let store: ReturnType<typeof import('../product-store.svelte').useProductStore>;

  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('returns expected API shape', async () => {
    const { useProductStore } = await import('../product-store.svelte');
    store = useProductStore();
    expect(store).toHaveProperty('products');
    expect(store).toHaveProperty('total');
    expect(store).toHaveProperty('loading');
    expect(store).toHaveProperty('searchQuery');
    expect(store).toHaveProperty('selectedCategories');
    expect(store).toHaveProperty('categories');
    expect(store).toHaveProperty('brands');
    expect(store).toHaveProperty('taxClasses');
    expect(store).toHaveProperty('unitsOfMeasure');
    expect(store).toHaveProperty('warningThreshold');
    expect(store).toHaveProperty('criticalThreshold');
    expect(store).toHaveProperty('loadProducts');
    expect(store).toHaveProperty('loadMasterData');
    expect(store).toHaveProperty('loadThresholds');
    expect(store).toHaveProperty('initialize');
    expect(store).toHaveProperty('clearSelection');
  });

  it('loads products successfully', async () => {
    mockGetProducts.mockResolvedValueOnce({ data: [{ id: 1, name: 'Product1' }], total: 1 });
    const { useProductStore } = await import('../product-store.svelte');
    store = useProductStore();
    await store.loadProducts();
    expect(mockGetProducts).toHaveBeenCalled();
    expect(store.products).toHaveLength(1);
    expect(store.total).toBe(1);
    expect(store.loading).toBe(false);
  });

  it('sets loading false on error', async () => {
    mockGetProducts.mockRejectedValueOnce(new Error('Network error'));
    const { useProductStore } = await import('../product-store.svelte');
    store = useProductStore();
    await store.loadProducts();
    expect(store.loading).toBe(false);
    expect(store.products).toEqual([]);
    expect(store.total).toBe(0);
  });

  it('sets searchQuery via setter', async () => {
    const { useProductStore } = await import('../product-store.svelte');
    store = useProductStore();
    store.searchQuery = 'test search';
    expect(store.searchQuery).toBe('test search');
  });

  it('loads master data', async () => {
    mockGetCategories.mockResolvedValueOnce([{ id: 1, name: 'Cat1' }]);
    mockGetBrands.mockResolvedValueOnce([{ id: 1, name: 'Brand1' }]);
    mockGetTaxClasses.mockResolvedValueOnce([{ id: 1, name: 'Tax1' }]);
    mockGetUnitsOfMeasure.mockResolvedValueOnce([{ id: 1, name: 'UOM1' }]);
    const { useProductStore } = await import('../product-store.svelte');
    store = useProductStore();
    await store.loadMasterData();
    expect(mockGetCategories).toHaveBeenCalled();
    expect(store.categoryObjects).toHaveLength(1);
    expect(store.brands).toHaveLength(1);
    expect(store.taxClasses).toHaveLength(1);
    expect(store.unitsOfMeasure).toHaveLength(1);
  });

  it('loads thresholds', async () => {
    mockGetStockThresholds.mockResolvedValueOnce({ warning: 15, critical: 8 });
    const { useProductStore } = await import('../product-store.svelte');
    store = useProductStore();
    await store.loadThresholds();
    expect(store.warningThreshold).toBe(15);
    expect(store.criticalThreshold).toBe(8);
  });

  it('clears selection', async () => {
    const { useProductStore } = await import('../product-store.svelte');
    store = useProductStore();
    store.selectedIds = new Set([1, 2]);
    store.clearSelection();
    expect(store.selectedIds).toEqual(new Set());
  });
});