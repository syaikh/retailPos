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
  if (filters.customer_group_id !== undefined) {
    params.customer_group_id = filters.customer_group_id;
  }
  const r = await apiClient.get('/customers', { params });
  return { data: r.data.data || [], total: r.data.total || 0 };
}

export async function createCustomer(data: { name: string; phone: string; email: string; address?: string; note?: string; customer_group_id?: number | null }): Promise<void> {
  await apiClient.post('/customers', data);
}

export async function updateCustomer(id: number, data: { name?: string; phone?: string; email?: string; address?: string; note?: string; is_active?: boolean; customer_group_id?: number | null }): Promise<void> {
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

