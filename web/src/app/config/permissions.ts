/**
 * Route Permissions Configuration
 * Maps routes to required permissions.
 * This is the single source of truth for route access control.
 *
 * Permission strings match the backend API (cmd/server/main.go).
 */

export const routePermissions: Record<string, string[]> = {
	'/pos': ['sale:create'],
	'/inventory': ['product:read'],
	'/inventory/products': ['product:read'],
	'/reports': ['report:read'],
	'/transactions': ['sale:read'],
	'/customers': ['customer:read'],
	'/categories': ['category:read'],
	'/brands': ['product:read'],
	'/units-of-measure': ['product:read'],
	'/admin': ['user:read'],
	'/admin/users': ['user:read'],
	'/admin/roles': ['role:read'],
	'/admin/audit-logs': ['audit:read'],
	'/admin/categories': ['category:read'],
	'/admin/brands': ['product:read'],
	'/admin/units-of-measure': ['product:read'],
	'/pricing-rules': ['pricing:read'],
	'/customer-groups': ['customer_group:read'],
	'/suppliers': ['pricing:read'],
	'/shifts': ['shift:read'],
};
