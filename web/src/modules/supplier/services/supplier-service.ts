import { apiFetch } from '$shared/api/http-client';
import type { Supplier, CreateSupplierPayload, UpdateSupplierPayload, ProductSupplier } from '../types';

export interface SupplierListParams {
  limit: number;
  offset: number;
  search?: string;
  is_active?: boolean;
  sort_by?: string;
  sort_dir?: string;
}

export interface SupplierListResponse {
  data: Supplier[];
  total: number;
}

export async function getSuppliers(params: SupplierListParams): Promise<SupplierListResponse> {
  const urlParams = new URLSearchParams({
    limit: params.limit.toString(),
    offset: params.offset.toString(),
  });
  if (params.search) urlParams.append('search', params.search);
  if (params.is_active !== undefined) urlParams.append('is_active', params.is_active.toString());
  if (params.sort_by) urlParams.append('sort_by', params.sort_by);
  if (params.sort_dir) urlParams.append('sort_dir', params.sort_dir);

  const r = await apiFetch(`/api/suppliers?${urlParams.toString()}`);
  if (r.ok) {
    const data = await r.json();
    return { data: data.data || [], total: data.total || 0 };
  }
  return { data: [], total: 0 };
}

export async function getSupplier(id: number): Promise<Supplier | null> {
  const r = await apiFetch(`/api/suppliers/${id}`);
  if (r.ok) {
    const data = await r.json();
    return data.data || null;
  }
  return null;
}

export async function createSupplier(payload: CreateSupplierPayload): Promise<boolean> {
  const r = await apiFetch('/api/suppliers', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
  return r.ok;
}

export async function updateSupplier(id: number, payload: UpdateSupplierPayload): Promise<boolean> {
  const r = await apiFetch(`/api/suppliers/${id}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
  });
  return r.ok;
}

export async function deleteSupplier(id: number): Promise<boolean> {
  const r = await apiFetch(`/api/suppliers/${id}`, { method: 'DELETE' });
  return r.ok;
}

export async function getSuppliersByProduct(productId: number): Promise<ProductSupplier[]> {
  const r = await apiFetch(`/api/products/${productId}/suppliers`);
  if (r.ok) {
    const data = await r.json();
    return data.data || [];
  }
  return [];
}

export async function getProductsBySupplier(supplierId: number): Promise<ProductSupplier[]> {
  const r = await apiFetch(`/api/suppliers/${supplierId}/products`);
  if (r.ok) {
    const data = await r.json();
    return data.data || [];
  }
  return [];
}

export async function linkProduct(supplierId: number, payload: { product_id: number; unit_cost: number; lead_time_days: number; is_preferred: boolean }): Promise<boolean> {
  const r = await apiFetch(`/api/suppliers/${supplierId}/products`, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
  return r.ok;
}

export async function unlinkProduct(supplierId: number, productId: number): Promise<boolean> {
  const r = await apiFetch(`/api/suppliers/${supplierId}/products/${productId}`, { method: 'DELETE' });
  return r.ok;
}

export async function bulkUpdateSuppliers(ids: number[], isActive: boolean): Promise<number> {
  const r = await apiFetch('/api/suppliers/bulk', {
    method: 'PUT',
    body: JSON.stringify({ ids, is_active: isActive }),
  });
  if (r.ok) {
    const data = await r.json();
    return data.updated || 0;
  }
  return 0;
}

export async function bulkDeleteSuppliers(ids: number[]): Promise<number> {
  const r = await apiFetch('/api/suppliers/bulk', {
    method: 'DELETE',
    body: JSON.stringify({ ids }),
  });
  if (r.ok) {
    const data = await r.json();
    return data.deleted || 0;
  }
  return 0;
}
