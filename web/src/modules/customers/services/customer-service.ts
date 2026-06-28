import apiClient from '$shared/api/http-client';
import type { Customer, CustomerFilters } from '../types';

export async function getCustomers(filters: CustomerFilters): Promise<{ data: Customer[]; total: number }> {
  const params: Record<string, string | number | undefined> = {
    limit: filters.limit ?? 20,
    offset: filters.offset ?? 0,
    search: filters.search || undefined,
  };
  if (filters.isActive !== undefined) {
    params.isActive = filters.isActive;
  }
  const r = await apiClient.get('/customers', { params });
  return { data: r.data.data || [], total: r.data.total || 0 };
}

export async function createCustomer(data: { name: string; phone: string; email: string; note?: string }): Promise<void> {
  await apiClient.post('/customers', data);
}

export async function updateCustomer(id: number, data: { name: string; phone?: string; email?: string; note?: string; is_active?: boolean }): Promise<void> {
  await apiClient.put(`/customers/${id}`, data);
}

export async function deleteCustomer(id: number): Promise<void> {
  await apiClient.delete(`/customers/${id}`);
}

export async function bulkUpdateStatus(ids: number[], isActive: boolean): Promise<void> {
  await apiClient.post('/customers/bulk/status', { ids, is_active: isActive });
}

export async function bulkDelete(ids: number[]): Promise<void> {
  await apiClient.post('/customers/bulk/delete', { ids });
}

export async function exportCustomers(format: 'csv' | 'xlsx'): Promise<void> {
  const token = sessionStorage.getItem('access_token');
  window.open(`/api/customers/export?format=${format}&token=${token}`, '_blank');
}

export async function importCustomers(file: File): Promise<any> {
  const formData = new FormData();
  formData.append('file', file);
  const r = await apiClient.post('/customers/import', formData);
  return r.data;
}
