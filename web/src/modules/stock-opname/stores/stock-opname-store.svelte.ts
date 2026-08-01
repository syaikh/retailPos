import {
  createStockOpname,
  listStockOpnames,
  getStockOpname,
  cancelStockOpname,
  assignCounter,
  getAssignments,
  reassignCounter,
  saveCount,
  getCountHistory,
  submitSession,
  startCounting,
  approveSession,
  rejectSession,
  requestRecount,
  resumeCounting,
  getSessionSummary,
  exportStockOpname,
} from '../services/stock-opname-service';
import type {
  StockOpnameSession,
  StockOpnameAssignment,
  CountRecord,
  SessionSummary,
  StockOpnameFilters,
  CreateStockOpnamePayload,
  AssignPayload,
  ReassignPayload,
  SaveCountPayload,
} from '../types';

let sessions = $state<StockOpnameSession[]>([]);
let total = $state(0);
let loading = $state(false);
let current = $state<StockOpnameSession | null>(null);
let currentAssignments = $state<StockOpnameAssignment[]>([]);
let currentSummary = $state<SessionSummary | null>(null);
let countHistory = $state<Record<number, CountRecord[]>>({});

let statusFilter = $state('');
let searchFilter = $state('');
let page = $state(0);
let pageSize = $state(20);

let abortController: AbortController | null = null;

export function useStockOpnameStore() {
  return {
    get sessions() { return sessions; },
    get total() { return total; },
    get loading() { return loading; },
    get current() { return current; },
    get currentAssignments() { return currentAssignments; },
    get currentSummary() { return currentSummary; },
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

    async cancelSession(id: number) {
      await cancelStockOpname(id);
      await this.loadSession(id);
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

    async approve(id: number, comment: string) {
      await approveSession(id, { comment });
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

    async exportCSV(id: number): Promise<Blob> {
      return exportStockOpname(id);
    },

    clearCurrent() {
      current = null;
      currentAssignments = [];
      currentSummary = null;
      countHistory = {};
    },
  };
}
