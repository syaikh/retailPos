import apiClient from '$shared/api/http-client';
import type { User, CreateUserPayload, UpdateUserPayload } from '../types';

export interface UserListParams {
  limit: number;
  offset: number;
  search?: string;
  sort?: string;
  sort_dir?: string;
  role_id?: string;
  is_active?: string;
}

export interface UserListResponse {
  data: User[];
  total: number;
}

export async function getUsers(params: UserListParams): Promise<UserListResponse> {
  const res = await apiClient.get('/admin/users', { params });
  return { data: res.data?.data || [], total: res.data?.total || 0 };
}

export async function getRolesList() {
  const res = await apiClient.get('/admin/roles');
  return res.data?.data || [];
}

export async function createUser(data: CreateUserPayload): Promise<void> {
  await apiClient({ url: '/admin/users', method: 'POST', data });
}

export async function updateUser(id: number, data: UpdateUserPayload): Promise<void> {
  await apiClient({ url: `/admin/users/${id}`, method: 'PUT', data });
}

export async function deleteUser(id: number): Promise<void> {
  await apiClient.delete(`/admin/users/${id}`);
}

export async function getSubordinates(id: number): Promise<User[]> {
  const res = await apiClient.get(`/admin/users/${id}/subordinates`);
  return res.data?.data || [];
}

export async function getManager(id: number): Promise<User | null> {
  const res = await apiClient.get(`/admin/users/${id}/manager`);
  return res.data?.data || null;
}

export async function getOrgChart(): Promise<User[]> {
  const res = await apiClient.get('/admin/users/org-chart');
  return res.data?.data || [];
}
