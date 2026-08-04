export function getDefaultRoute(user: { role?: string | { name: string } } | null | undefined): string {
  if (!user) return '/';
  // @display-only — default-route hanyalah navigasi awal UX; otorisasi akses tetap via permission.
  const role = typeof user.role === 'string' ? user.role : user.role?.name ?? '';
  if (role === 'cashier') return '/shifts';
  if (role === 'staff') return '/inventory/products';
  return '/';
}
