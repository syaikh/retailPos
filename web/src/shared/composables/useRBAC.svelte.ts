import { useAuthStore } from '$modules/auth';
import type { User } from '$modules/auth';

const ADMIN_ROLES = ['superadmin', 'admin'] as const;
const MANAGER_ROLES = ['superadmin', 'admin', 'manager'] as const;

function resolveRoleName(user: User | null): string {
  if (!user) return '';
  const role = user.role;
  if (typeof role === 'string') return role;
  if (role?.name) return role.name;
  return '';
}

type Role = typeof ADMIN_ROLES[number];

export function useRBAC() {
  const store = useAuthStore();

  const userRole = $derived(resolveRoleName(store.user));
  const isAdmin = $derived(ADMIN_ROLES.includes(userRole as Role));
  const isSuperAdmin = $derived(userRole === 'superadmin');
  const isManager = $derived(MANAGER_ROLES.includes(userRole as Role));
  const isCashier = $derived(userRole === 'cashier');

  return {
    get userRole() { return userRole; },
    get isAdmin() { return isAdmin; },
    get isSuperAdmin() { return isSuperAdmin; },
    get isManager() { return isManager; },
    get isCashier() { return isCashier; },
    get canCreate() { return isAdmin; },
    get canEdit() { return isAdmin; },
    get canDelete() { return isSuperAdmin; },
    get canView() { return userRole !== '' && !isCashier; },
    get canEditSuperadmin() { return isSuperAdmin; },
    get canAssignManager() { return isAdmin; },
  };
}
