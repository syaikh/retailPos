import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockApiFetch = vi.fn();

vi.mock('$shared/api/http-client', () => ({
  apiFetch: (...args: unknown[]) => mockApiFetch(...args),
}));

describe('supplier-service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('getSuppliers', () => {
    it('builds basic query params', async () => {
      mockApiFetch.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [{ id: 1, name: 'PT Maju' }], total: 1 }) });

      const { getSuppliers } = await import('../supplier-service');
      const result = await getSuppliers({ limit: 20, offset: 0 });

      expect(mockApiFetch).toHaveBeenCalled();
      const url = mockApiFetch.mock.calls[0][0] as string;
      expect(url).toContain('limit=20');
      expect(url).toContain('offset=0');
      expect(result.data).toHaveLength(1);
      expect(result.total).toBe(1);
    });

    it('includes search and is_active filters', async () => {
      mockApiFetch.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [], total: 0 }) });

      const { getSuppliers } = await import('../supplier-service');
      await getSuppliers({ limit: 10, offset: 0, search: 'Maju', is_active: true });

      const url = mockApiFetch.mock.calls[0][0] as string;
      expect(url).toContain('search=Maju');
      expect(url).toContain('is_active=true');
    });

    it('includes sort_by and sort_dir', async () => {
      mockApiFetch.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [], total: 0 }) });

      const { getSuppliers } = await import('../supplier-service');
      await getSuppliers({ limit: 10, offset: 0, sort_by: 'name', sort_dir: 'asc' });

      const url = mockApiFetch.mock.calls[0][0] as string;
      expect(url).toContain('sort_by=name');
      expect(url).toContain('sort_dir=asc');
    });

    it('returns empty on error', async () => {
      mockApiFetch.mockResolvedValueOnce({ ok: false });

      const { getSuppliers } = await import('../supplier-service');
      const result = await getSuppliers({ limit: 10, offset: 0 });
      expect(result.data).toEqual([]);
      expect(result.total).toBe(0);
    });
  });

  describe('getSupplier', () => {
    it('returns single supplier', async () => {
      mockApiFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ data: { id: 1, name: 'PT Maju', code: 'SUP-001' } }),
      });

      const { getSupplier } = await import('../supplier-service');
      const result = await getSupplier(1);

      expect(mockApiFetch).toHaveBeenCalledWith('/api/suppliers/1');
      expect(result).toEqual({ id: 1, name: 'PT Maju', code: 'SUP-001' });
    });

    it('returns null on error', async () => {
      mockApiFetch.mockResolvedValueOnce({ ok: false });

      const { getSupplier } = await import('../supplier-service');
      const result = await getSupplier(99);
      expect(result).toBeNull();
    });
  });

  describe('createSupplier', () => {
    it('sends POST with correct payload', async () => {
      mockApiFetch.mockResolvedValueOnce({ ok: true });

      const { createSupplier } = await import('../supplier-service');
      const result = await createSupplier({
        name: 'PT Baru',
        code: 'SUP-NEW',
        is_active: true,
      });

      expect(result).toBe(true);
      expect(mockApiFetch).toHaveBeenCalledWith('/api/suppliers', expect.objectContaining({
        method: 'POST',
      }));
      const body = JSON.parse(mockApiFetch.mock.calls[0][1].body);
      expect(body.name).toBe('PT Baru');
      expect(body.code).toBe('SUP-NEW');
      expect(body.is_active).toBe(true);
    });
  });

  describe('updateSupplier', () => {
    it('sends PUT with correct payload', async () => {
      mockApiFetch.mockResolvedValueOnce({ ok: true });

      const { updateSupplier } = await import('../supplier-service');
      const result = await updateSupplier(1, {
        name: 'PT Update',
        is_active: false,
      });

      expect(result).toBe(true);
      expect(mockApiFetch).toHaveBeenCalledWith('/api/suppliers/1', expect.objectContaining({
        method: 'PUT',
      }));
      const body = JSON.parse(mockApiFetch.mock.calls[0][1].body);
      expect(body.name).toBe('PT Update');
      expect(body.is_active).toBe(false);
    });
  });

  describe('deleteSupplier', () => {
    it('sends DELETE request', async () => {
      mockApiFetch.mockResolvedValueOnce({ ok: true });

      const { deleteSupplier } = await import('../supplier-service');
      const result = await deleteSupplier(5);

      expect(result).toBe(true);
      expect(mockApiFetch).toHaveBeenCalledWith('/api/suppliers/5', { method: 'DELETE' });
    });
  });

  describe('getSuppliersByProduct', () => {
    it('returns suppliers for product', async () => {
      mockApiFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({
          data: [{ id: 1, product_id: 10, supplier_id: 1, unit_cost: 50000, lead_time_days: 3, is_preferred: true }],
        }),
      });

      const { getSuppliersByProduct } = await import('../supplier-service');
      const result = await getSuppliersByProduct(10);

      expect(mockApiFetch).toHaveBeenCalledWith('/api/products/10/suppliers');
      expect(result).toHaveLength(1);
      expect(result[0].unit_cost).toBe(50000);
      expect(result[0].is_preferred).toBe(true);
    });

    it('returns empty on error', async () => {
      mockApiFetch.mockResolvedValueOnce({ ok: false });

      const { getSuppliersByProduct } = await import('../supplier-service');
      const result = await getSuppliersByProduct(99);
      expect(result).toEqual([]);
    });
  });

  describe('linkProduct', () => {
    it('sends POST with correct payload', async () => {
      mockApiFetch.mockResolvedValueOnce({ ok: true });

      const { linkProduct } = await import('../supplier-service');
      const result = await linkProduct(1, {
        product_id: 10,
        unit_cost: 50000,
        lead_time_days: 7,
        is_preferred: true,
      });

      expect(result).toBe(true);
      expect(mockApiFetch).toHaveBeenCalledWith('/api/suppliers/1/products', expect.objectContaining({
        method: 'POST',
      }));
      const body = JSON.parse(mockApiFetch.mock.calls[0][1].body);
      expect(body.product_id).toBe(10);
      expect(body.unit_cost).toBe(50000);
      expect(body.lead_time_days).toBe(7);
      expect(body.is_preferred).toBe(true);
    });
  });

  describe('unlinkProduct', () => {
    it('sends DELETE request', async () => {
      mockApiFetch.mockResolvedValueOnce({ ok: true });

      const { unlinkProduct } = await import('../supplier-service');
      const result = await unlinkProduct(1, 10);

      expect(result).toBe(true);
      expect(mockApiFetch).toHaveBeenCalledWith('/api/suppliers/1/products/10', { method: 'DELETE' });
    });
  });
});
