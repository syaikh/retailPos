import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockFetch = vi.fn();
vi.mock('$shared/api/http-client', () => ({
  apiFetch: (...args: unknown[]) => mockFetch(...args),
}));

describe('roles-service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('getRoles returns roles array', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ data: [{ id: 1, name: 'admin', is_system: true, permissions: [] }] }),
    });

    const { getRoles } = await import('../roles-service');
    const result = await getRoles();

    expect(mockFetch).toHaveBeenCalledWith('/api/admin/roles');
    expect(result).toHaveLength(1);
    expect(result[0].name).toBe('admin');
  });

  it('getRoles returns empty array on non-ok', async () => {
    mockFetch.mockResolvedValueOnce({ ok: false });

    const { getRoles } = await import('../roles-service');
    const result = await getRoles();

    expect(result).toEqual([]);
  });

  it('getPermissions returns permissions array', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ data: [{ id: 1, name: 'Create User', code: 'user:create' }] }),
    });

    const { getPermissions } = await import('../roles-service');
    const result = await getPermissions();

    expect(mockFetch).toHaveBeenCalledWith('/api/admin/permissions');
    expect(result).toHaveLength(1);
  });

  it('createRole posts to /api/admin/roles', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ data: { id: 5 } }),
    });

    const { createRole } = await import('../roles-service');
    const result = await createRole({ name: 'manager', description: 'Manages stuff' });

    expect(mockFetch).toHaveBeenCalledWith('/api/admin/roles', { method: 'POST', body: JSON.stringify({ name: 'manager', description: 'Manages stuff' }) });
    expect(result).toEqual({ id: 5 });
  });

  it('updateRolePermissions puts to /api/admin/roles/:id/permissions', async () => {
    mockFetch.mockResolvedValueOnce({ ok: true });

    const { updateRolePermissions } = await import('../roles-service');
    const result = await updateRolePermissions(1, [1, 2, 3]);

    expect(mockFetch).toHaveBeenCalledWith('/api/admin/roles/1/permissions', { method: 'PUT', body: JSON.stringify({ permission_ids: [1, 2, 3] }) });
    expect(result).toBe(true);
  });

  it('deleteRole deletes /api/admin/roles/:id', async () => {
    mockFetch.mockResolvedValueOnce({ ok: true });

    const { deleteRole } = await import('../roles-service');
    const result = await deleteRole(1);

    expect(mockFetch).toHaveBeenCalledWith('/api/admin/roles/1', { method: 'DELETE' });
    expect(result).toBe(true);
  });
});
