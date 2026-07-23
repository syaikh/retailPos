import { apiFetch } from '$shared/api/http-client';
import type { MasterCategory, MasterBrand, MasterUnitOfMeasure, CreateCategoryPayload, UpdateCategoryPayload, CreateBrandPayload, UpdateBrandPayload, CreateUnitOfMeasurePayload, UpdateUnitOfMeasurePayload } from '../types';

export interface CategoryListParams {
  limit: number;
  offset: number;
  search?: string;
  sort?: string;
  dir?: string;
}

export interface CategoryListResponse {
  data: MasterCategory[];
  total: number;
}

export async function getCategories(params: CategoryListParams): Promise<CategoryListResponse> {
  const urlParams = new URLSearchParams({
    limit: params.limit.toString(),
    offset: params.offset.toString(),
  });
  if (params.search) urlParams.append('search', params.search);
  if (params.sort) urlParams.append('sort', params.sort);
  if (params.dir) urlParams.append('dir', params.dir);

  const res = await apiFetch(`/api/categories/manage?${urlParams.toString()}`);
  if (res.ok) {
    const data = await res.json();
    return { data: data.data || [], total: data.total || 0 };
  }
  return { data: [], total: 0 };
}

export async function createCategory(payload: CreateCategoryPayload): Promise<boolean> {
  const r = await apiFetch('/api/categories', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
  return r.ok;
}

export async function updateCategory(id: number, payload: UpdateCategoryPayload): Promise<boolean> {
  const r = await apiFetch(`/api/categories/${id}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
  });
  return r.ok;
}

export async function deleteCategory(id: number): Promise<boolean> {
  const r = await apiFetch(`/api/categories/${id}`, { method: 'DELETE' });
  return r.ok;
}

// Brands
export interface BrandListResponse {
  data: MasterBrand[];
  total: number;
}

export interface BrandListParams {
  limit: number;
  offset: number;
  search?: string;
}

export async function getBrands(params: BrandListParams): Promise<BrandListResponse> {
  const urlParams = new URLSearchParams({
    limit: params.limit.toString(),
    offset: params.offset.toString(),
  });
  if (params.search) urlParams.append('search', params.search);

  const r = await apiFetch(`/api/brands?${urlParams.toString()}`);
  if (r.ok) {
    const data = await r.json();
    return { data: data.data || [], total: data.total || 0 };
  }
  return { data: [], total: 0 };
}

export async function createBrand(payload: CreateBrandPayload): Promise<boolean> {
  const r = await apiFetch('/api/brands', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
  return r.ok;
}

export async function updateBrand(id: number, payload: UpdateBrandPayload): Promise<boolean> {
  const r = await apiFetch(`/api/brands/${id}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
  });
  return r.ok;
}

export async function deleteBrand(id: number): Promise<boolean> {
  const r = await apiFetch(`/api/brands/${id}`, { method: 'DELETE' });
  return r.ok;
}

// Units of Measure
export interface UomListResponse {
  data: MasterUnitOfMeasure[];
  total: number;
}

export interface UomListParams {
  limit: number;
  offset: number;
  search?: string;
}

export async function getUnitsOfMeasure(params: UomListParams): Promise<UomListResponse> {
  const urlParams = new URLSearchParams({
    limit: params.limit.toString(),
    offset: params.offset.toString(),
  });
  if (params.search) urlParams.append('search', params.search);

  const r = await apiFetch(`/api/units-of-measure?${urlParams.toString()}`);
  if (r.ok) {
    const data = await r.json();
    return { data: data.data || [], total: data.total || 0 };
  }
  return { data: [], total: 0 };
}

export async function createUnitOfMeasure(payload: CreateUnitOfMeasurePayload): Promise<boolean> {
  const r = await apiFetch('/api/units-of-measure', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
  return r.ok;
}

export async function updateUnitOfMeasure(id: number, payload: UpdateUnitOfMeasurePayload): Promise<boolean> {
  const r = await apiFetch(`/api/units-of-measure/${id}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
  });
  return r.ok;
}

export async function deleteUnitOfMeasure(id: number): Promise<boolean> {
  const r = await apiFetch(`/api/units-of-measure/${id}`, { method: 'DELETE' });
  return r.ok;
}

// Export & Import
function getToken(): string {
  return sessionStorage.getItem('access_token') || '';
}

function downloadExport(url: string, filename: string) {
  const token = getToken();
  window.open(`${url}&token=${token}`, '_blank');
}


export async function exportBrands(format: 'csv' | 'xlsx'): Promise<void> {
  downloadExport(`/api/brands/export?format=${format}`, `brands-${format}`);
}

export async function importBrands(file: File): Promise<any> {
  const formData = new FormData();
  formData.append('file', file);
  const r = await apiFetch('/api/brands/import', {
    method: 'POST',
    body: formData,
  });
  return r.ok ? r.json() : r.json().then(e => Promise.reject(e));
}

