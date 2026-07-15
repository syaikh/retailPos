import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockGet = vi.fn();
const mockPost = vi.fn();

vi.mock('$shared/api/http-client', () => ({
  default: {
    get: (...args: unknown[]) => mockGet(...args),
    post: (...args: unknown[]) => mockPost(...args),
  },
}));

describe('pos-service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('getPosProducts fetches active products', async () => {
    mockGet.mockResolvedValueOnce({ data: { data: [{ id: 1, name: 'Item' }], total: 1 } });

    const { getPosProducts } = await import('../pos-service');
    const result = await getPosProducts(20, 0, '');

    expect(mockGet).toHaveBeenCalledWith('/products?limit=20&offset=0&search=&status=active');
    expect(result.data).toHaveLength(1);
    expect(result.total).toBe(1);
  });

  it('getPosProducts includes search query', async () => {
    mockGet.mockResolvedValueOnce({ data: { data: [], total: 0 } });

    const { getPosProducts } = await import('../pos-service');
    await getPosProducts(20, 0, 'test');

    expect(mockGet).toHaveBeenCalledWith('/products?limit=20&offset=0&search=test&status=active');
  });

  it('getCustomers fetches customer list', async () => {
    mockGet.mockResolvedValueOnce({ data: { data: [{ id: 1, name: 'John' }] } });

    const { getCustomers } = await import('../pos-service');
    const result = await getCustomers(200);

    expect(mockGet).toHaveBeenCalledWith('/customers?limit=200');
    expect(result).toHaveLength(1);
  });

  it('searchCustomers builds query params', async () => {
    mockGet.mockResolvedValueOnce({ data: { data: [] } });

    const { searchCustomers } = await import('../pos-service');
    await searchCustomers('John', 10);

    expect(mockGet).toHaveBeenCalledWith('/customers', {
      params: { search: 'John', limit: 10 },
    });
  });

  it('createSale posts to /sales', async () => {
    mockPost.mockResolvedValueOnce({ data: { data: { id: 1, invoice_number: 'INV-001' } } });

    const { createSale } = await import('../pos-service');
    const payload = {
      items: [{ product_id: 1, quantity: 2, unit_price: 5000, subtotal: 10000 }],
      cashier_id: 1, store_id: null, subtotal: 10000, discount: 0, tax: 0,
      total_amount: 10000, payment_method: 'Cash', customer_id: null, status: 'completed',
    };
    const result = await createSale(payload) as { data: { data: { invoice_number: string } } };

    expect(mockPost).toHaveBeenCalledWith('/sales', payload);
    expect(result.data.data.invoice_number).toBe('INV-001');
  });

  it('getSaleById fetches sale detail', async () => {
    mockGet.mockResolvedValueOnce({ data: { data: { id: 1, items: [] } } });

    const { getSaleById } = await import('../pos-service');
    const result = await getSaleById(1);

    expect(mockGet).toHaveBeenCalledWith('/sales/1');
    expect(result).toEqual({ id: 1, items: [] });
  });

  it('getLastSale returns first sale from list', async () => {
    mockGet.mockResolvedValueOnce({ data: { data: [{ id: 1, invoice_number: 'INV-001' }] } });

    const { getLastSale } = await import('../pos-service');
    const result = await getLastSale();

    expect(result).toEqual({ id: 1, invoice_number: 'INV-001' });
  });

  it('getLastSale returns null on empty', async () => {
    mockGet.mockResolvedValueOnce({ data: { data: [] } });

    const { getLastSale } = await import('../pos-service');
    const result = await getLastSale();

    expect(result).toBeNull();
  });

  it('getLastSale uses Jakarta-aware endDate', async () => {
    mockGet.mockResolvedValueOnce({ data: { data: [] } });

    const { getLastSale } = await import('../pos-service');
    await getLastSale();

    const url = mockGet.mock.calls[0][0];
    expect(url).toMatch(/endDate=\d{4}-\d{2}-\d{2}$/);
    expect(url).not.toContain('endDate=undefined');
  });

  it('getLastSale handles single object response', async () => {
    mockGet.mockResolvedValueOnce({ data: { id: 1, invoice_number: 'INV-001' } });

    const { getLastSale } = await import('../pos-service');
    const result = await getLastSale();

    expect(result).toEqual({ id: 1, invoice_number: 'INV-001' });
  });
});
