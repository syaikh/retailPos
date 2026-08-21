import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockApiFetch = vi.fn();

vi.mock('$shared/api/http-client', () => ({
  apiFetch: (...args: unknown[]) => mockApiFetch(...args),
}));

import { getPurchaseOrders } from '../po-service';

describe('po-service getPurchaseOrders', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockApiFetch.mockResolvedValue({
      ok: true,
      json: async () => ({ data: [], total: 0 }),
    });
  });

  it('defaults sort_by to updated_at and sort_dir to desc', async () => {
    await getPurchaseOrders({ page: 0, pageSize: 20, search: '' });

    expect(mockApiFetch).toHaveBeenCalledTimes(1);
    const url = mockApiFetch.mock.calls[0][0] as string;
    expect(url).toContain('sort_by=updated_at');
    expect(url).toContain('sort_dir=desc');
  });

  it('uses provided sort parameters when present', async () => {
    await getPurchaseOrders({ page: 1, pageSize: 10, search: 'abc', sortBy: 'po_number', sortDir: 'asc' });

    const url = mockApiFetch.mock.calls[0][0] as string;
    expect(url).toContain('sort_by=po_number');
    expect(url).toContain('sort_dir=asc');
  });
});
