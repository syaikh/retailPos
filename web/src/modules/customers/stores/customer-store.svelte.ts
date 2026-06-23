import { getCustomers } from '../services/customer-service';
import type { Customer, CustomerFilters } from '../types';

let customers = $state<Customer[]>([]);
let total = $state(0);
let loading = $state(false);
let limit = $state(20);
let offset = $state(0);
let searchQuery = $state('');
let statusFilter = $state('all');
let selectedIds = $state(new Set<number>());
let initialized = false;

export function useCustomerStore() {
  async function load(newOffset = offset, newLimit = limit) {
    selectedIds = new Set();
    loading = true;
    try {
      offset = newOffset;
      limit = newLimit;
      const filters: CustomerFilters = { limit, offset, search: searchQuery || undefined };
      if (statusFilter === 'active') filters.isActive = 'true';
      else if (statusFilter === 'inactive') filters.isActive = 'false';
      const result = await getCustomers(filters);
      customers = result.data;
      total = result.total;
    } catch {
      customers = [];
      total = 0;
    } finally {
      loading = false;
    }
  }

  if (!initialized) {
    initialized = true;
  }

  return {
    get customers() { return customers; },
    get total() { return total; },
    get loading() { return loading; },
    get limit() { return limit; },
    get offset() { return offset; },
    get searchQuery() { return searchQuery; },
    set searchQuery(v: string) { searchQuery = v; },
    get statusFilter() { return statusFilter; },
    set statusFilter(v: string) { statusFilter = v; },
    get selectedIds() { return selectedIds; },
    set selectedIds(v: Set<number>) { selectedIds = v; },
    load,
    clearSelection() { selectedIds = new Set(); },
  };
}
