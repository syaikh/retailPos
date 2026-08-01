import { apiFetch } from '$shared/api/http-client';
import type { Store, CreateStorePayload, UpdateStorePayload, StoreListParams, StoreListResponse } from '../types';

export async function getStores(params: StoreListParams): Promise<StoreListResponse> {
  const urlParams = new URLSearchParams({
    limit: params.limit.toString(),
    offset: params.offset.toString(),
  });
  if (params.search) urlParams.append('search', params.search);
  if (params.is_active !== undefined) urlParams.append('is_active', params.is_active.toString());

  const res = await apiFetch(`/api/stores?${urlParams.toString()}`);
  if (res.ok) {
    const data = await res.json();
    return { data: data.data || [], total: data.total || 0 };
  }
  return { data: [], total: 0 };
}

export async function getActiveStores(): Promise<Store[]> {
  const r = await apiFetch('/api/stores/active');
  if (r.ok) {
    const data = await r.json();
    return data.data || [];
  }
  return [];
}

export async function getStore(id: number): Promise<Store | null> {
  const r = await apiFetch(`/api/stores/${id}`);
  if (r.ok) {
    const data = await r.json();
    return data.data || null;
  }
  return null;
}

export async function createStore(payload: CreateStorePayload): Promise<boolean> {
  const r = await apiFetch('/api/stores', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
  return r.ok;
}

export async function updateStore(id: number, payload: UpdateStorePayload): Promise<boolean> {
  const r = await apiFetch(`/api/stores/${id}`, {
    method: 'PUT',
    body: JSON.stringify(payload),
  });
  return r.ok;
}

export async function deleteStore(id: number): Promise<boolean> {
  const r = await apiFetch(`/api/stores/${id}`, { method: 'DELETE' });
  return r.ok;
}
