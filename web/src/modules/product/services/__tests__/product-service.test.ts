import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockGet = vi.fn();
const mockPost = vi.fn();
const mockPut = vi.fn();
const mockDelete = vi.fn();

vi.mock('$shared/api/http-client', () => ({
  default: {
    get: (...args: unknown[]) => mockGet(...args),
    post: (...args: unknown[]) => mockPost(...args),
    put: (...args: unknown[]) => mockPut(...args),
    delete: (...args: unknown[]) => mockDelete(...args),
  },
}));

describe('product-service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('getProducts builds query params', async () => {
    mockGet.mockResolvedValueOnce({ data: { data: [{ id: 1, name: 'Test' }], total: 1 } });

    const { getProducts } = await import('../product-service');
    const result = await getProducts({ limit: 20, offset: 0, search: 'test' });

    expect(mockGet).toHaveBeenCalled();
    const url = mockGet.mock.calls[0][0] as string;
    expect(url).toContain('/products?');
    expect(url).toContain('limit=20');
    expect(url).toContain('offset=0');
    expect(url).toContain('search=test');
    expect(result.data).toHaveLength(1);
    expect(result.total).toBe(1);
  });

  it('getProducts includes status and category filters', async () => {
    mockGet.mockResolvedValueOnce({ data: { data: [], total: 0 } });

    const { getProducts } = await import('../product-service');
    await getProducts({ limit: 20, offset: 0, status: 'active', category: ['Food', 'Drink'] });

    const url = mockGet.mock.calls[0][0] as string;
    expect(url).toContain('status=active');
    expect(url).toContain('category=Food%2CDrink');
  });

  it('getProducts includes maxStock filter', async () => {
    mockGet.mockResolvedValueOnce({ data: { data: [], total: 0 } });

    const { getProducts } = await import('../product-service');
    await getProducts({ limit: 20, offset: 0, maxStock: 5 });

    const url = mockGet.mock.calls[0][0] as string;
    expect(url).toContain('maxStock=5');
  });

  it('createProduct posts to /products', async () => {
    mockPost.mockResolvedValueOnce({ status: 201 });

    const { createProduct } = await import('../product-service');
    await createProduct({
      name: 'Test Product', sku: 'TST-001', category_name: 'Food',
      category: 'Food', price: 10000, cost: 0, stock: 10,
      barcode: '', brand_id: null, unit_of_measure_id: null,
      tax_class_id: null, weight_grams: null, description: '', status: 'draft',
    });

    expect(mockPost).toHaveBeenCalledWith('/products', expect.objectContaining({
      name: 'Test Product',
      sku: 'TST-001',
    }));
  });

  it('updateProduct puts to /products/:id', async () => {
    mockPut.mockResolvedValueOnce({ status: 200 });

    const { updateProduct } = await import('../product-service');
    await updateProduct(1, {
      name: 'Updated', sku: 'TST-002', category_name: 'Food',
      category: 'Food', price: 15000, cost: 0, stock: 5,
      barcode: '', brand_id: null, unit_of_measure_id: null,
      tax_class_id: null, weight_grams: null, description: '', status: 'active',
    });

    expect(mockPut).toHaveBeenCalledWith('/products/1', expect.any(Object));
  });

  it('deleteProduct deletes to /products/:id', async () => {
    mockDelete.mockResolvedValueOnce({ status: 200 });

    const { deleteProduct } = await import('../product-service');
    await deleteProduct(1);

    expect(mockDelete).toHaveBeenCalledWith('/products/1');
  });

  it('bulkUpdateStatus posts to /products/bulk/status', async () => {
    mockPost.mockResolvedValueOnce({ status: 200 });

    const { bulkUpdateStatus } = await import('../product-service');
    await bulkUpdateStatus([1, 2], 'active');

    expect(mockPost).toHaveBeenCalledWith('/products/bulk/status', {
      ids: [1, 2], status: 'active',
    });
  });

  it('getCategories fetches categories', async () => {
    mockGet.mockResolvedValueOnce({ data: { data: [{ id: 1, name: 'Food' }] } });

    const { getCategories } = await import('../product-service');
    const result = await getCategories();

    expect(mockGet).toHaveBeenCalledWith('/categories');
    expect(result).toHaveLength(1);
    expect(result[0].name).toBe('Food');
  });

  it('getBrands fetches brands', async () => {
    mockGet.mockResolvedValueOnce({ data: { data: [{ id: 1, name: 'Brand A' }] } });

    const { getBrands } = await import('../product-service');
    const result = await getBrands();

    expect(mockGet).toHaveBeenCalledWith('/brands');
    expect(result).toHaveLength(1);
  });

  it('getTaxClasses fetches tax classes', async () => {
    mockGet.mockResolvedValueOnce({ data: { data: [{ id: 1, name: 'PPN', rate_percent: 11 }] } });

    const { getTaxClasses } = await import('../product-service');
    const result = await getTaxClasses();

    expect(mockGet).toHaveBeenCalledWith('/tax-classes');
    expect(result).toHaveLength(1);
  });

  it('getUnitsOfMeasure fetches units of measure', async () => {
    mockGet.mockResolvedValueOnce({ data: { data: [{ id: 1, code: 'PCS', name: 'Pieces' }] } });

    const { getUnitsOfMeasure } = await import('../product-service');
    const result = await getUnitsOfMeasure();

    expect(mockGet).toHaveBeenCalledWith('/units-of-measure');
    expect(result).toHaveLength(1);
  });

  it('getStockThresholds returns defaults on error', async () => {
    mockGet.mockRejectedValueOnce(new Error('Network error'));

    const { getStockThresholds } = await import('../product-service');
    const result = await getStockThresholds();

    expect(result).toEqual({ warning: 10, critical: 5 });
  });

  it('getStockThresholds returns server values', async () => {
    mockGet.mockResolvedValueOnce({ data: { warning: 15, critical: 8 } });

    const { getStockThresholds } = await import('../product-service');
    const result = await getStockThresholds();

    expect(result).toEqual({ warning: 15, critical: 8 });
  });

  it('adjustStock posts to /inventory/adjust', async () => {
    mockPost.mockResolvedValueOnce({ status: 200 });

    const { adjustStock } = await import('../product-service');
    await adjustStock(1, -5, 'Damaged goods');

    expect(mockPost).toHaveBeenCalledWith('/inventory/adjust', {
      product_id: 1,
      quantity_change: -5,
      notes: 'Damaged goods',
    });
  });

  it('getProductById returns product data', async () => {
    mockGet.mockResolvedValueOnce({ data: { id: 1, name: 'Test', sku: 'TST' } });

    const { getProductById } = await import('../product-service');
    const result = await getProductById(1);

    expect(mockGet).toHaveBeenCalledWith('/products/1');
    expect(result).toEqual({ id: 1, name: 'Test', sku: 'TST' });
  });

  it('getProductById returns null on error', async () => {
    mockGet.mockRejectedValueOnce(new Error('Not found'));

    const { getProductById } = await import('../product-service');
    const result = await getProductById(999);

    expect(result).toBeNull();
  });
});
