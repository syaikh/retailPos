import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockCreateStockOpname = vi.fn();
const mockListStockOpnames = vi.fn();
const mockGetStockOpname = vi.fn();
const mockOpenStockOpname = vi.fn();
const mockCancelStockOpname = vi.fn();
const mockAssignCounter = vi.fn();
const mockGetAssignableUsers = vi.fn();
const mockGetAssignments = vi.fn();
const mockReassignCounter = vi.fn();
const mockSaveCount = vi.fn();
const mockGetCountHistory = vi.fn();
const mockSubmitSession = vi.fn();
const mockStartCounting = vi.fn();
const mockVerifySession = vi.fn();
const mockRejectSession = vi.fn();
const mockRequestRecount = vi.fn();
const mockResumeCounting = vi.fn();
const mockPostAdjustment = vi.fn();
const mockCloseStockOpname = vi.fn();
const mockGetSessionSummary = vi.fn();
const mockListAdjustments = vi.fn();
const mockGetAdjustment = vi.fn();
const mockExportStockOpname = vi.fn();
const mockUseWebSocket = vi.fn();

vi.mock('../../services/stock-opname-service', () => ({
  createStockOpname: (...args: unknown[]) => mockCreateStockOpname(...args),
  listStockOpnames: (...args: unknown[]) => mockListStockOpnames(...args),
  getStockOpname: (...args: unknown[]) => mockGetStockOpname(...args),
  openStockOpname: (...args: unknown[]) => mockOpenStockOpname(...args),
  cancelStockOpname: (...args: unknown[]) => mockCancelStockOpname(...args),
  assignCounter: (...args: unknown[]) => mockAssignCounter(...args),
  getAssignableUsers: (...args: unknown[]) => mockGetAssignableUsers(...args),
  getAssignments: (...args: unknown[]) => mockGetAssignments(...args),
  reassignCounter: (...args: unknown[]) => mockReassignCounter(...args),
  saveCount: (...args: unknown[]) => mockSaveCount(...args),
  getCountHistory: (...args: unknown[]) => mockGetCountHistory(...args),
  submitSession: (...args: unknown[]) => mockSubmitSession(...args),
  startCounting: (...args: unknown[]) => mockStartCounting(...args),
  verifySession: (...args: unknown[]) => mockVerifySession(...args),
  rejectSession: (...args: unknown[]) => mockRejectSession(...args),
  requestRecount: (...args: unknown[]) => mockRequestRecount(...args),
  resumeCounting: (...args: unknown[]) => mockResumeCounting(...args),
  postAdjustment: (...args: unknown[]) => mockPostAdjustment(...args),
  closeStockOpname: (...args: unknown[]) => mockCloseStockOpname(...args),
  getSessionSummary: (...args: unknown[]) => mockGetSessionSummary(...args),
  listAdjustments: (...args: unknown[]) => mockListAdjustments(...args),
  getAdjustment: (...args: unknown[]) => mockGetAdjustment(...args),
  exportStockOpname: (...args: unknown[]) => mockExportStockOpname(...args),
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
    expect(store).toHaveProperty('adjustments');
    expect(store).toHaveProperty('adjustmentsTotal');
    expect(store).toHaveProperty('adjustmentsLoading');
    expect(store).toHaveProperty('currentAdjustment');
    expect(store).toHaveProperty('statusFilter');
    expect(store).toHaveProperty('searchFilter');
    expect(store).toHaveProperty('page');
    expect(store).toHaveProperty('pageSize');
    expect(store).toHaveProperty('offset');
    expect(store).toHaveProperty('loadSessions');
    expect(store).toHaveProperty('loadSession');
    expect(store).toHaveProperty('createSession');
    expect(store).toHaveProperty('open');
    expect(store).toHaveProperty('cancelSession');
    expect(store).toHaveProperty('assign');
    expect(store).toHaveProperty('reassign');
    expect(store).toHaveProperty('saveCount');
    expect(store).toHaveProperty('getCountHistory');
    expect(store).toHaveProperty('submit');
    expect(store).toHaveProperty('start');
    expect(store).toHaveProperty('verify');
    expect(store).toHaveProperty('reject');
    expect(store).toHaveProperty('recount');
    expect(store).toHaveProperty('resume');
    expect(store).toHaveProperty('post');
    expect(store).toHaveProperty('close');
    expect(store).toHaveProperty('loadAdjustments');
    expect(store).toHaveProperty('loadAdjustment');
    expect(store).toHaveProperty('exportCSV');
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

  it('creates a session via the service', async () => {
    mockCreateStockOpname.mockResolvedValueOnce({ id: 9, session_number: 'SO-0009' });
    const { useStockOpnameStore } = await import('../stock-opname-store.svelte');
    store = useStockOpnameStore();
    const result = await store.createSession({
      scopes: [{ scope_type: 'store', scope_id: 1 }],
      blind_count: true,
    });
    expect(mockCreateStockOpname).toHaveBeenCalledWith({
      scopes: [{ scope_type: 'store', scope_id: 1 }],
      blind_count: true,
    });
    expect(result.id).toBe(9);
  });

  it('opens a session and reloads it', async () => {
    mockOpenStockOpname.mockResolvedValueOnce(undefined);
    mockGetStockOpname.mockResolvedValueOnce({ id: 7, session_number: 'SO-0007', status: 'open' });
    const { useStockOpnameStore } = await import('../stock-opname-store.svelte');
    store = useStockOpnameStore();
    await store.open(7, 'opening');
    expect(mockOpenStockOpname).toHaveBeenCalledWith(7, 'opening');
    expect(mockGetStockOpname).toHaveBeenCalledWith(7);
    expect(store.current?.status).toBe('open');
  });

  it('verifies a session and reloads it', async () => {
    mockVerifySession.mockResolvedValueOnce(undefined);
    mockGetStockOpname.mockResolvedValueOnce({ id: 7, session_number: 'SO-0007', status: 'approved' });
    const { useStockOpnameStore } = await import('../stock-opname-store.svelte');
    store = useStockOpnameStore();
    await store.verify(7, 'all good');
    expect(mockVerifySession).toHaveBeenCalledWith(7, { comment: 'all good' });
    expect(store.current?.status).toBe('approved');
  });

  it('rejects a session and reloads it', async () => {
    mockRejectSession.mockResolvedValueOnce(undefined);
    mockGetStockOpname.mockResolvedValueOnce({ id: 7, session_number: 'SO-0007', status: 'needs_recount' });
    const { useStockOpnameStore } = await import('../stock-opname-store.svelte');
    store = useStockOpnameStore();
    await store.reject(7, 'miscount');
    expect(mockRejectSession).toHaveBeenCalledWith(7, { comment: 'miscount' });
    expect(store.current?.status).toBe('needs_recount');
  });

  it('requests a recount and reloads the session', async () => {
    mockRequestRecount.mockResolvedValueOnce(undefined);
    mockGetStockOpname.mockResolvedValueOnce({ id: 7, session_number: 'SO-0007', status: 'counting' });
    const { useStockOpnameStore } = await import('../stock-opname-store.svelte');
    store = useStockOpnameStore();
    await store.recount(7, 'recount please');
    expect(mockRequestRecount).toHaveBeenCalledWith(7, { comment: 'recount please' });
    expect(store.current?.status).toBe('counting');
  });

  it('resumes counting and reloads the session', async () => {
    mockResumeCounting.mockResolvedValueOnce(undefined);
    mockGetStockOpname.mockResolvedValueOnce({ id: 7, session_number: 'SO-0007', status: 'counting' });
    const { useStockOpnameStore } = await import('../stock-opname-store.svelte');
    store = useStockOpnameStore();
    await store.resume(7);
    expect(mockResumeCounting).toHaveBeenCalledWith(7);
  });

  it('posts an adjustment, returns it, and reloads the session', async () => {
    mockPostAdjustment.mockResolvedValueOnce({ id: 3, adjustment_number: 'IA-0003', session_id: 7 });
    mockGetStockOpname.mockResolvedValueOnce({ id: 7, session_number: 'SO-0007', status: 'posted' });
    const { useStockOpnameStore } = await import('../stock-opname-store.svelte');
    store = useStockOpnameStore();
    const adjustment = await store.post(7, { notes: 'posted' });
    expect(mockPostAdjustment).toHaveBeenCalledWith(7, { notes: 'posted' });
    expect(adjustment.adjustment_number).toBe('IA-0003');
    expect(store.current?.status).toBe('posted');
  });

  it('closes a session and reloads it', async () => {
    mockCloseStockOpname.mockResolvedValueOnce(undefined);
    mockGetStockOpname.mockResolvedValueOnce({ id: 7, session_number: 'SO-0007', status: 'closed' });
    const { useStockOpnameStore } = await import('../stock-opname-store.svelte');
    store = useStockOpnameStore();
    await store.close(7);
    expect(mockCloseStockOpname).toHaveBeenCalledWith(7);
    expect(store.current?.status).toBe('closed');
  });

  it('loads adjustments into the adjustments state', async () => {
    mockListAdjustments.mockResolvedValueOnce({ data: [{ id: 1, adjustment_number: 'IA-0001' }], total: 1 });
    const { useStockOpnameStore } = await import('../stock-opname-store.svelte');
    store = useStockOpnameStore();
    await store.loadAdjustments({ limit: 20, offset: 0 });
    expect(mockListAdjustments).toHaveBeenCalledWith({ limit: 20, offset: 0 }, expect.anything());
    expect(store.adjustments).toHaveLength(1);
    expect(store.adjustmentsTotal).toBe(1);
  });

  it('loads a single adjustment', async () => {
    mockGetAdjustment.mockResolvedValueOnce({ id: 2, adjustment_number: 'IA-0002' });
    const { useStockOpnameStore } = await import('../stock-opname-store.svelte');
    store = useStockOpnameStore();
    await store.loadAdjustment(2);
    expect(mockGetAdjustment).toHaveBeenCalledWith(2);
    expect(store.currentAdjustment?.id).toBe(2);
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
        'so_opened',
        'so_submitted',
        'so_approved',
        'so_posted',
        'so_closed',
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
      expect(ws.on).toHaveBeenCalledTimes(9);
      expect(unsub2()).toBeUndefined();
    });

    it('reloads the list when a session event arrives and no current session is open', async () => {
      mockListStockOpnames.mockResolvedValueOnce({ data: [], total: 0 });
      const ws = makeMockWS();
      mockUseWebSocket.mockReturnValue(ws);
      const { useStockOpnameStore } = await import('../stock-opname-store.svelte');
      store = useStockOpnameStore();
      store.subscribeToWS();

      ws._fire('so_opened', { session_id: 3, session_number: 'SO-0003' });
      await vi.waitFor(() => {
        expect(mockListStockOpnames).toHaveBeenCalled();
      });
      expect(mockGetStockOpname).not.toHaveBeenCalled();
    });

    it('reloads the current session when an event matches its id', async () => {
      mockGetStockOpname
        .mockResolvedValueOnce({ id: 5, session_number: 'SO-0005', status: 'counting', summary: { total_items: 1 } })
        .mockResolvedValueOnce({ id: 5, session_number: 'SO-0005', status: 'verification', summary: { total_items: 1 } });
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

    it('reloads the current session when so_posted arrives for it', async () => {
      mockGetStockOpname
        .mockResolvedValueOnce({ id: 5, session_number: 'SO-0005', status: 'approved', summary: { total_items: 1 } })
        .mockResolvedValueOnce({ id: 5, session_number: 'SO-0005', status: 'posted', summary: { total_items: 1 } });
      const ws = makeMockWS();
      mockUseWebSocket.mockReturnValue(ws);
      const { useStockOpnameStore } = await import('../stock-opname-store.svelte');
      store = useStockOpnameStore();
      await store.loadSession(5);
      store.subscribeToWS();

      ws._fire('so_posted', { session_id: 5, session_number: 'SO-0005' });
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
      expect(ws.on).toHaveBeenCalledTimes(9);
    });
  });
});
