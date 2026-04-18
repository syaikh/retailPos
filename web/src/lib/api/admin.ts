import client from './client';

export interface Role {
	id: number;
	name: string;
	description: string;
	is_system: boolean;
	permissions: string[];
}

export interface Permission {
	id: number;
	code: string;
	description: string;
	category: string;
}

export interface User {
	id: number;
	username: string;
	role_id: number;
	role_name: string;
	is_system_role: boolean;
}

export async function getRoles(): Promise<Role[]> {
	const { data } = await client.get('/roles');
	return data.roles;
}

export async function getPermissions(): Promise<Permission[]> {
	const { data } = await client.get('/permissions');
	return data.permissions;
}

export async function createRole(name: string, description: string, permissionIds: number[]) {
	const { data } = await client.post('/roles', {
		name,
		description,
		permission_ids: permissionIds
	});
	return data;
}

export async function updateRolePermissions(roleId: number, permissionIds: number[]) {
	const { data } = await client.put(`/roles/${roleId}/permissions`, {
		permission_ids: permissionIds
	});
	return data;
}

export async function deleteRole(roleId: number) {
	const { data } = await client.delete(`/roles/${roleId}`);
	return data;
}

export async function getUsers(): Promise<User[]> {
	const { data } = await client.get('/users');
	return data.users;
}

export async function updateUserRole(userId: number, roleId: number) {
	const { data } = await client.put(`/users/${userId}/role`, {
		role_id: roleId
	});
	return data;
}