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

describe('inventory-service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('getLocationStock', () => {
    it('requests rack stock for a product and returns data rows', async () => {
      mockGet.mockResolvedValueOnce({
        data: { data: [{ product_id: 7, location_id: 3, quantity: 4 }] },
      });

      const { getLocationStock } = await import('../inventory-service');
      const rows = await getLocationStock(7);

      expect(mockGet).toHaveBeenCalledWith('/inventory/locations', { params: { product_id: 7 } });
      expect(rows).toEqual([{ product_id: 7, location_id: 3, quantity: 4 }]);
    });

    it('includes location_id filter when provided', async () => {
      mockGet.mockResolvedValueOnce({ data: { data: [] } });

      const { getLocationStock } = await import('../inventory-service');
      await getLocationStock(7, 3);

      expect(mockGet).toHaveBeenCalledWith('/inventory/locations', {
        params: { product_id: 7, location_id: 3 },
      });
    });

    it('defaults to empty array when response has no data', async () => {
      mockGet.mockResolvedValueOnce({ data: {} });

      const { getLocationStock } = await import('../inventory-service');
      const rows = await getLocationStock(7);

      expect(rows).toEqual([]);
    });
  });

  describe('setLocationStock', () => {
    it('posts the set payload to /inventory/locations', async () => {
      mockPost.mockResolvedValueOnce({});

      const { setLocationStock } = await import('../inventory-service');
      await setLocationStock({ product_id: 7, location_id: 3, quantity: 12 });

      expect(mockPost).toHaveBeenCalledWith('/inventory/locations', {
        product_id: 7,
        location_id: 3,
        quantity: 12,
      });
    });
  });

  describe('transferLocationStock', () => {
    it('posts the transfer payload to /inventory/locations/transfer', async () => {
      mockPost.mockResolvedValueOnce({});

      const { transferLocationStock } = await import('../inventory-service');
      await transferLocationStock({
        product_id: 7,
        from_location_id: 3,
        to_location_id: 4,
        quantity: 5,
      });

      expect(mockPost).toHaveBeenCalledWith('/inventory/locations/transfer', {
        product_id: 7,
        from_location_id: 3,
        to_location_id: 4,
        quantity: 5,
      });
    });
  });

  describe('adjustStock', () => {
    it('posts the adjustment payload to /inventory/adjust', async () => {
      mockPost.mockResolvedValueOnce({});

      const { adjustStock } = await import('../inventory-service');
      await adjustStock({ product_id: 7, quantity_change: 5, notes: 'restock' });

      expect(mockPost).toHaveBeenCalledWith('/inventory/adjust', {
        product_id: 7,
        quantity_change: 5,
        notes: 'restock',
      });
    });
  });
});
