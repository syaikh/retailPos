import { apiFetch } from '$shared/api/http-client';
import type { MasterCategory, CreateCategoryPayload, UpdateCategoryPayload } from '../types';

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
