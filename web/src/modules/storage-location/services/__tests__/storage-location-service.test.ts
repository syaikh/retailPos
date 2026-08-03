import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockApiClient = {
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  delete: vi.fn(),
};

vi.mock('$shared/api/http-client', () => ({
  default: mockApiClient,
}));

beforeEach(() => {
  vi.clearAllMocks();
});

describe('storage-location-service', () => {
  describe('getStorageLocations', () => {
    it('fetches with basic params', async () => {
      mockApiClient.get.mockResolvedValueOnce({ data: { data: [{ id: 1, code: 'RAK-A-01' }], total: 1 } });
      const { getStorageLocations } = await import('../storage-location-service');
      const result = await getStorageLocations({ limit: 20, offset: 0 });
      expect(mockApiClient.get).toHaveBeenCalledWith('/storage-locations', { params: { limit: 20, offset: 0, search: undefined } });
      expect(result.data).toHaveLength(1);
      expect(result.total).toBe(1);
    });

    it('includes search and is_active filters', async () => {
      mockApiClient.get.mockResolvedValueOnce({ data: { data: [], total: 0 } });
      const { getStorageLocations } = await import('../storage-location-service');
      await getStorageLocations({ limit: 10, offset: 0, search: 'rak', is_active: true });
      const params = mockApiClient.get.mock.calls[0][1].params;
      expect(params.search).toBe('rak');
      expect(params.is_active).toBe('true');
    });

    it('sets is_active to false when is_active is false', async () => {
      mockApiClient.get.mockResolvedValueOnce({ data: { data: [], total: 0 } });
      const { getStorageLocations } = await import('../storage-location-service');
      await getStorageLocations({ limit: 10, offset: 0, is_active: false });
      const params = mockApiClient.get.mock.calls[0][1].params;
      expect(params.is_active).toBe('false');
    });
  });

  describe('getStorageLocation', () => {
    it('fetches single location by id', async () => {
      mockApiClient.get.mockResolvedValueOnce({ data: { data: { id: 1, code: 'RAK-A-01' } } });
      const { getStorageLocation } = await import('../storage-location-service');
      const result = await getStorageLocation(1);
      expect(mockApiClient.get).toHaveBeenCalledWith('/storage-locations/1');
      expect(result?.code).toBe('RAK-A-01');
    });
  });

  describe('createStorageLocation', () => {
    it('sends POST with payload', async () => {
      mockApiClient.post.mockResolvedValueOnce({ ok: true });
      const { createStorageLocation } = await import('../storage-location-service');
      await createStorageLocation({ code: 'RAK-A-01', name: 'Rak A', warehouse_id: 1 });
      expect(mockApiClient.post).toHaveBeenCalledWith('/storage-locations', { code: 'RAK-A-01', name: 'Rak A', warehouse_id: 1 });
    });

    it('sends POST with store scope', async () => {
      mockApiClient.post.mockResolvedValueOnce({ ok: true });
      const { createStorageLocation } = await import('../storage-location-service');
      await createStorageLocation({ code: 'SL-01', name: 'Toko Utama', store_id: 2 });
      expect(mockApiClient.post).toHaveBeenCalledWith('/storage-locations', { code: 'SL-01', name: 'Toko Utama', store_id: 2 });
    });
  });

  describe('updateStorageLocation', () => {
    it('sends PUT with payload', async () => {
      mockApiClient.put.mockResolvedValueOnce({ ok: true });
      const { updateStorageLocation } = await import('../storage-location-service');
      await updateStorageLocation(1, { name: 'Updated', is_active: false });
      expect(mockApiClient.put).toHaveBeenCalledWith('/storage-locations/1', { name: 'Updated', is_active: false });
    });
  });

  describe('deleteStorageLocation', () => {
    it('sends DELETE', async () => {
      mockApiClient.delete.mockResolvedValueOnce({ ok: true });
      const { deleteStorageLocation } = await import('../storage-location-service');
      await deleteStorageLocation(1);
      expect(mockApiClient.delete).toHaveBeenCalledWith('/storage-locations/1');
    });
  });

  describe('bulkUpdateStorageLocations', () => {
    it('sends PUT to /bulk with ids and is_active', async () => {
      mockApiClient.put.mockResolvedValueOnce({ data: { updated: 3 } });
      const { bulkUpdateStorageLocations } = await import('../storage-location-service');
      const result = await bulkUpdateStorageLocations([1, 2, 3], true);
      expect(mockApiClient.put).toHaveBeenCalledWith('/storage-locations/bulk', { ids: [1, 2, 3], is_active: true });
      expect(result).toBe(3);
    });

    it('sends bulk deactivate', async () => {
      mockApiClient.put.mockResolvedValueOnce({ data: { updated: 2 } });
      const { bulkUpdateStorageLocations } = await import('../storage-location-service');
      const result = await bulkUpdateStorageLocations([4, 5], false);
      expect(mockApiClient.put).toHaveBeenCalledWith('/storage-locations/bulk', { ids: [4, 5], is_active: false });
      expect(result).toBe(2);
    });
  });

  describe('bulkDeleteStorageLocations', () => {
    it('sends DELETE to /bulk with ids', async () => {
      mockApiClient.delete.mockResolvedValueOnce({ data: { deleted: 2 } });
      const { bulkDeleteStorageLocations } = await import('../storage-location-service');
      const result = await bulkDeleteStorageLocations([1, 2]);
      expect(mockApiClient.delete).toHaveBeenCalledWith('/storage-locations/bulk', { data: { ids: [1, 2] } });
      expect(result).toBe(2);
    });
  });
});
