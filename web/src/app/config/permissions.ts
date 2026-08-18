/**
 * Route Permissions Configuration
 * Maps routes to required permissions.
 * This is the single source of truth for route access control.
 *
 * Permission codes reference the shared registry (src/shared/constants/permissions.ts),
 * which is synced to the backend permissions table (see docs/audits/permission-matrix-final.md).
 */

import { Permissions } from '$shared/constants/permissions';

export const routePermissions: Record<string, string[]> = {
	'/': [Permissions.dashboard.view],
	'/pos': [Permissions.sale.create],
	'/inventory': [Permissions.product.view],
	'/inventory/products': [Permissions.product.view],
	'/reports': [Permissions.report.view],
	'/transactions': [Permissions.sale.view],
	'/customers': [Permissions.customer.view],
	'/categories': [Permissions.category.view],
	'/brands': [Permissions.product.view],
	'/units-of-measure': [Permissions.product.view],
	'/admin': [Permissions.user.view],
	'/admin/users': [Permissions.user.view],
	'/admin/roles': [Permissions.role.view],
	'/admin/audit-logs': [Permissions.audit.view],
	'/stores': [Permissions.store.view],
	'/admin/categories': [Permissions.category.view],
	'/admin/brands': [Permissions.product.view],
	'/admin/units-of-measure': [Permissions.product.view],
	'/pricing-rules': [Permissions.pricing.view],
	'/customer-groups': [Permissions.customerGroup.view],
	'/suppliers': [Permissions.pricing.view],
	'/purchase-orders': [Permissions.purchaseOrder.view],
	'/shifts': [Permissions.shift.view],
	'/stock-opnames': [Permissions.stockOpname.view],
	'/stock-opnames/adjustments': [Permissions.stockOpname.report],
	'/storage-locations': [Permissions.storageLocation.view],
	'/consignment': [Permissions.consignment.view],
	'/admin/settings': [Permissions.appSettings.view],
};
