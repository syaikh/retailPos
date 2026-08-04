/**
 * Role Registry — role names for presentation/ownership purposes only.
 *
 * Sprint 0 policy (rbac-sprint0-audit.md §8 K3):
 * - Roles may be used for DISPLAY (labels, badges) and OWNERSHIP (data scoping).
 * - Roles must NEVER be used for authorization decisions.
 * - Authorization must always go through useRBAC().can()/canAny()/canAll()
 *   with permission codes from `permissions.ts`.
 */

export const Roles = {
  superadmin: 'superadmin',
  admin: 'admin',
  manager: 'manager',
  cashier: 'cashier',
  staff: 'staff',
} as const;

export type RoleName = (typeof Roles)[keyof typeof Roles];

export const ALL_ROLES: readonly string[] = Object.values(Roles);
