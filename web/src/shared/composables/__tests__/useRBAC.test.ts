import { describe, it, expect, beforeEach } from 'vitest';
import { useRBAC } from '../useRBAC.svelte';
import { useAuthStore } from '$modules/auth';
import { ALL_PERMISSIONS } from '$shared/constants/permissions';
import { Roles } from '$shared/constants/roles';

/**
 * Role × permission matrix — source of truth: docs/audits/permission-matrix-final.md
 * (72 × 5, target state setelah migration 023: staff = 5).
 */
const MATRIX: Record<string, readonly string[]> = {
  [Roles.superadmin]: ALL_PERMISSIONS,
  [Roles.admin]: ALL_PERMISSIONS.filter(
    (p) => !['user.delete', 'role.update', 'role.delete', 'audit.view', 'purchase_order.delete'].includes(p),
  ),
  [Roles.manager]: [
    'dashboard.view', 'report.view',
    'product.view', 'product.update', 'category.view', 'category.create',
    'sale.view', 'shift.view', 'shift.create', 'shift.review', 'shift.audit',
    'customer.view', 'customer.create', 'customer.update',
    'pricing.view', 'pricing.create', 'pricing.update',
    'inventory.adjust', 'customer_group.view',
    'purchase_order.view', 'purchase_order.create', 'purchase_order.update', 'purchase_order.confirm', 'purchase_order.receive', 'purchase_order.cancel',
    'stock_opname.view', 'stock_opname.create', 'stock_opname.assign', 'stock_opname.recount', 'stock_opname.cancel', 'stock_opname.export', 'stock_opname.verify', 'stock_opname.post', 'stock_opname.close', 'stock_opname.report',
    'storage_location.view',
  ],
  [Roles.cashier]: [
    'product.view',
    'sale.view', 'sale.create', 'sale.park',
    'shift.view', 'shift.create',
    'customer.view',
    'stock_opname.view', 'stock_opname.count', 'stock_opname.submit',
    'storage_location.view',
  ],
  [Roles.staff]: [
    'product.view',
    'stock_opname.view', 'stock_opname.count', 'stock_opname.submit',
    'storage_location.view',
  ],
};

function setRole(role: string, permissions: readonly string[], roleAsObject = false): void {
  const authStore = useAuthStore();
  authStore.setUser({
    id: 1,
    username: role,
    email: `${role}@test.com`,
    role: roleAsObject ? { id: 1, name: role } : role,
    permissions: [...permissions],
  });
}

describe('useRBAC', () => {
  beforeEach(() => {
    useAuthStore().clearUser();
  });

  it('exports a function and returns the expected API shape', () => {
    expect(typeof useRBAC).toBe('function');
    const rbac = useRBAC();
    expect(typeof rbac.can).toBe('function');
    expect(typeof rbac.canAny).toBe('function');
    expect(typeof rbac.canAll).toBe('function');
    expect('userRole' in rbac).toBe(true);
    expect('roleDisplayName' in rbac).toBe(true);
    expect('isCashier' in rbac).toBe(true);
  });

  it('matches the 72×5 permission matrix for every role', () => {
    for (const [role, granted] of Object.entries(MATRIX)) {
      setRole(role, granted);
      const rbac = useRBAC();
      for (const permission of ALL_PERMISSIONS) {
        expect(rbac.can(permission), `${role} can ${permission}`).toBe(granted.includes(permission));
      }
    }
  });

  it('can returns true only for granted permissions', () => {
    setRole(Roles.admin, ['product.create', 'product.update']);
    const rbac = useRBAC();
    expect(rbac.can('product.create')).toBe(true);
    expect(rbac.can('product.update')).toBe(true);
    expect(rbac.can('product.delete')).toBe(false);
  });

  it('canAny returns true when at least one permission is granted', () => {
    setRole(Roles.manager, ['store.view']);
    const rbac = useRBAC();
    expect(rbac.canAny(['store.view', 'store.create'])).toBe(true);
    expect(rbac.canAny(['store.create', 'store.delete'])).toBe(false);
  });

  it('canAll returns true only when every permission is granted', () => {
    setRole(Roles.admin, ['user.view', 'user.create', 'user.update']);
    const rbac = useRBAC();
    expect(rbac.canAll(['user.view', 'user.create'])).toBe(true);
    expect(rbac.canAll(['user.view', 'user.delete'])).toBe(false);
  });

  it('resolves userRole from a string role', () => {
    setRole(Roles.cashier, MATRIX[Roles.cashier]);
    const rbac = useRBAC();
    expect(rbac.userRole).toBe('cashier');
  });

  it('resolves userRole from an object role ({ id, name })', () => {
    setRole(Roles.manager, MATRIX[Roles.manager], true);
    const rbac = useRBAC();
    expect(rbac.userRole).toBe('manager');
  });

  it('roleDisplayName is capitalized for UI display', () => {
    setRole(Roles.manager, MATRIX[Roles.manager]);
    const rbac = useRBAC();
    expect(rbac.roleDisplayName).toBe('Manager');
  });

  it('isCashier is true only for the cashier role', () => {
    setRole(Roles.cashier, MATRIX[Roles.cashier]);
    expect(useRBAC().isCashier).toBe(true);
    setRole(Roles.admin, MATRIX[Roles.admin]);
    expect(useRBAC().isCashier).toBe(false);
    setRole(Roles.staff, MATRIX[Roles.staff]);
    expect(useRBAC().isCashier).toBe(false);
  });

  it('treats an unauthenticated user as having no permissions and no role', () => {
    const rbac = useRBAC();
    expect(rbac.userRole).toBe('');
    expect(rbac.roleDisplayName).toBe('');
    expect(rbac.can('dashboard.view')).toBe(false);
    expect(rbac.canAny(ALL_PERMISSIONS)).toBe(false);
    expect(rbac.canAll([])).toBe(true);
  });
});
