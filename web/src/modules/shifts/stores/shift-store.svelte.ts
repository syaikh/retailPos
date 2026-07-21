import {
  openShift,
  closeShift,
  getActiveShift,
  listShifts,
  getShiftById,
  reviewShift,
  auditShift,
} from '../services/shift-service';
import type { Shift, ShiftFilters } from '../types';

let activeShift = $state<Shift | null>(null);
let shifts = $state<Shift[]>([]);
let total = $state(0);
let loading = $state(false);

let statusFilter = $state('');
let userIdFilter = $state<number | null>(null);
let page = $state(0);
let pageSize = $state(20);
let sortBy = $state('opened_at');
let sortDir = $state<'asc' | 'desc'>('desc');

let abortController: AbortController | null = null;

let initialized = false;

export function useShiftStore() {
  if (!initialized) {
    initialized = true;
  }

  return {
    get activeShift() { return activeShift; },
    get shifts() { return shifts; },
    get total() { return total; },
    get loading() { return loading; },
    set loading(v: boolean) { loading = v; },
    get statusFilter() { return statusFilter; },
    set statusFilter(v: string) { statusFilter = v; },
    get userIdFilter() { return userIdFilter; },
    set userIdFilter(v: number | null) { userIdFilter = v; },
    get page() { return page; },
    set page(v: number) { page = v; },
    get pageSize() { return pageSize; },
    set pageSize(v: number) { pageSize = v; },
    get sortBy() { return sortBy; },
    set sortBy(v: string) { sortBy = v; },
    get sortDir() { return sortDir; },
    set sortDir(v: 'asc' | 'desc') { sortDir = v; },
    get offset() { return page * pageSize; },

    get currentFilters(): ShiftFilters {
      return {
        status: statusFilter,
        userId: userIdFilter,
        limit: pageSize,
        offset: page * pageSize,
        sortBy,
        sortDir,
      };
    },

    async loadActiveShift(signal?: AbortSignal) {
      try {
        const shift = await getActiveShift();
        if (signal?.aborted) return;
        activeShift = shift;
      } catch {
        if (signal?.aborted) return;
        activeShift = null;
      }
    },

    async loadShifts(filters: ShiftFilters) {
      abortController?.abort();
      const controller = new AbortController();
      abortController = controller;

      loading = true;
      try {
        const result = await listShifts(filters, controller.signal);
        if (controller.signal.aborted) return;
        shifts = result.data;
        total = result.total;
      } catch {
        if (controller.signal.aborted) return;
        shifts = [];
        total = 0;
      } finally {
        if (!controller.signal.aborted) loading = false;
      }
    },

    async doOpenShift(storeId: number | null, openingBalance: number) {
      const shift = await openShift(storeId, openingBalance);
      activeShift = shift;
      return shift;
    },

    async doCloseShift(shiftId: number, closingBalance: number, notes: string | null) {
      const shift = await closeShift(shiftId, closingBalance, notes);
      activeShift = null;
      return shift;
    },

    async loadShiftById(id: number) {
      return getShiftById(id);
    },

    async doReviewShift(shiftId: number) {
      const shift = await reviewShift(shiftId);
      const idx = shifts.findIndex(s => s.id === shiftId);
      if (idx !== -1) shifts[idx] = shift;
      if (activeShift?.id === shiftId) activeShift = shift;
      return shift;
    },

    async doAuditShift(shiftId: number, actualBalance: number) {
      return auditShift(shiftId, actualBalance);
    },
  };
}
