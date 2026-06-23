import { getSalesHistory } from '../services/sales-service';
import type { Sale, SaleFilters } from '../types';

let salesData = $state<Sale[]>([]);
let total = $state(0);
let loading = $state(true);
let limit = $state(20);
let offset = $state(0);
let searchQuery = $state('');
let sortBy = $state('created_at');
let sortDir = $state('DESC');
let initialized = false;

export function useSalesStore() {
  async function load(filters: SaleFilters) {
    loading = true;
    try {
      const result = await getSalesHistory(filters);
      salesData = result.data;
      total = result.total;
      limit = filters.limit;
      offset = filters.offset;
      sortBy = filters.sortBy || 'created_at';
      sortDir = filters.sortDir || 'DESC';
    } catch {
      salesData = [];
      total = 0;
    } finally {
      loading = false;
    }
  }

  if (!initialized) {
    initialized = true;
  }

  return {
    get salesData() { return salesData; },
    get total() { return total; },
    get loading() { return loading; },
    get limit() { return limit; },
    get offset() { return offset; },
    get searchQuery() { return searchQuery; },
    set searchQuery(v: string) { searchQuery = v; },
    get sortBy() { return sortBy; },
    get sortDir() { return sortDir; },
    load,
  };
}
