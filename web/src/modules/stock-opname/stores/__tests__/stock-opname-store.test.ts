import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockListStockOpnames = vi.fn();
const mockGetStockOpname = vi.fn();
const mockGetSessionSummary = vi.fn();
const mockUseWebSocket = vi.fn();

vi.mock('../../services/stock-opname-service', () => ({
  listStockOpnames: (...args: unknown[]) => mockListStockOpnames(...args),
  getStockOpname: (...args: unknown[]) => mockGetStockOpname(...args),
  getSessionSummary: (...args: unknown[]) => mockGetSessionSummary(...args),
}));

vi.mock('$shared/api/websocket', () => ({
  useWebSocket: () => mockUseWebSocket(),
}));

function makeMockWS() {
  const handlers: Record<string, (data?: unknown) => void> = {};
  return {
    on: vi.fn((event: string, cb: (data?: unknown) => void) => {
      handlers[event] = cb;
      return () => { delete handlers[event]; };
    }),
    _fire: (event: string, data?: unknown) => handlers[event]?.(data),
    _clear: () => Object.keys(handlers).forEach(k => delete handlers[k]),
  };
}

describe('stock-opname-store', () => {
  let store: ReturnType<typeof import('../stock-opname-store.svelte').useStockOpnameStore>;

  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
  });

  it('returns expected API shape', async () => {
    const { useStockOpnameStore } = await import('../stock-opname-store.svelte');
    store = useStockOpnameStore();
    expect(store).toHaveProperty('sessions');
    expect(store).toHaveProperty('total');
    expect(store).toHaveProperty('loading');
    expect(store).toHaveProperty('current');
    expect(store).toHaveProperty('currentAssignments');
    expect(store).toHaveProperty('assignableUsers');
    expect(store).toHaveProperty('currentSummary');
    expect(store).toHaveProperty('statusFilter');
    expect(store).toHaveProperty('searchFilter');
    expect(store).toHaveProperty('page');
    expect(store).toHaveProperty('pageSize');
    expect(store).toHaveProperty('offset');
    expect(store).toHaveProperty('loadSessions');
    expect(store).toHaveProperty('loadSession');
    expect(store).toHaveProperty('subscribeToWS');
  });

  it('loads sessions successfully', async () => {
    mockListStockOpnames.mockResolvedValueOnce({ data: [{ id: 1 }], total: 1 });
    const { useStockOpnameStore } = await import('../stock-opname-store.svelte');
    store = useStockOpnameStore();
    store.pageSize = 20;
    store.page = 0;
    await store.loadSessions({ status: '', search: '', limit: 20, offset: 0 });
    expect(mockListStockOpnames).toHaveBeenCalled();
    expect(store.sessions).toHaveLength(1);
    expect(store.total).toBe(1);
    expect(store.loading).toBe(false);
  });

  it('loads a session into current', async () => {
    mockGetStockOpname.mockResolvedValueOnce({ id: 7, session_number: 'SO-0007', summary: { total_items: 1, counted_items: 0, pending_items: 1 } });
    const { useStockOpnameStore } = await import('../stock-opname-store.svelte');
    store = useStockOpnameStore();
    await store.loadSession(7);
    expect(mockGetStockOpname).toHaveBeenCalledWith(7);
    expect(store.current?.id).toBe(7);
  });

  describe('subscribeToWS', () => {
    it('subscribes to all so_* status events', async () => {
      const ws = makeMockWS();
      mockUseWebSocket.mockReturnValue(ws);
      const { useStockOpnameStore } = await import('../stock-opname-store.svelte');
      store = useStockOpnameStore();
      const unsub = store.subscribeToWS();

      const events = ws.on.mock.calls.map((c: [string, unknown]) => c[0]);
      expect(events).toEqual([
        'so_created',
        'so_submitted',
        'so_approved',
        'so_rejected',
        'so_needs_recount',
        'so_cancelled',
      ]);
      expect(unsub).toEqual(expect.any(Function));
    });

    it('returns noop if already subscribed', async () => {
      const ws = makeMockWS();
      mockUseWebSocket.mockReturnValue(ws);
      const { useStockOpnameStore } = await import('../stock-opname-store.svelte');
      store = useStockOpnameStore();
      store.subscribeToWS();
      const unsub2 = store.subscribeToWS();
      expect(ws.on).toHaveBeenCalledTimes(6);
      expect(unsub2()).toBeUndefined();
    });

    it('reloads the list when a session event arrives and no current session is open', async () => {
      mockListStockOpnames.mockResolvedValueOnce({ data: [], total: 0 });
      const ws = makeMockWS();
      mockUseWebSocket.mockReturnValue(ws);
      const { useStockOpnameStore } = await import('../stock-opname-store.svelte');
      store = useStockOpnameStore();
      store.subscribeToWS();

      ws._fire('so_submitted', { session_id: 3, session_number: 'SO-0003' });
      await vi.waitFor(() => {
        expect(mockListStockOpnames).toHaveBeenCalled();
      });
      expect(mockGetStockOpname).not.toHaveBeenCalled();
    });

    it('reloads the current session when an event matches its id', async () => {
      mockGetStockOpname
        .mockResolvedValueOnce({ id: 5, session_number: 'SO-0005', status: 'counting', summary: { total_items: 1 } })
        .mockResolvedValueOnce({ id: 5, session_number: 'SO-0005', status: 'pending_approval', summary: { total_items: 1 } });
      const ws = makeMockWS();
      mockUseWebSocket.mockReturnValue(ws);
      const { useStockOpnameStore } = await import('../stock-opname-store.svelte');
      store = useStockOpnameStore();
      await store.loadSession(5);
      store.subscribeToWS();

      ws._fire('so_submitted', { session_id: 5, session_number: 'SO-0005' });
      await vi.waitFor(() => {
        expect(mockGetStockOpname).toHaveBeenCalledTimes(2);
      });
    });

    it('unsubscribing removes all handlers', async () => {
      const ws = makeMockWS();
      mockUseWebSocket.mockReturnValue(ws);
      const { useStockOpnameStore } = await import('../stock-opname-store.svelte');
      store = useStockOpnameStore();
      const unsub = store.subscribeToWS();

      unsub();
      expect(ws.on).toHaveBeenCalledTimes(6);
    });
  });
});
