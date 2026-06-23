import { apiFetch } from '$shared/api/http-client';
import type { Role, Permission, CreateRolePayload, UpdateRolePayload } from '../types';

export interface RolesResponse {
  data: Role[];
}

export interface PermissionsResponse {
  data: Permission[];
}

export async function getRoles(): Promise<Role[]> {
  const res = await apiFetch('/api/admin/roles');
  if (res.ok) {
    const data: RolesResponse = await res.json();
    return data.data || [];
  }
  return [];
}

export async function getPermissions(): Promise<Permission[]> {
  const res = await apiFetch('/api/admin/permissions');
  if (res.ok) {
    const data: PermissionsResponse = await res.json();
    return data.data || [];
  }
  return [];
}

export async function createRole(data: CreateRolePayload): Promise<{ id: number } | null> {
  const r = await apiFetch('/api/admin/roles', {
    method: 'POST',
    body: JSON.stringify(data),
  });
  if (r.ok) {
    const newRole = await r.json();
    return newRole.data;
  }
  return null;
}

export async function updateRole(id: number, data: UpdateRolePayload): Promise<boolean> {
  const r = await apiFetch(`/api/admin/roles/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
  return r.ok;
}

export async function updateRolePermissions(id: number, permissionIds: number[]): Promise<boolean> {
  const r = await apiFetch(`/api/admin/roles/${id}/permissions`, {
    method: 'PUT',
    body: JSON.stringify({ permission_ids: permissionIds }),
  });
  return r.ok;
}

export async function deleteRole(id: number): Promise<boolean> {
  const r = await apiFetch(`/api/admin/roles/${id}`, { method: 'DELETE' });
  return r.ok;
}
