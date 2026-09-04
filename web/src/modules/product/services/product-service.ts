import apiClient from '$shared/api/http-client';
import type { Product, ProductFilters, Category, Brand, TaxClass, UnitOfMeasure, StockThreshold, ProductFormData, Warehouse } from '../types';

const productCache = new Map<number, Promise<Product | null>>();

export async function getProducts(filters: ProductFilters): Promise<{ data: Product[]; total: number }> {
  const params = new URLSearchParams({
    limit: filters.limit.toString(),
    offset: (filters.offset ?? 0).toString(),
    search: filters.search || '',
  });
  if (filters.category && filters.category.length > 0) {
    params.append('category', filters.category.join(','));
  }
  if (filters.status) {
    params.append('status', filters.status);
  }
  if (filters.maxStock !== undefined) {
    params.append('maxStock', filters.maxStock.toString());
  }
  const r = await apiClient.get(`/products?${params.toString()}`);
  return { data: r.data.data || [], total: r.data.total || 0 };
}

export async function getProductById(id: number): Promise<Product | null> {
  const cached = productCache.get(id);
  if (cached) return cached;
  const p = apiClient.get(`/products/${id}`)
    .then(r => r.data?.data || null)
    .catch(err => { productCache.delete(id); throw err; });
  productCache.set(id, p);
  return p;
}

export async function getProductsByIds(ids: number[]): Promise<Map<number, Product>> {
  const unique = [...new Set(ids.filter(id => id > 0))];
  if (unique.length === 0) return new Map();

  const uncached = unique.filter(id => !productCache.has(id));
  if (uncached.length > 0) {
    const batchPromise = apiClient.get('/products', { params: { ids: uncached.join(',') } })
      .then(r => {
        const products: Product[] = r.data?.data || [];
        for (const product of products) {
          productCache.set(product.id, Promise.resolve(product));
        }
        return products;
      })
      .catch(() => [] as Product[]);
    await batchPromise;
  }

  const results = await Promise.allSettled(unique.map(id => getProductById(id)));
  const map = new Map<number, Product>();
  results.forEach((r, i) => {
    if (r.status === 'fulfilled' && r.value) map.set(unique[i], r.value);
  });
  return map;
}

export function invalidateProductCache(id?: number) {
  if (id) productCache.delete(id);
  else productCache.clear();
}

export function clearProductCache() {
  productCache.clear();
}

export async function getNextSku(): Promise<string> {
  const r = await apiClient.get('/products/next-sku');
  return r.data.data;
}

export async function createProduct(data: ProductFormData & { category_name?: string }): Promise<void> {
  await apiClient.post('/products', data);
}

export async function updateProduct(id: number, data: ProductFormData & { category_name?: string }): Promise<void> {
  await apiClient.put(`/products/${id}`, data);
}

export async function deleteProduct(id: number): Promise<void> {
  await apiClient.delete(`/products/${id}`);
}

export async function bulkUpdateStatus(ids: number[], status: string): Promise<void> {
  await apiClient.post('/products/bulk/status', { ids, status });
}

export async function getCategories(): Promise<Category[]> {
  const r = await apiClient.get('/categories');
  return r.data.data || [];
}

export async function createCategory(name: string): Promise<Category> {
  const r = await apiClient.post('/categories', { name });
  return r.data.data;
}

export async function getBrands(): Promise<Brand[]> {
  const r = await apiClient.get('/brands', { params: { limit: 1000, offset: 0 } });
  return r.data.data || [];
}

export async function createBrand(name: string): Promise<Brand> {
  const r = await apiClient.post('/brands', { name });
  return r.data.data;
}

export async function getTaxClasses(): Promise<TaxClass[]> {
  const r = await apiClient.get('/tax-classes');
  return r.data.data || [];
}

export async function getWarehouses(): Promise<Warehouse[]> {
  const r = await apiClient.get('/warehouses');
  return r.data.data || [];
}

export interface ProductOption {
  id: number;
  sku: string;
  name: string;
}

export async function getProductOptions(): Promise<ProductOption[]> {
  const r = await apiClient.get('/products/options');
  return r.data.data || [];
}

export async function getUnitsOfMeasure(): Promise<UnitOfMeasure[]> {
  const r = await apiClient.get('/units-of-measure', { params: { limit: 1000, offset: 0 } });
  return r.data.data || [];
}

export async function getStockThresholds(): Promise<StockThreshold> {
  try {
    const r = await apiClient.get('/stock-thresholds');
    return { warning: r.data.warning ?? 10, critical: r.data.critical ?? 5 };
  } catch {
    return { warning: 10, critical: 5 };
  }
}


export async function adjustStock(productId: number, quantityChange: number, notes: string): Promise<void> {
  await apiClient.post('/inventory/adjust', {
    product_id: productId,
    quantity_change: quantityChange,
    notes,
  });
}
