import type { SaleFilters } from '../types';

export interface QueryManagerConfig {
  getFilters: () => SaleFilters;
  fetch: (filters: SaleFilters, signal: AbortSignal) => Promise<void>;
  searchDebounceMs?: number;
  amountDebounceMs?: number;
  batchWindowMs?: number;
}

export interface QueryManager {
  destroy: () => void;
  isPending: () => boolean;
  notify: (filters: SaleFilters, changed: Set<string>) => void;
}

export function createQueryManager(config: QueryManagerConfig): QueryManager {
  const searchDebounceMs = config.searchDebounceMs ?? 400;
  const amountDebounceMs = config.amountDebounceMs ?? 600;
  const batchWindowMs = config.batchWindowMs ?? 50;

  let abortController: AbortController | null = null;
  let batchTimer: ReturnType<typeof setTimeout> | null = null;
  let searchTimer: ReturnType<typeof setTimeout> | null = null;
  let amountTimer: ReturnType<typeof setTimeout> | null = null;
  let previousFiltersJson = '';
  let pendingFilters: SaleFilters | null = null;
  let destroyed = false;

  function clearAllTimers() {
    if (batchTimer) { clearTimeout(batchTimer); batchTimer = null; }
    if (searchTimer) { clearTimeout(searchTimer); searchTimer = null; }
    if (amountTimer) { clearTimeout(amountTimer); amountTimer = null; }
  }

  function isDuplicate(filters: SaleFilters): boolean {
    const json = JSON.stringify(filters);
    if (json === previousFiltersJson) return true;
    previousFiltersJson = json;
    return false;
  }

  async function executeFetch(filters: SaleFilters) {
    if (destroyed) return;
    if (isDuplicate(filters)) return;
    pendingFilters = null;

    abortController?.abort();
    abortController = new AbortController();

    try {
      await config.fetch(filters, abortController.signal);
    } catch (e) {
      if (e instanceof DOMException && e.name === 'AbortError') return;
      throw e;
    }
  }

  function doFetch(filters: SaleFilters) {
    pendingFilters = null;
    executeFetch(filters);
  }

  function scheduleBatch() {
    if (batchTimer) clearTimeout(batchTimer);
    batchTimer = setTimeout(() => {
      batchTimer = null;
      if (searchTimer || amountTimer) return;
      if (pendingFilters) {
        const f = pendingFilters;
        pendingFilters = null;
        doFetch(f);
      }
    }, batchWindowMs);
  }

  function notify(filters: SaleFilters, changed: Set<string>) {
    if (destroyed) return;
    pendingFilters = filters;

    if (!previousFiltersJson) {
      doFetch(filters);
      previousFiltersJson = JSON.stringify(filters);
      return;
    }

    const hasSearch = changed.has('searchQuery');
    const hasAmount = changed.has('minTotal') || changed.has('maxTotal');
    const hasImmediate = changed.has('paymentMethods') || changed.has('dateRange') || changed.has('startDate') || changed.has('endDate') || changed.has('page') || changed.has('pageSize') || changed.has('sortBy') || changed.has('sortDir');

    if (hasSearch) {
      if (searchTimer) clearTimeout(searchTimer);
      searchTimer = setTimeout(() => {
        searchTimer = null;
        if (batchTimer) { clearTimeout(batchTimer); batchTimer = null; }
        if (pendingFilters) {
          const f = pendingFilters;
          pendingFilters = null;
          doFetch(f);
        }
      }, searchDebounceMs);
    }

    if (hasAmount) {
      if (amountTimer) clearTimeout(amountTimer);
      amountTimer = setTimeout(() => {
        amountTimer = null;
        if (batchTimer) { clearTimeout(batchTimer); batchTimer = null; }
        if (pendingFilters) {
          const f = pendingFilters;
          pendingFilters = null;
          doFetch(f);
        }
      }, amountDebounceMs);
    }

    if (hasImmediate && !hasSearch && !hasAmount) {
      scheduleBatch();
    } else if (hasImmediate && (hasSearch || hasAmount)) {
      let maxTimer = 0;
      if (hasSearch) maxTimer = Math.max(maxTimer, searchDebounceMs);
      if (hasAmount) maxTimer = Math.max(maxTimer, amountDebounceMs);
      if (batchTimer) clearTimeout(batchTimer);
      batchTimer = setTimeout(() => {
        batchTimer = null;
        if (searchTimer || amountTimer) return;
        if (pendingFilters) {
          const f = pendingFilters;
          pendingFilters = null;
          doFetch(f);
        }
      }, Math.min(maxTimer, batchWindowMs));
    }
  }

  return {
    destroy() {
      destroyed = true;
      clearAllTimers();
      abortController?.abort();
      abortController = null;
    },
    isPending() {
      return batchTimer !== null || searchTimer !== null || amountTimer !== null;
    },
    notify,
  };
}
