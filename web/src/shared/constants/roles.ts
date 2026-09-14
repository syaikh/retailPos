/**
 * Role Registry — role names for presentation/ownership purposes only.
 *
 * Sprint 0 policy (rbac-sprint0-audit.md §8 K3):
 * - Roles may be used for DISPLAY (labels, badges) and OWNERSHIP (data scoping).
 * - Roles must NEVER be used for authorization decisions.
 * - Authorization must always go through useRBAC().can()/canAny()/canAll()
 *   with permission codes from `permissions.ts`.
 *
 * Role names follow migration 044 (retail job titles): `manager` (was `admin`),
 * `supervisor` (was `manager`), `inventory_staff` (was `staff`), new `finance`.
 */

export const Roles = {
  superadmin: "superadmin",
  manager: "manager",
  supervisor: "supervisor",
  cashier: "cashier",
  inventory_staff: "inventory_staff",
  finance: "finance",
} as const;

export type RoleName = (typeof Roles)[keyof typeof Roles];

export const ALL_ROLES: readonly string[] = Object.values(Roles);
