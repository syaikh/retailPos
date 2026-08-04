import { useAuthStore } from '$modules/auth';
import type { User } from '$modules/auth';
import { Roles } from '$shared/constants/roles';

function resolveRoleName(user: User | null): string {
  if (!user) return '';
  const role = user.role;
  if (typeof role === 'string') return role;
  if (role?.name) return role.name;
  return '';
}

export function useRBAC() {
  const store = useAuthStore();

  const userPerms = $derived(store.user?.permissions ?? []);
  const userRole = $derived(resolveRoleName(store.user));

  function can(permission: string): boolean {
    return userPerms.includes(permission);
  }
  function canAny(permissions: string[]): boolean {
    return permissions.some((p) => userPerms.includes(p));
  }
  function canAll(permissions: string[]): boolean {
    return permissions.every((p) => userPerms.includes(p));
  }

  // @display-only — untuk UI (label/badge), bukan authorization.
  const roleDisplayName = $derived(userRole ? userRole.charAt(0).toUpperCase() + userRole.slice(1) : '');

  // @ownership-only — hanya untuk data-scope (filter own data), bukan authorization.
  const isCashier = $derived(userRole === Roles.cashier);

  return {
    can,
    canAny,
    canAll,
    get userRole() { return userRole; },
    get roleDisplayName() { return roleDisplayName; },
    get isCashier() { return isCashier; },
  };
}
