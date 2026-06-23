import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockGet = vi.fn();
const mockDelete = vi.fn();
const mockClient = vi.fn();

vi.mock('$shared/api/http-client', () => ({
  default: Object.assign(
    (...args: unknown[]) => mockClient(...args),
    { get: (...args: unknown[]) => mockGet(...args), delete: (...args: unknown[]) => mockDelete(...args) },
  ),
}));

describe('users-service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('getUsers returns users list and total', async () => {
    mockGet.mockResolvedValueOnce({
      data: { data: [{ id: 1, username: 'admin' }], total: 1 },
    });

    const { getUsers } = await import('../users-service');
    const result = await getUsers({ limit: 20, offset: 0 });

    expect(mockGet).toHaveBeenCalledWith('/admin/users', { params: { limit: 20, offset: 0 } });
    expect(result.data).toHaveLength(1);
    expect(result.total).toBe(1);
  });

  it('getUsers returns defaults on missing data', async () => {
    mockGet.mockResolvedValueOnce({ data: {} });

    const { getUsers } = await import('../users-service');
    const result = await getUsers({ limit: 20, offset: 0 });

    expect(result.data).toEqual([]);
    expect(result.total).toBe(0);
  });

  it('getRolesList returns roles array', async () => {
    mockGet.mockResolvedValueOnce({ data: { data: [{ id: 1, name: 'admin' }] } });

    const { getRolesList } = await import('../users-service');
    const result = await getRolesList();

    expect(result).toHaveLength(1);
    expect(result[0].name).toBe('admin');
  });

  it('getRolesList returns empty array on no data', async () => {
    mockGet.mockResolvedValueOnce({ data: {} });

    const { getRolesList } = await import('../users-service');
    const result = await getRolesList();

    expect(result).toEqual([]);
  });

  it('createUser posts to /admin/users', async () => {
    mockClient.mockResolvedValueOnce({});

    const { createUser } = await import('../users-service');
    await createUser({ username: 'newuser', email: 'a@b.com', password: 'secret', role_id: 2, is_active: true });

    expect(mockClient).toHaveBeenCalledWith({ url: '/admin/users', method: 'POST', data: { username: 'newuser', email: 'a@b.com', password: 'secret', role_id: 2, is_active: true } });
  });

  it('updateUser puts to /admin/users/:id', async () => {
    mockClient.mockResolvedValueOnce({});

    const { updateUser } = await import('../users-service');
    await updateUser(1, { email: 'new@b.com' });

    expect(mockClient).toHaveBeenCalledWith({ url: '/admin/users/1', method: 'PUT', data: { email: 'new@b.com' } });
  });

  it('deleteUser deletes /admin/users/:id', async () => {
    mockDelete.mockResolvedValueOnce({});

    const { deleteUser } = await import('../users-service');
    await deleteUser(1);

    expect(mockDelete).toHaveBeenCalledWith('/admin/users/1');
  });
});
