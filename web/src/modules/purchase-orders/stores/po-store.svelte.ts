import { getPurchaseOrders, getPurchaseOrderById, createPurchaseOrder, updatePurchaseOrder, getReceipts, createGoodsReceipt } from '../services/po-service';
import type { PurchaseOrder, PurchaseOrderFilters } from '../types';

let purchaseOrdersData = $state<PurchaseOrder[]>([]);
let total = $state(0);
let loading = $state(true);

let searchQuery = $state('');
let statusFilter = $state('');
let supplierFilter = $state('');
let startDate = $state('');
let endDate = $state('');
let page = $state(0);
let pageSize = $state(20);
let sortBy = $state('created_at');
let sortDir = $state<'asc' | 'desc'>('desc');

let selectedPO = $state<PurchaseOrder | null>(null);
let receipts = $state<any[]>([]);

let initialized = false;

export function usePurchaseOrderStore() {
  if (!initialized) {
    initialized = true;
  }

  return {
    get purchaseOrdersData() { return purchaseOrdersData; },
    get total() { return total; },
    get loading() { return loading; },
    set loading(v: boolean) { loading = v; },
    get searchQuery() { return searchQuery; },
    set searchQuery(v: string) { searchQuery = v; },
    get statusFilter() { return statusFilter; },
    set statusFilter(v: string) { statusFilter = v; },
    get supplierFilter() { return supplierFilter; },
    set supplierFilter(v: string) { supplierFilter = v; },
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
    get selectedPO() { return selectedPO; },
    set selectedPO(v: PurchaseOrder | null) { selectedPO = v; },
    get receipts() { return receipts; },
    set receipts(v: any[]) { receipts = v; },

    get currentFilters(): PurchaseOrderFilters {
      return {
        search: searchQuery || undefined,
        status: statusFilter || undefined,
        supplier_id: supplierFilter || undefined,
        startDate: startDate || undefined,
        endDate: endDate || undefined,
        page,
        pageSize,
        sortBy,
        sortDir,
      };
    },

    async load(filters: PurchaseOrderFilters, signal?: AbortSignal) {
      loading = true;
      try {
        const result = await getPurchaseOrders(filters, signal);
        if (signal?.aborted) return;
        purchaseOrdersData = result.data;
        total = result.total;
      } catch {
        if (signal?.aborted) return;
        purchaseOrdersData = [];
        total = 0;
      } finally {
        if (signal?.aborted) return;
        loading = false;
      }
    },

    async loadDetail(id: number) {
      loading = true;
      try {
        const po = await getPurchaseOrderById(id);
        if (po) {
          selectedPO = po;
        }
      } finally {
        loading = false;
      }
    },

    async loadReceipts(poId: number) {
      receipts = await getReceipts(poId);
    },

    async create(po: any) {
      return createPurchaseOrder(po);
    },

    async update(id: number, po: any) {
      return updatePurchaseOrder(id, po);
    },

    async receive(gr: any) {
      return createGoodsReceipt(gr);
    },
  };
}
