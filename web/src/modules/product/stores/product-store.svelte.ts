import { getProducts, getCategories, getBrands, getTaxClasses, getUnitsOfMeasure, getStockThresholds } from '../services/product-service';
import type { Product, Category, Brand, TaxClass, UnitOfMeasure, StockThreshold } from '../types';

let products = $state<Product[]>([]);
let total = $state(0);
let loading = $state(true);
let limit = $state(20);
let offset = $state(0);
let searchQuery = $state('');
let selectedCategories = $state<string[]>(['All']);
let categories = $state<string[]>(['All']);
let categoryObjects = $state<Category[]>([]);
let brands = $state<Brand[]>([]);
let taxClasses = $state<TaxClass[]>([]);
let unitsOfMeasure = $state<UnitOfMeasure[]>([]);
let warningThreshold = $state(10);
let criticalThreshold = $state(5);
let lowStockOnly = $state(false);
let filterStatus = $state('all');
let sortBy = $state('name');
let sortDir = $state('asc');
let selectedIds = $state(new Set<number>());
let initialized = false;

export function useProductStore() {
  async function loadProducts(newOffset?: number, newLimit?: number) {
    if (newOffset !== undefined) offset = newOffset;
    if (newLimit !== undefined) limit = newLimit;
    selectedIds = new Set();
    loading = true;
    try {
      const filteredCategories = selectedCategories.filter(c => c.toLowerCase() !== 'all');
      const filters: {
        limit: number;
        offset: number;
        search?: string;
        category?: string[];
        status?: string;
        maxStock?: number;
      } = { limit, offset, search: searchQuery || undefined };
      if (filteredCategories.length > 0) filters.category = filteredCategories;
      if (lowStockOnly) filters.maxStock = criticalThreshold;
      if (filterStatus !== 'all') filters.status = filterStatus;
      const result = await getProducts(filters);
      products = result.data;
      total = result.total;
    } catch {
      products = [];
      total = 0;
    } finally {
      loading = false;
    }
  }

  async function loadMasterData() {
    try {
      const [catList, brandList, taxList, uomList] = await Promise.all([
        getCategories(), getBrands(), getTaxClasses(), getUnitsOfMeasure(),
      ]);
      categoryObjects = catList;
      categories = ['All', ...catList.map(c => c.name)];
      brands = brandList;
      taxClasses = taxList;
      unitsOfMeasure = uomList;
    } catch {
      // silent fail
    }
  }

  async function loadThresholds() {
    const t = await getStockThresholds();
    warningThreshold = t.warning;
    criticalThreshold = t.critical;
  }

  async function initialize() {
    await Promise.all([loadMasterData(), loadThresholds()]);
    await loadProducts(0, limit);
  }

  if (!initialized) {
    initialized = true;
  }

  return {
    get products() { return products; },
    get total() { return total; },
    get loading() { return loading; },
    set loading(v: boolean) { loading = v; },
    get limit() { return limit; },
    get offset() { return offset; },
    get searchQuery() { return searchQuery; },
    set searchQuery(v: string) { searchQuery = v; },
    get selectedCategories() { return selectedCategories; },
    set selectedCategories(v: string[]) { selectedCategories = v; },
    get categories() { return categories; },
    get categoryObjects() { return categoryObjects; },
    get brands() { return brands; },
    get taxClasses() { return taxClasses; },
    get unitsOfMeasure() { return unitsOfMeasure; },
    get warningThreshold() { return warningThreshold; },
    get criticalThreshold() { return criticalThreshold; },
    get lowStockOnly() { return lowStockOnly; },
    set lowStockOnly(v: boolean) { lowStockOnly = v; },
    get filterStatus() { return filterStatus; },
    set filterStatus(v: string) { filterStatus = v; },
    get sortBy() { return sortBy; },
    set sortBy(v: string) { sortBy = v; },
    get sortDir() { return sortDir; },
    set sortDir(v: string) { sortDir = v; },
    get selectedIds() { return selectedIds; },
    set selectedIds(v: Set<number>) { selectedIds = v; },
    loadProducts,
    loadMasterData,
    loadThresholds,
    initialize,
    clearSelection() { selectedIds = new Set(); },
  };
}
