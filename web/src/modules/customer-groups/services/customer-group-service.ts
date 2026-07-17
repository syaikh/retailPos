import apiClient from '$shared/api/http-client';
import type { CustomerGroup, CustomerGroupFilters } from '../types';

export async function getCustomerGroups(filters: CustomerGroupFilters): Promise<{ data: CustomerGroup[]; total: number }> {
  const params: Record<string, string | number | undefined> = {
    limit: filters.limit ?? 20,
    offset: filters.offset ?? 0,
    search: filters.search || undefined,
  };
  if (filters.is_active !== undefined) {
    params.is_active = filters.is_active ? 'true' : 'false';
  }
  const r = await apiClient.get('/customer-groups', { params });
  return { data: r.data.data || [], total: r.data.total || 0 };
}

export async function getCustomerGroup(id: number): Promise<CustomerGroup | null> {
  const r = await apiClient.get(`/customer-groups/${id}`);
  return r.data.data || null;
}

export async function createCustomerGroup(data: { name: string; description?: string }): Promise<void> {
  await apiClient.post('/customer-groups', data);
}

export async function updateCustomerGroup(id: number, data: { name?: string; description?: string; is_active?: boolean }): Promise<void> {
  await apiClient.put(`/customer-groups/${id}`, data);
}

export async function deleteCustomerGroup(id: number): Promise<void> {
  await apiClient.delete(`/customer-groups/${id}`);
}
