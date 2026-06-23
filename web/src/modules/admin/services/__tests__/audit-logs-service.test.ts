import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockGet = vi.fn();
vi.mock('$shared/api/http-client', () => ({
  default: { get: (...args: unknown[]) => mockGet(...args) },
}));

describe('audit-logs-service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('getAuditLogs fetches with query params', async () => {
    mockGet.mockResolvedValueOnce({
      data: { data: [{ id: 1, action: 'create', entity_type: 'user' }], total: 1 },
    });

    const { getAuditLogs } = await import('../audit-logs-service');
    const result = await getAuditLogs({
      limit: 20,
      offset: 0,
      search: 'admin',
      start_date: '2024-01-01T00:00:00.000Z',
      end_date: '2024-01-02T00:00:00.000Z',
      action: 'create',
      entity_type: 'user',
    });

    expect(mockGet).toHaveBeenCalled();
    const url: string = mockGet.mock.calls[0][0];
    expect(url).toContain('audit-logs?');
    expect(url).toContain('limit=20');
    expect(url).toContain('offset=0');
    expect(url).toContain('search=admin');
    expect(url).toContain('action=create');
    expect(url).toContain('entity_type=user');
    expect(result.data).toHaveLength(1);
    expect(result.total).toBe(1);
  });

  it('getAuditLogs handles missing data gracefully', async () => {
    mockGet.mockResolvedValueOnce({ data: {} });

    const { getAuditLogs } = await import('../audit-logs-service');
    const result = await getAuditLogs({
      limit: 20,
      offset: 0,
      search: '',
      start_date: '',
      end_date: '',
    });

    expect(result.data).toEqual([]);
    expect(result.total).toBe(0);
  });

  it('getAuditLogs accepts AbortSignal', async () => {
    const controller = new AbortController();
    mockGet.mockResolvedValueOnce({ data: { data: [], total: 0 } });

    const { getAuditLogs } = await import('../audit-logs-service');
    await getAuditLogs(
      { limit: 20, offset: 0, search: '', start_date: '', end_date: '' },
      controller.signal,
    );

    expect(mockGet.mock.calls[0][1]?.signal).toBe(controller.signal);
  });

  it('buildExportUrl returns correct URL', async () => {
    const { buildExportUrl } = await import('../audit-logs-service');
    const url = buildExportUrl('csv', {
      search: 'admin',
      start_date: '2024-01-01T00:00:00.000Z',
      end_date: '2024-01-02T00:00:00.000Z',
      action: 'create',
    });

    expect(url).toContain('/api/audit-logs/export?');
    expect(url).toContain('format=csv');
    expect(url).toContain('action=create');
  });

  it('buildExportUrl omits optional filters when not provided', async () => {
    const { buildExportUrl } = await import('../audit-logs-service');
    const url = buildExportUrl('xlsx', {
      search: '',
      start_date: '',
      end_date: '',
    });

    expect(url).not.toContain('action=');
    expect(url).not.toContain('entity_type=');
  });
});
