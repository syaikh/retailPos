import { useAuthStore } from '$modules/auth';

const ADMIN_ROLES = ['superadmin', 'admin'] as const;
const MANAGER_ROLES = ['superadmin', 'admin', 'manager'] as const;

function resolveRoleName(user: any): string {
  if (!user) return '';
  const role = user.role;
  if (typeof role === 'string') return role;
  if (role?.name) return role.name;
  return '';
}

export function useRBAC() {
  const store = useAuthStore();

  const userRole = $derived(resolveRoleName(store.user));
  const isAdmin = $derived(ADMIN_ROLES.includes(userRole as any));
  const isSuperAdmin = $derived(userRole === 'superadmin');
  const isManager = $derived(MANAGER_ROLES.includes(userRole as any));
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
  };
}
