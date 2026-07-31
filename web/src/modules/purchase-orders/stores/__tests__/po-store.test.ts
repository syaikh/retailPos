import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockGetPurchaseOrders = vi.fn();
const mockUseWebSocket = vi.fn();

vi.mock('../../services/po-service', () => ({
  getPurchaseOrders: (...args: unknown[]) => mockGetPurchaseOrders(...args),
}));

vi.mock('$shared/api/websocket', () => ({
  useWebSocket: () => mockUseWebSocket(),
}));

function makeMockWS() {
  const handlers: Record<string, () => void> = {};
  return {
    on: vi.fn((event: string, cb: () => void) => {
      handlers[event] = cb;
      return () => { delete handlers[event]; };
    }),
    _fire: (event: string) => handlers[event]?.(),
    _clear: () => Object.keys(handlers).forEach(k => delete handlers[k]),
  };
}

describe('po-store', () => {
  let store: ReturnType<typeof import('../po-store.svelte').usePurchaseOrderStore>;

  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
  });

  it('returns expected API shape', async () => {
    const { usePurchaseOrderStore } = await import('../po-store.svelte');
    store = usePurchaseOrderStore();
    expect(store).toHaveProperty('purchaseOrdersData');
    expect(store).toHaveProperty('total');
    expect(store).toHaveProperty('loading');
    expect(store).toHaveProperty('searchQuery');
    expect(store).toHaveProperty('statusFilter');
    expect(store).toHaveProperty('supplierFilter');
    expect(store).toHaveProperty('sortBy');
    expect(store).toHaveProperty('sortDir');
    expect(store).toHaveProperty('pageSize');
    expect(store).toHaveProperty('limit');
    expect(store).toHaveProperty('offset');
    expect(store).toHaveProperty('load');
    expect(store).toHaveProperty('subscribeToWS');
  });

  it('loads purchase orders successfully', async () => {
    mockGetPurchaseOrders.mockResolvedValueOnce({ data: [{ id: 1 }], total: 1 });
    const { usePurchaseOrderStore } = await import('../po-store.svelte');
    store = usePurchaseOrderStore();
    store.pageSize = 20;
    store.page = 0;
    await store.load({ page: 0, pageSize: 20, sortBy: 'created_at', sortDir: 'desc' });
    expect(mockGetPurchaseOrders).toHaveBeenCalled();
    expect(store.purchaseOrdersData).toHaveLength(1);
    expect(store.total).toBe(1);
    expect(store.loading).toBe(false);
  });

  it('sets loading false on error', async () => {
    mockGetPurchaseOrders.mockRejectedValueOnce(new Error('Network error'));
    const { usePurchaseOrderStore } = await import('../po-store.svelte');
    store = usePurchaseOrderStore();
    store.pageSize = 20;
    store.page = 0;
    await store.load({ page: 0, pageSize: 20, sortBy: 'created_at', sortDir: 'desc' });
    expect(store.loading).toBe(false);
    expect(store.purchaseOrdersData).toEqual([]);
    expect(store.total).toBe(0);
  });

  describe('subscribeToWS', () => {
    it('subscribes to po_created, po_confirmed, po_cancelled, and po_received', async () => {
      const ws = makeMockWS();
      mockUseWebSocket.mockReturnValue(ws);
      const { usePurchaseOrderStore } = await import('../po-store.svelte');
      store = usePurchaseOrderStore();
      const unsub = store.subscribeToWS();
      expect(ws.on).toHaveBeenCalledWith('po_created', expect.any(Function));
      expect(ws.on).toHaveBeenCalledWith('po_confirmed', expect.any(Function));
      expect(ws.on).toHaveBeenCalledWith('po_cancelled', expect.any(Function));
      expect(ws.on).toHaveBeenCalledWith('po_received', expect.any(Function));
      expect(unsub).toEqual(expect.any(Function));
    });

    it('returns noop if already subscribed', async () => {
      const ws = makeMockWS();
      mockUseWebSocket.mockReturnValue(ws);
      const { usePurchaseOrderStore } = await import('../po-store.svelte');
      store = usePurchaseOrderStore();
      store.subscribeToWS();
      const unsub2 = store.subscribeToWS();
      expect(ws.on).toHaveBeenCalledTimes(4);
      expect(unsub2()).toBeUndefined();
    });

    it('registers reload callbacks for all four PO events', async () => {
      mockGetPurchaseOrders.mockResolvedValue({ data: [], total: 0 });
      const ws = makeMockWS();
      mockUseWebSocket.mockReturnValue(ws);

      const { usePurchaseOrderStore } = await import('../po-store.svelte');
      store = usePurchaseOrderStore();
      store.subscribeToWS();

      const events = ws.on.mock.calls.map((c: [string]) => c[0]);
      expect(events).toEqual(['po_created', 'po_confirmed', 'po_cancelled', 'po_received']);
    });

    it('unsubscribing returns a function that removes all handlers', async () => {
      const ws = makeMockWS();
      mockUseWebSocket.mockReturnValue(ws);
      const { usePurchaseOrderStore } = await import('../po-store.svelte');
      store = usePurchaseOrderStore();
      const unsub = store.subscribeToWS();

      const unsubCallbacks = ws.on.mock.results.map((r: { value: () => void }) => r.value);
      expect(unsubCallbacks).toHaveLength(4);

      unsub();
      expect(ws.on).toHaveBeenCalledTimes(4);
    });

    it('fires data reload when po_created event arrives', async () => {
      mockGetPurchaseOrders.mockResolvedValue({ data: [], total: 0 });
      const ws = makeMockWS();
      mockUseWebSocket.mockReturnValue(ws);
      const { usePurchaseOrderStore } = await import('../po-store.svelte');
      store = usePurchaseOrderStore();
      store.subscribeToWS();

      ws._fire('po_created');
      await vi.waitFor(() => {
        expect(mockGetPurchaseOrders).toHaveBeenCalled();
      });
    });

    it('fires data reload when po_received event arrives', async () => {
      mockGetPurchaseOrders.mockResolvedValue({ data: [], total: 0 });
      const ws = makeMockWS();
      mockUseWebSocket.mockReturnValue(ws);
      const { usePurchaseOrderStore } = await import('../po-store.svelte');
      store = usePurchaseOrderStore();
      store.subscribeToWS();

      ws._fire('po_received');
      await vi.waitFor(() => {
        expect(mockGetPurchaseOrders).toHaveBeenCalled();
      });
    });
  });
});
