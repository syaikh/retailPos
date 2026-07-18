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

describe('customer-group-service', () => {
  describe('getCustomerGroups', () => {
    it('fetches with basic params', async () => {
      mockApiClient.get.mockResolvedValueOnce({ data: { data: [{ id: 1, name: 'VIP' }], total: 1 } });
      const { getCustomerGroups } = await import('../customer-group-service');
      const result = await getCustomerGroups({ limit: 20, offset: 0 });
      expect(mockApiClient.get).toHaveBeenCalledWith('/customer-groups', { params: { limit: 20, offset: 0, search: undefined, has_customers: undefined } });
      expect(result.data).toHaveLength(1);
      expect(result.total).toBe(1);
    });

    it('includes search, is_active, and has_customers filters', async () => {
      mockApiClient.get.mockResolvedValueOnce({ data: { data: [], total: 0 } });
      const { getCustomerGroups } = await import('../customer-group-service');
      await getCustomerGroups({ limit: 10, offset: 0, search: 'member', is_active: true, has_customers: true });
      const params = mockApiClient.get.mock.calls[0][1].params;
      expect(params.search).toBe('member');
      expect(params.is_active).toBe('true');
      expect(params.has_customers).toBe('true');
    });

    it('sets has_customers to false when has_customers is false', async () => {
      mockApiClient.get.mockResolvedValueOnce({ data: { data: [], total: 0 } });
      const { getCustomerGroups } = await import('../customer-group-service');
      await getCustomerGroups({ limit: 10, offset: 0, has_customers: false });
      const params = mockApiClient.get.mock.calls[0][1].params;
      expect(params.has_customers).toBe('false');
    });
  });

  describe('getCustomerGroup', () => {
    it('fetches single group by id', async () => {
      mockApiClient.get.mockResolvedValueOnce({ data: { data: { id: 1, name: 'VIP' } } });
      const { getCustomerGroup } = await import('../customer-group-service');
      const result = await getCustomerGroup(1);
      expect(mockApiClient.get).toHaveBeenCalledWith('/customer-groups/1');
      expect(result?.name).toBe('VIP');
    });
  });

  describe('createCustomerGroup', () => {
    it('sends POST with payload including color', async () => {
      mockApiClient.post.mockResolvedValueOnce({ ok: true });
      const { createCustomerGroup } = await import('../customer-group-service');
      await createCustomerGroup({ name: 'New Group', description: 'Test', color: '#6C5CE7' });
      expect(mockApiClient.post).toHaveBeenCalledWith('/customer-groups', { name: 'New Group', description: 'Test', color: '#6C5CE7' });
    });
  });

  describe('updateCustomerGroup', () => {
    it('sends PUT with payload including color', async () => {
      mockApiClient.put.mockResolvedValueOnce({ ok: true });
      const { updateCustomerGroup } = await import('../customer-group-service');
      await updateCustomerGroup(1, { name: 'Updated', is_active: false, color: '#00B894' });
      expect(mockApiClient.put).toHaveBeenCalledWith('/customer-groups/1', { name: 'Updated', is_active: false, color: '#00B894' });
    });
  });

  describe('deleteCustomerGroup', () => {
    it('sends DELETE', async () => {
      mockApiClient.delete.mockResolvedValueOnce({ ok: true });
      const { deleteCustomerGroup } = await import('../customer-group-service');
      await deleteCustomerGroup(1);
      expect(mockApiClient.delete).toHaveBeenCalledWith('/customer-groups/1');
    });
  });

  describe('bulkUpdateCustomerGroups', () => {
    it('sends PUT to /bulk with ids and is_active', async () => {
      mockApiClient.put.mockResolvedValueOnce({ data: { updated: 3 } });
      const { bulkUpdateCustomerGroups } = await import('../customer-group-service');
      const result = await bulkUpdateCustomerGroups([1, 2, 3], true);
      expect(mockApiClient.put).toHaveBeenCalledWith('/customer-groups/bulk', { ids: [1, 2, 3], is_active: true });
      expect(result).toBe(3);
    });

    it('sends bulk deactivate', async () => {
      mockApiClient.put.mockResolvedValueOnce({ data: { updated: 2 } });
      const { bulkUpdateCustomerGroups } = await import('../customer-group-service');
      const result = await bulkUpdateCustomerGroups([4, 5], false);
      expect(mockApiClient.put).toHaveBeenCalledWith('/customer-groups/bulk', { ids: [4, 5], is_active: false });
      expect(result).toBe(2);
    });
  });

  describe('bulkDeleteCustomerGroups', () => {
    it('sends DELETE to /bulk with ids', async () => {
      mockApiClient.delete.mockResolvedValueOnce({ data: { deleted: 2 } });
      const { bulkDeleteCustomerGroups } = await import('../customer-group-service');
      const result = await bulkDeleteCustomerGroups([1, 2]);
      expect(mockApiClient.delete).toHaveBeenCalledWith('/customer-groups/bulk', { data: { ids: [1, 2] } });
      expect(result).toBe(2);
    });
  });
});
