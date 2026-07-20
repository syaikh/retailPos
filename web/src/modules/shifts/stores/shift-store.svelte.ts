import {
  openShift,
  closeShift,
  getActiveShift,
  listShifts,
  getShiftById,
} from '../services/shift-service';
import type { Shift, ShiftFilters } from '../types';

let activeShift = $state<Shift | null>(null);
let shifts = $state<Shift[]>([]);
let total = $state(0);
let loading = $state(false);

let statusFilter = $state('');
let page = $state(0);
let pageSize = $state(20);
let sortBy = $state('opened_at');
let sortDir = $state<'asc' | 'desc'>('desc');

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
        userId: null,
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

    async loadShifts(filters: ShiftFilters, signal?: AbortSignal) {
      loading = true;
      try {
        const result = await listShifts(filters);
        if (signal?.aborted) return;
        shifts = result.data;
        total = result.total;
      } catch {
        if (signal?.aborted) return;
        shifts = [];
        total = 0;
      } finally {
        if (signal?.aborted) return;
        loading = false;
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
  };
}
