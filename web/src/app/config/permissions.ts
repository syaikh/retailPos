/**
 * Route Permissions Configuration
 * Maps routes to required permissions.
 * This is the single source of truth for route access control.
 *
 * Permission strings match the backend API (cmd/server/main.go).
 */

export const routePermissions: Record<string, string[]> = {
	'/pos': ['sale.create'],
	'/inventory': ['product.view'],
	'/inventory/products': ['product.view'],
	'/reports': ['report.view'],
	'/transactions': ['sale.view'],
	'/customers': ['customer.view'],
	'/categories': ['category.view'],
	'/brands': ['product.view'],
	'/units-of-measure': ['product.view'],
	'/admin': ['user.view'],
	'/admin/users': ['user.view'],
	'/admin/roles': ['role.view'],
	'/admin/audit-logs': ['audit.view'],
	'/stores': ['store.view'],
	'/admin/categories': ['category.view'],
	'/admin/brands': ['product.view'],
	'/admin/units-of-measure': ['product.view'],
	'/pricing-rules': ['pricing.view'],
	'/customer-groups': ['customer_group.view'],
	'/suppliers': ['pricing.view'],
	'/shifts': ['shift.view'],
	'/stock-opnames': ['stock_opname.view'],
};
