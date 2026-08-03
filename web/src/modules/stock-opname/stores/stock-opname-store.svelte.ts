import {
  createStockOpname,
  listStockOpnames,
  getStockOpname,
  openStockOpname,
  cancelStockOpname,
  assignCounter,
  getAssignableUsers,
  getAssignments,
  reassignCounter,
  saveCount,
  getCountHistory,
  submitSession,
  startCounting,
  verifySession,
  rejectSession,
  requestRecount,
  resumeCounting,
  postAdjustment,
  closeStockOpname,
  getSessionSummary,
  listAdjustments,
  getAdjustment,
  exportStockOpname,
} from '../services/stock-opname-service';
import { useWebSocket } from '$shared/api/websocket';
import type {
  StockOpnameSession,
  StockOpnameAssignment,
  AssignableUser,
  CountRecord,
  SessionSummary,
  StockOpnameFilters,
  CreateStockOpnamePayload,
  AssignPayload,
  ReassignPayload,
  SaveCountPayload,
  PostAdjustmentPayload,
  Adjustment,
} from '../types';

let sessions = $state<StockOpnameSession[]>([]);
let total = $state(0);
let loading = $state(false);
let current = $state<StockOpnameSession | null>(null);
let currentAssignments = $state<StockOpnameAssignment[]>([]);
let assignableUsers = $state<AssignableUser[]>([]);
let assignableLoading = $state(false);
let currentSummary = $state<SessionSummary | null>(null);
let countHistory = $state<Record<number, CountRecord[]>>({});

let adjustments = $state<Adjustment[]>([]);
let adjustmentsTotal = $state(0);
let adjustmentsLoading = $state(false);
let currentAdjustment = $state<Adjustment | null>(null);

let statusFilter = $state('');
let searchFilter = $state('');
let page = $state(0);
let pageSize = $state(20);

let abortController: AbortController | null = null;
let wsSubscribed = false;

