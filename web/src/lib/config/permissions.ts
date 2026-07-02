/**
 * Route Permissions Configuration
 * Maps routes to required permissions.
 * This is the single source of truth for route access control.
 */

export const routePermissions: Record<string, string[]> = {
	'/': ['dashboard:read'],
	'/pos': ['pos:access'],
	'/inventory': ['inventory:read'],
	'/inventory/groups': ['inventory:group:read'],
	'/reports': ['reports:read'],
	'/admin/users': ['users:read'],
	'/admin/roles': ['users:roles:manage']
};
