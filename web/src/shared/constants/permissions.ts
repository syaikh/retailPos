/**
 * Permission Registry — single source of truth for all permission codes.
 *
 * Sync source: database (permissions table, 74 live codes) — see
 * docs/audits/permission-matrix-final.md (72, approved 2026-08-04) and
 * docs/audits/permission-additions-sprint1.md (+product.history.view,
 * +product.cost.view).
 *
 * Sprint 0 rules (permission-matrix-final.md §8 Scope Freeze):
 * - Do NOT add/remove/rename permission codes in this file.
 * - Do NOT use raw string literals for permissions in components — reference these constants.
 * - Changes only via backlog (Sprint 1+), not mid-sprint.
 */

export const Permissions = {
  dashboard: {
    view: 'dashboard.view',
  },
  user: {
    view: 'user.view',
    create: 'user.create',
    update: 'user.update',
    delete: 'user.delete',
  },
  role: {
    view: 'role.view',
    create: 'role.create',
    update: 'role.update',
    delete: 'role.delete',
  },
  audit: {
    view: 'audit.view',
  },
  report: {
    view: 'report.view',
  },
  product: {
    view: 'product.view',
    create: 'product.create',
    update: 'product.update',
    delete: 'product.delete',
    export: 'product.export',
    import: 'product.import',
    historyView: 'product.history.view',
    costView: 'product.cost.view',
  },
  category: {
    view: 'category.view',
    create: 'category.create',
    update: 'category.update',
    delete: 'category.delete',
    export: 'category.export',
    import: 'category.import',
  },
  sale: {
    view: 'sale.view',
    create: 'sale.create',
    park: 'sale.park',
  },
  shift: {
    view: 'shift.view',
    create: 'shift.create',
    review: 'shift.review',
    audit: 'shift.audit',
  },
  customer: {
    view: 'customer.view',
    create: 'customer.create',
    update: 'customer.update',
    delete: 'customer.delete',
    export: 'customer.export',
    import: 'customer.import',
  },
  pricing: {
    view: 'pricing.view',
    create: 'pricing.create',
    update: 'pricing.update',
    delete: 'pricing.delete',
  },
  inventory: {
    adjust: 'inventory.adjust',
  },
  store: {
    view: 'store.view',
    create: 'store.create',
    update: 'store.update',
    delete: 'store.delete',
  },
  customerGroup: {
    view: 'customer_group.view',
    create: 'customer_group.create',
    update: 'customer_group.update',
    delete: 'customer_group.delete',
  },
  purchaseOrder: {
    view: 'purchase_order.view',
    create: 'purchase_order.create',
    update: 'purchase_order.update',
    delete: 'purchase_order.delete',
    confirm: 'purchase_order.confirm',
    receive: 'purchase_order.receive',
    cancel: 'purchase_order.cancel',
  },
  stockOpname: {
    view: 'stock_opname.view',
    create: 'stock_opname.create',
    assign: 'stock_opname.assign',
    count: 'stock_opname.count',
    submit: 'stock_opname.submit',
    recount: 'stock_opname.recount',
    cancel: 'stock_opname.cancel',
    export: 'stock_opname.export',
    verify: 'stock_opname.verify',
    post: 'stock_opname.post',
    close: 'stock_opname.close',
    report: 'stock_opname.report',
  },
  storageLocation: {
    view: 'storage_location.view',
    create: 'storage_location.create',
    update: 'storage_location.update',
    delete: 'storage_location.delete',
  },
  consignment: {
    view: 'consignment.view',
    create: 'consignment.create',
    update: 'consignment.update',
    settle: 'consignment.settle',
    pay: 'consignment.pay',
  },
  appSettings: {
    view: 'app_settings.view',
    update: 'app_settings.update',
  },
} as const;

export type PermissionCode = (typeof Permissions)[keyof typeof Permissions][keyof (typeof Permissions)[keyof typeof Permissions]];

/**
 * Flat list of every permission code. Used by the RBAC composable and tests.
 * Derived from the registry above so it can never drift from it.
 */
export const ALL_PERMISSIONS: readonly string[] = Object.values(Permissions).flatMap(
  (group) => Object.values(group),
);

/**
 * Permission groups used to gate whole sections of the sidebar.
 */
export const MASTER_DATA_MANAGE_PERMISSIONS = [
  Permissions.product.create,
  Permissions.category.create,
  Permissions.customer.create,
  Permissions.pricing.create,
  Permissions.store.create,
  Permissions.customerGroup.create,
] as const;
