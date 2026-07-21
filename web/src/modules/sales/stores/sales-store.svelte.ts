import { getSalesHistory, getPaymentMethods } from '../services/sales-service';
import type { Sale, SaleFilters, FilterState } from '../types';

let salesData = $state<Sale[]>([]);
let total = $state(0);
let loading = $state(true);

let searchQuery = $state('');
let paymentMethods = $state<string[]>([]);
let minTotal = $state<number | null>(null);
let maxTotal = $state<number | null>(null);
let dateRange = $state('last30d');
let startDate = $state('');
let endDate = $state('');
let page = $state(0);
let pageSize = $state(20);
let sortBy = $state('created_at');
let cashierId = $state<number | null>(null);
let sortDir = $state<'asc' | 'desc'>('desc');

let paymentMethodOptions = $state<{ code: string; name: string }[]>([]);
let initialized = false;

export function useSalesStore() {
  if (!initialized) {
    initialized = true;
  }

  return {
    get salesData() { return salesData; },
    get total() { return total; },
    get loading() { return loading; },
    set loading(v: boolean) { loading = v; },
    get searchQuery() { return searchQuery; },
    set searchQuery(v: string) { searchQuery = v; },
    get paymentMethods() { return paymentMethods; },
    set paymentMethods(v: string[]) { paymentMethods = v; },
    get minTotal() { return minTotal; },
    set minTotal(v: number | null) { minTotal = v; },
    get maxTotal() { return maxTotal; },
    set maxTotal(v: number | null) { maxTotal = v; },
    get dateRange() { return dateRange; },
    set dateRange(v: string) { dateRange = v; },
    get startDate() { return startDate; },
    set startDate(v: string) { startDate = v; },
    get endDate() { return endDate; },
    set endDate(v: string) { endDate = v; },
    get page() { return page; },
    set page(v: number) { page = v; },
    get pageSize() { return pageSize; },
    set pageSize(v: number) { pageSize = v; },
    get limit() { return pageSize; },
    set limit(v: number) { pageSize = v; },
    get offset() { return page * pageSize; },
    set offset(v: number) { page = Math.floor(v / pageSize); },
    get sortBy() { return sortBy; },
    set sortBy(v: string) { sortBy = v; },
    get sortDir(): 'asc' | 'desc' { return sortDir; },
    set sortDir(v: 'asc' | 'desc') { sortDir = v; },
    get cashierId() { return cashierId; },
    set cashierId(v: number | null) { cashierId = v; },
    get paymentMethodOptions() { return paymentMethodOptions; },
    set paymentMethodOptions(v: { code: string; name: string }[]) { paymentMethodOptions = v; },

    get currentFilters(): SaleFilters {
      return {
        startDate,
        endDate,
        limit: pageSize,
        offset: page * pageSize,
        search: searchQuery || undefined,
        sortBy,
        sortDir,
        paymentMethods: paymentMethods.length > 0 ? paymentMethods : undefined,
        minTotal: minTotal !== null && minTotal > 0 ? minTotal : undefined,
        maxTotal: maxTotal !== null && maxTotal < 50000000 ? maxTotal : undefined,
        cashierId: cashierId ?? undefined,
        dateRange,
      };
    },

    get filterState(): FilterState {
      return {
        searchQuery,
        paymentMethods,
        minTotal,
        maxTotal,
        dateRange,
        startDate,
        endDate,
        page,
        pageSize,
        sortBy,
        sortDir,
      };
    },

    async load(filters: SaleFilters, signal?: AbortSignal) {
      loading = true;
      try {
        const result = await getSalesHistory(filters, signal);
        if (signal?.aborted) return;
        salesData = result.data;
        total = result.total;
      } catch {
        if (signal?.aborted) return;
        salesData = [];
        total = 0;
      } finally {
        if (signal?.aborted) return;
        loading = false;
      }
    },

    async loadPaymentMethods(signal?: AbortSignal) {
      try {
        const methods = await getPaymentMethods(signal);
        if (signal?.aborted) return;
        paymentMethodOptions = methods;
      } catch {
        if (signal?.aborted) return;
      }
    },
  };
}
