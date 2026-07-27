import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockFetch = vi.fn();
vi.mock('$shared/api/http-client', () => ({
  apiFetch: (...args: unknown[]) => mockFetch(...args),
}));

describe('settings-service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // Categories
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

  // Brands
  it('getBrands returns data and total', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ data: [{ id: 1, name: 'Sony' }], total: 1 }),
    });

    const { getBrands } = await import('../settings-service');
    const result = await getBrands({ limit: 20, offset: 0 });

    expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining('/api/brands?'));
    expect(result.data).toHaveLength(1);
    expect(result.total).toBe(1);
  });

  it('getBrands returns defaults on non-ok', async () => {
    mockFetch.mockResolvedValueOnce({ ok: false });

    const { getBrands } = await import('../settings-service');
    const result = await getBrands({ limit: 20, offset: 0 });

    expect(result.data).toEqual([]);
    expect(result.total).toBe(0);
  });

  it('getBrands includes search param when provided', async () => {
    mockFetch.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({ data: [], total: 0 }) });

    const { getBrands } = await import('../settings-service');
    await getBrands({ limit: 10, offset: 0, search: 'sony' });

    const url: string = mockFetch.mock.calls[0][0];
    expect(url).toContain('search=sony');
  });

  it('createBrand posts to /api/brands', async () => {
    mockFetch.mockResolvedValueOnce({ ok: true });

    const { createBrand } = await import('../settings-service');
    const result = await createBrand({ name: 'New Brand' });

    expect(mockFetch).toHaveBeenCalledWith('/api/brands', {
      method: 'POST',
      body: JSON.stringify({ name: 'New Brand' }),
    });
    expect(result).toBe(true);
  });

  it('updateBrand puts to /api/brands/:id', async () => {
    mockFetch.mockResolvedValueOnce({ ok: true });

    const { updateBrand } = await import('../settings-service');
    const result = await updateBrand(1, { name: 'Updated Brand' });

    expect(mockFetch).toHaveBeenCalledWith('/api/brands/1', {
      method: 'PUT',
      body: JSON.stringify({ name: 'Updated Brand' }),
    });
    expect(result).toBe(true);
  });

  it('deleteBrand deletes /api/brands/:id', async () => {
    mockFetch.mockResolvedValueOnce({ ok: true });

    const { deleteBrand } = await import('../settings-service');
    const result = await deleteBrand(1);

    expect(mockFetch).toHaveBeenCalledWith('/api/brands/1', { method: 'DELETE' });
    expect(result).toBe(true);
  });

  // UOM
  it('getUnitsOfMeasure returns data and total', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ data: [{ id: 1, code: 'PCS', name: 'Pieces' }], total: 1 }),
    });

    const { getUnitsOfMeasure } = await import('../settings-service');
    const result = await getUnitsOfMeasure({ limit: 20, offset: 0 });

    expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining('/api/units-of-measure?'));
    expect(result.data).toHaveLength(1);
    expect(result.total).toBe(1);
  });

  it('getUnitsOfMeasure returns defaults on non-ok', async () => {
    mockFetch.mockResolvedValueOnce({ ok: false });

    const { getUnitsOfMeasure } = await import('../settings-service');
    const result = await getUnitsOfMeasure({ limit: 20, offset: 0 });

    expect(result.data).toEqual([]);
    expect(result.total).toBe(0);
  });

  it('createUnitOfMeasure posts to /api/units-of-measure', async () => {
    mockFetch.mockResolvedValueOnce({ ok: true });

    const { createUnitOfMeasure } = await import('../settings-service');
    const result = await createUnitOfMeasure({ code: 'KG', name: 'Kilogram' });

    expect(mockFetch).toHaveBeenCalledWith('/api/units-of-measure', {
      method: 'POST',
      body: JSON.stringify({ code: 'KG', name: 'Kilogram' }),
    });
    expect(result).toBe(true);
  });

  it('updateUnitOfMeasure puts to /api/units-of-measure/:id', async () => {
    mockFetch.mockResolvedValueOnce({ ok: true });

    const { updateUnitOfMeasure } = await import('../settings-service');
    const result = await updateUnitOfMeasure(1, { name: 'Updated UOM' });

    expect(mockFetch).toHaveBeenCalledWith('/api/units-of-measure/1', {
      method: 'PUT',
      body: JSON.stringify({ name: 'Updated UOM' }),
    });
    expect(result).toBe(true);
  });

  it('deleteUnitOfMeasure deletes /api/units-of-measure/:id', async () => {
    mockFetch.mockResolvedValueOnce({ ok: true });

    const { deleteUnitOfMeasure } = await import('../settings-service');
    const result = await deleteUnitOfMeasure(1);

    expect(mockFetch).toHaveBeenCalledWith('/api/units-of-measure/1', { method: 'DELETE' });
    expect(result).toBe(true);
  });

  // Export & Import
  it('exportBrands opens download URL with token', async () => {
    const openSpy = vi.fn();
    vi.spyOn(window, 'open').mockImplementation(openSpy);
    const getItemSpy = vi.spyOn(sessionStorage, 'getItem').mockReturnValue('test-token');

    const { exportBrands } = await import('../settings-service');
    await exportBrands('csv');

    expect(getItemSpy).toHaveBeenCalledWith('access_token');
    expect(openSpy).toHaveBeenCalledWith('/api/brands/export?format=csv&token=test-token', '_blank');
  });

  it('importBrands sends file and returns json on success', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ imported: 5 }),
    });

    const { importBrands } = await import('../settings-service');
    const file = new File(['data'], 'test.csv', { type: 'text/csv' });
    const result = await importBrands(file);

    expect(mockFetch).toHaveBeenCalledWith('/api/brands/import', {
      method: 'POST',
      body: expect.any(FormData),
    });
    expect(result).toEqual({ imported: 5 });
  });

  it('importBrands rejects on error response', async () => {
    const errResponse = { error: 'Invalid format' };
    mockFetch.mockResolvedValueOnce({
      ok: false,
      json: () => Promise.resolve(errResponse),
    });

    const { importBrands } = await import('../settings-service');
    const file = new File(['data'], 'test.csv', { type: 'text/csv' });

    await expect(importBrands(file)).rejects.toEqual(errResponse);
  });
});
