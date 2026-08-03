import apiClient from '$shared/api/http-client';
import type { StorageLocation, StorageLocationFilters } from '../types';

export async function getStorageLocations(filters: StorageLocationFilters): Promise<{ data: StorageLocation[]; total: number }> {
  const params: Record<string, string | number | undefined> = {
    limit: filters.limit ?? 20,
    offset: filters.offset ?? 0,
    search: filters.search || undefined,
  };
  if (filters.is_active !== undefined) {
    params.is_active = filters.is_active ? 'true' : 'false';
  }
  const r = await apiClient.get('/storage-locations', { params });
  return { data: r.data.data || [], total: r.data.total || 0 };
}

export async function getStorageLocation(id: number): Promise<StorageLocation | null> {
  const r = await apiClient.get(`/storage-locations/${id}`);
  return r.data.data || null;
}

export async function createStorageLocation(data: { code: string; name: string; warehouse_id?: number | null; store_id?: number | null; notes?: string }): Promise<void> {
  await apiClient.post('/storage-locations', data);
}

export async function updateStorageLocation(id: number, data: { code?: string; name?: string; warehouse_id?: number | null; store_id?: number | null; notes?: string; is_active?: boolean }): Promise<void> {
  await apiClient.put(`/storage-locations/${id}`, data);
}

export async function deleteStorageLocation(id: number): Promise<void> {
  await apiClient.delete(`/storage-locations/${id}`);
}

export async function bulkUpdateStorageLocations(ids: number[], isActive: boolean): Promise<number> {
  const r = await apiClient.put('/storage-locations/bulk', { ids, is_active: isActive });
  return r.data.updated || 0;
}

export async function bulkDeleteStorageLocations(ids: number[]): Promise<number> {
  const r = await apiClient.delete('/storage-locations/bulk', { data: { ids } });
  return r.data.deleted || 0;
}