export function useStockOpnameStore() {
  return {
    get sessions() { return sessions; },
    get total() { return total; },
    get loading() { return loading; },
    get current() { return current; },
    get currentAssignments() { return currentAssignments; },
    get assignableUsers() { return assignableUsers; },
    get assignableLoading() { return assignableLoading; },
    get currentSummary() { return currentSummary; },
    get adjustments() { return adjustments; },
    get adjustmentsTotal() { return adjustmentsTotal; },
    get adjustmentsLoading() { return adjustmentsLoading; },
    get currentAdjustment() { return currentAdjustment; },
    get statusFilter() { return statusFilter; },
    set statusFilter(v: string) { statusFilter = v; },
    get searchFilter() { return searchFilter; },
    set searchFilter(v: string) { searchFilter = v; },
    get page() { return page; },
    set page(v: number) { page = v; },
    get pageSize() { return pageSize; },
    set pageSize(v: number) { pageSize = v; },
    get offset() { return page * pageSize; },

    get currentFilters(): StockOpnameFilters {
      return {
        status: statusFilter,
        search: searchFilter,
        limit: pageSize,
        offset: page * pageSize,
      };
    },

    async loadSessions(filters: StockOpnameFilters) {
      abortController?.abort();
      const controller = new AbortController();
      abortController = controller;

      loading = true;
      try {
        const result = await listStockOpnames(filters, controller.signal);
        if (controller.signal.aborted) return;
        sessions = result.data;
        total = result.total;
      } catch {
        if (controller.signal.aborted) return;
        sessions = [];
        total = 0;
      } finally {
        if (!controller.signal.aborted) loading = false;
      }
    },

    async createSession(payload: CreateStockOpnamePayload): Promise<StockOpnameSession> {
      const session = await createStockOpname(payload);
      return session;
    },

    async loadSession(id: number) {
      loading = true;
      try {
        current = await getStockOpname(id);
        currentAssignments = current?.assignments ?? [];
        currentSummary = current?.summary ?? null;
        if (current && !current.summary) {
          currentSummary = await getSessionSummary(id);
        }
      } finally {
        loading = false;
      }
    },

    async open(id: number, comment: string) {
      await openStockOpname(id, comment);
      await this.loadSession(id);
    },

    async cancelSession(id: number) {
      await cancelStockOpname(id);
      await this.loadSession(id);
    },

    async loadAssignableUsers(search?: string) {
      assignableLoading = true;
      try {
        assignableUsers = await getAssignableUsers(search);
      } catch {
        assignableUsers = [];
      } finally {
        assignableLoading = false;
      }
    },

    async assign(id: number, payload: AssignPayload) {
      await assignCounter(id, payload);
      currentAssignments = await getAssignments(id);
    },

    async reassign(id: number, assignmentId: number, payload: ReassignPayload) {
      await reassignCounter(id, assignmentId, payload);
      currentAssignments = await getAssignments(id);
    },

    async saveCount(itemId: number, payload: SaveCountPayload) {
      await saveCount(itemId, payload);
      const item = current?.items?.find(i => i.id === itemId);
      if (item) {
        item.physical_qty = payload.physical_qty;
        item.status = 'counted';
      }
      const summary = await getSessionSummary(current!.id);
      currentSummary = summary;
    },

    async getCountHistory(itemId: number): Promise<CountRecord[]> {
      if (countHistory[itemId]) return countHistory[itemId];
      const history = await getCountHistory(itemId);
      countHistory = { ...countHistory, [itemId]: history };
      return history;
    },

    async submit(id: number) {
      await submitSession(id);
      await this.loadSession(id);
    },

    async start(id: number) {
      await startCounting(id);
      await this.loadSession(id);
    },

    async verify(id: number, comment: string) {
      await verifySession(id, { comment });
      await this.loadSession(id);
    },

    async reject(id: number, comment: string) {
      await rejectSession(id, { comment });
      await this.loadSession(id);
    },

    async recount(id: number, comment: string) {
      await requestRecount(id, { comment });
      await this.loadSession(id);
    },

    async resume(id: number) {
      await resumeCounting(id);
      await this.loadSession(id);
    },

    async post(id: number, payload: PostAdjustmentPayload): Promise<Adjustment> {
      const adjustment = await postAdjustment(id, payload);
      await this.loadSession(id);
      return adjustment;
    },

    async close(id: number) {
      await closeStockOpname(id);
      await this.loadSession(id);
    },

    async loadAdjustments(filters: { status?: string; search?: string; limit: number; offset: number }) {
      abortController?.abort();
      const controller = new AbortController();
      abortController = controller;

      adjustmentsLoading = true;
      try {
        const result = await listAdjustments(filters, controller.signal);
        if (controller.signal.aborted) return;
        adjustments = result.data;
        adjustmentsTotal = result.total;
      } catch {
        if (controller.signal.aborted) return;
        adjustments = [];
        adjustmentsTotal = 0;
      } finally {
        if (!controller.signal.aborted) adjustmentsLoading = false;
      }
    },

    async loadAdjustment(id: number) {
      currentAdjustment = await getAdjustment(id);
    },

    async exportCSV(id: number): Promise<Blob> {
      return exportStockOpname(id);
    },

    subscribeToWS(onStatus?: (data: any) => void): () => void {
      if (wsSubscribed) return () => {};
      wsSubscribed = true;
      const ws = useWebSocket();
      const reload = (data: any) => {
        this.loadSessions(this.currentFilters);
        const cur = current;
        if (cur && cur.id === data.session_id) {
          this.loadSession(cur.id);
        }
        onStatus?.(data);
      };
      const unsubs = [
        ws.on('so_created', reload),
        ws.on('so_opened', reload),
        ws.on('so_submitted', reload),
        ws.on('so_approved', reload),
        ws.on('so_posted', reload),
        ws.on('so_closed', reload),
        ws.on('so_rejected', reload),
        ws.on('so_needs_recount', reload),
        ws.on('so_cancelled', reload),
      ];
      return () => {
        unsubs.forEach(fn => fn());
        wsSubscribed = false;
      };
    },

    clearCurrent() {
      current = null;
      currentAssignments = [];
      assignableUsers = [];
      currentSummary = null;
      countHistory = {};
    },
  };
}
