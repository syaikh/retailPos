import apiClient from '$shared/api/http-client';
import type { Product, ProductFilters, Category, Brand, TaxClass, UnitOfMeasure, StockThreshold, ProductFormData } from '../types';

export async function getProducts(filters: ProductFilters): Promise<{ data: Product[]; total: number }> {
  const params = new URLSearchParams({
    limit: filters.limit.toString(),
    offset: filters.offset.toString(),
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
  try {
    const r = await apiClient.get(`/products/${id}`);
    return r.data || null;
  } catch {
    return null;
  }
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

export async function getBrands(): Promise<Brand[]> {
  const r = await apiClient.get('/brands');
  return r.data.data || [];
}

export async function getTaxClasses(): Promise<TaxClass[]> {
  const r = await apiClient.get('/tax-classes');
  return r.data.data || [];
}

export async function getUnitsOfMeasure(): Promise<UnitOfMeasure[]> {
  const r = await apiClient.get('/units-of-measure');
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

export async function exportProducts(format: 'csv' | 'xlsx'): Promise<void> {
  const token = sessionStorage.getItem('access_token');
  window.open(`/api/products/export?format=${format}&token=${token}`, '_blank');
}

export async function importProducts(file: File): Promise<any> {
  const formData = new FormData();
  formData.append('file', file);
  const r = await apiClient.post('/products/import', formData);
  return r.data;
}

export async function adjustStock(productId: number, quantityChange: number, notes: string): Promise<void> {
  await apiClient.post('/inventory/adjust', {
    product_id: productId,
    quantity_change: quantityChange,
    notes,
  });
}
