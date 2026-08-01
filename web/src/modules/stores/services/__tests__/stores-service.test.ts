import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockApiFetch = vi.fn();

vi.mock('$shared/api/http-client', () => ({
  apiFetch: (...args: unknown[]) => mockApiFetch(...args),
}));

describe('stores-service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('getStores', () => {
    it('builds basic query params', async () => {
      mockApiFetch.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [{ id: 1, name: 'Pusat' }], total: 1 }) });

      const { getStores } = await import('../stores-service');
      const result = await getStores({ limit: 20, offset: 0 });

      expect(mockApiFetch).toHaveBeenCalled();
      const url = mockApiFetch.mock.calls[0][0] as string;
      expect(url).toContain('/api/stores?');
      expect(url).toContain('limit=20');
      expect(url).toContain('offset=0');
      expect(result.data).toHaveLength(1);
      expect(result.total).toBe(1);
    });

    it('includes search and is_active filters', async () => {
      mockApiFetch.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [], total: 0 }) });

      const { getStores } = await import('../stores-service');
      await getStores({ limit: 10, offset: 0, search: 'cabang', is_active: false });

      const url = mockApiFetch.mock.calls[0][0] as string;
      expect(url).toContain('search=cabang');
      expect(url).toContain('is_active=false');
    });

    it('omits filters when not provided', async () => {
      mockApiFetch.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [], total: 0 }) });

      const { getStores } = await import('../stores-service');
      await getStores({ limit: 10, offset: 0 });

      const url = mockApiFetch.mock.calls[0][0] as string;
      expect(url).not.toContain('search=');
      expect(url).not.toContain('is_active=');
    });

    it('returns empty on error', async () => {
      mockApiFetch.mockResolvedValueOnce({ ok: false });

      const { getStores } = await import('../stores-service');
      const result = await getStores({ limit: 10, offset: 0 });
      expect(result.data).toEqual([]);
      expect(result.total).toBe(0);
    });
  });

  describe('getActiveStores', () => {
    it('hits /api/stores/active and maps data', async () => {
      mockApiFetch.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [{ id: 1, name: 'Pusat' }] }) });

      const { getActiveStores } = await import('../stores-service');
      const result = await getActiveStores();

      const url = mockApiFetch.mock.calls[0][0] as string;
      expect(url).toContain('/api/stores/active');
      expect(result).toHaveLength(1);
    });

    it('returns empty on error', async () => {
      mockApiFetch.mockResolvedValueOnce({ ok: false });

      const { getActiveStores } = await import('../stores-service');
      const result = await getActiveStores();
      expect(result).toEqual([]);
    });
  });

  describe('createStore', () => {
    it('sends POST with correct payload', async () => {
      mockApiFetch.mockResolvedValueOnce({ ok: true });

      const { createStore } = await import('../stores-service');
      const result = await createStore({ name: 'Cabang Bandung', address: 'Jl. Merdeka No. 1', phone: '022-123456' });

      expect(mockApiFetch).toHaveBeenCalledWith('/api/stores', {
        method: 'POST',
        body: JSON.stringify({ name: 'Cabang Bandung', address: 'Jl. Merdeka No. 1', phone: '022-123456' }),
      });
      expect(result).toBe(true);
    });
  });

  describe('updateStore', () => {
    it('sends PUT with partial payload', async () => {
      mockApiFetch.mockResolvedValueOnce({ ok: true });

      const { updateStore } = await import('../stores-service');
      const result = await updateStore(5, { name: 'Renamed', is_active: false });

      expect(mockApiFetch).toHaveBeenCalledWith('/api/stores/5', {
        method: 'PUT',
        body: JSON.stringify({ name: 'Renamed', is_active: false }),
      });
      expect(result).toBe(true);
    });
  });

  describe('deleteStore', () => {
    it('sends DELETE for the store id', async () => {
      mockApiFetch.mockResolvedValueOnce({ ok: true });

      const { deleteStore } = await import('../stores-service');
      const result = await deleteStore(9);

      expect(mockApiFetch).toHaveBeenCalledWith('/api/stores/9', { method: 'DELETE' });
      expect(result).toBe(true);
    });
  });
});
