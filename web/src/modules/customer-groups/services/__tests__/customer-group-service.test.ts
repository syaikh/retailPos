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
      expect(mockApiClient.get).toHaveBeenCalledWith('/customer-groups', { params: { limit: 20, offset: 0, search: undefined } });
      expect(result.data).toHaveLength(1);
      expect(result.total).toBe(1);
    });

    it('includes search and is_active filters', async () => {
      mockApiClient.get.mockResolvedValueOnce({ data: { data: [], total: 0 } });
      const { getCustomerGroups } = await import('../customer-group-service');
      await getCustomerGroups({ limit: 10, offset: 0, search: 'member', is_active: true });
      const params = mockApiClient.get.mock.calls[0][1].params;
      expect(params.search).toBe('member');
      expect(params.is_active).toBe('true');
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
    it('sends POST with payload', async () => {
      mockApiClient.post.mockResolvedValueOnce({ ok: true });
      const { createCustomerGroup } = await import('../customer-group-service');
      await createCustomerGroup({ name: 'New Group', description: 'Test' });
      expect(mockApiClient.post).toHaveBeenCalledWith('/customer-groups', { name: 'New Group', description: 'Test' });
    });
  });

  describe('updateCustomerGroup', () => {
    it('sends PUT with payload', async () => {
      mockApiClient.put.mockResolvedValueOnce({ ok: true });
      const { updateCustomerGroup } = await import('../customer-group-service');
      await updateCustomerGroup(1, { name: 'Updated', is_active: false });
      expect(mockApiClient.put).toHaveBeenCalledWith('/customer-groups/1', { name: 'Updated', is_active: false });
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
});
