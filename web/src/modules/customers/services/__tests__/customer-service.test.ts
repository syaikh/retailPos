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

describe('customer-service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('getCustomers builds correct query params', async () => {
    mockGet.mockResolvedValueOnce({ data: { data: [{ id: 1, name: 'Test' }], total: 1 } });

    const { getCustomers } = await import('../customer-service');
    const result = await getCustomers({ search: 'test', limit: 10, offset: 0 });

    expect(mockGet).toHaveBeenCalledWith('/customers', {
      params: { limit: 10, offset: 0, search: 'test' },
    });
    expect(result.data).toHaveLength(1);
    expect(result.total).toBe(1);
  });

  it('getCustomers passes isActive filter', async () => {
    mockGet.mockResolvedValueOnce({ data: { data: [], total: 0 } });

    const { getCustomers } = await import('../customer-service');
    await getCustomers({ isActive: 'true', limit: 20, offset: 0 });

    expect(mockGet).toHaveBeenCalledWith('/customers', {
      params: { limit: 20, offset: 0, isActive: 'true' },
    });
  });

  it('createCustomer posts to /customers', async () => {
    mockPost.mockResolvedValueOnce({ status: 200 });

    const { createCustomer } = await import('../customer-service');
    await createCustomer({ name: 'John', phone: '081234', email: 'john@test.com' });

    expect(mockPost).toHaveBeenCalledWith('/customers', {
      name: 'John', phone: '081234', email: 'john@test.com',
    });
  });

  it('updateCustomer puts to /customers/:id', async () => {
    mockPut.mockResolvedValueOnce({ status: 200 });

    const { updateCustomer } = await import('../customer-service');
    await updateCustomer(1, { name: 'Updated' });

    expect(mockPut).toHaveBeenCalledWith('/customers/1', { name: 'Updated' });
  });

  it('deleteCustomer deletes to /customers/:id', async () => {
    mockDelete.mockResolvedValueOnce({ status: 200 });

    const { deleteCustomer } = await import('../customer-service');
    await deleteCustomer(1);

    expect(mockDelete).toHaveBeenCalledWith('/customers/1');
  });

  it('bulkUpdateStatus posts to /customers/bulk/status', async () => {
    mockPost.mockResolvedValueOnce({ status: 200 });

    const { bulkUpdateStatus } = await import('../customer-service');
    await bulkUpdateStatus([1, 2], true);

    expect(mockPost).toHaveBeenCalledWith('/customers/bulk/status', {
      ids: [1, 2], is_active: true,
    });
  });

  it('bulkDelete posts to /customers/bulk/delete', async () => {
    mockPost.mockResolvedValueOnce({ status: 200 });

    const { bulkDelete } = await import('../customer-service');
    await bulkDelete([1, 2, 3]);

    expect(mockPost).toHaveBeenCalledWith('/customers/bulk/delete', { ids: [1, 2, 3] });
  });
});
