export function getDefaultRoute(user: { role?: string | { name: string } } | null | undefined): string {
  if (!user) return '/';
  const role = typeof user.role === 'string' ? user.role : user.role?.name ?? '';
  if (role === 'cashier') return '/pos';
  if (role === 'staff') return '/inventory/products';
  return '/';
}
