import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockFetch = vi.fn();
vi.mock('$shared/api/http-client', () => ({
  apiFetch: (...args: unknown[]) => mockFetch(...args),
}));

describe('settings-service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('getCategories returns data and total', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ data: [{ id: 1, name: 'Electronics' }], total: 1 }),
    });

    const { getCategories } = await import('../settings-service');
    const result = await getCategories({ limit: 20, offset: 0 });

    expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining('/api/categories/manage?'));
    expect(result.data).toHaveLength(1);
    expect(result.total).toBe(1);
  });

  it('getCategories returns defaults on non-ok', async () => {
    mockFetch.mockResolvedValueOnce({ ok: false });

    const { getCategories } = await import('../settings-service');
    const result = await getCategories({ limit: 20, offset: 0 });

    expect(result.data).toEqual([]);
    expect(result.total).toBe(0);
  });

  it('getCategories returns defaults on missing data', async () => {
    mockFetch.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({}) });

    const { getCategories } = await import('../settings-service');
    const result = await getCategories({ limit: 20, offset: 0 });

    expect(result.data).toEqual([]);
    expect(result.total).toBe(0);
  });

  it('getCategories includes search/sort/dir params when provided', async () => {
    mockFetch.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [], total: 0 }) });

    const { getCategories } = await import('../settings-service');
    await getCategories({ limit: 10, offset: 0, search: 'food', sort: 'name', dir: 'asc' });

    const url: string = mockFetch.mock.calls[0][0];
    expect(url).toContain('search=food');
    expect(url).toContain('sort=name');
    expect(url).toContain('dir=asc');
  });

  it('createCategory posts to /api/categories', async () => {
    mockFetch.mockResolvedValueOnce({ ok: true });

    const { createCategory } = await import('../settings-service');
    const result = await createCategory({ name: 'New Cat' });

    expect(mockFetch).toHaveBeenCalledWith('/api/categories', {
      method: 'POST',
      body: JSON.stringify({ name: 'New Cat' }),
    });
    expect(result).toBe(true);
  });

  it('updateCategory puts to /api/categories/:id', async () => {
    mockFetch.mockResolvedValueOnce({ ok: true });

    const { updateCategory } = await import('../settings-service');
    const result = await updateCategory(1, { name: 'Updated' });

    expect(mockFetch).toHaveBeenCalledWith('/api/categories/1', {
      method: 'PUT',
      body: JSON.stringify({ name: 'Updated' }),
    });
    expect(result).toBe(true);
  });

  it('deleteCategory deletes /api/categories/:id', async () => {
    mockFetch.mockResolvedValueOnce({ ok: true });

    const { deleteCategory } = await import('../settings-service');
    const result = await deleteCategory(1);

    expect(mockFetch).toHaveBeenCalledWith('/api/categories/1', { method: 'DELETE' });
    expect(result).toBe(true);
  });
});
