import { BarChart3, ClipboardList, Clock, LayoutDashboard, Package, Percent, Settings, Shield, ShoppingCart, Store, Tag, Truck, UserPlus, Users, Warehouse } from 'lucide-svelte';
import { labels } from '$shared/i18n';

export function permissionGroupKey(code: string): string {
  let key = code.split('.')[0];
  if (key === 'role') key = 'user';
  return key;
}

const GROUP_META: Record<string, { label: string; icon: typeof Shield }> = {
  user: { label: `${labels.user} & ${labels.role}`, icon: Users },
  product: { label: labels.product, icon: Package },
  category: { label: labels.category, icon: Tag },
  sale: { label: 'Sales', icon: ShoppingCart },
  shift: { label: labels.shifts, icon: Clock },
  inventory: { label: labels.inventory, icon: Warehouse },
  customer: { label: labels.customer, icon: UserPlus },
  customer_group: { label: labels.customerGroup, icon: Users },
  pricing: { label: labels.pricingRules, icon: Percent },
  purchase_order: { label: labels.purchaseOrder, icon: Truck },
  stock_opname: { label: labels.stockOpname, icon: ClipboardList },
  storage_location: { label: labels.storageLocation, icon: Warehouse },
  store: { label: labels.store, icon: Store },
  report: { label: labels.reports, icon: BarChart3 },
  dashboard: { label: labels.dashboard, icon: LayoutDashboard },
  pos: { label: 'POS', icon: Store },
  audit: { label: labels.system, icon: Settings },
};

const GROUP_ORDER = [
  'user', 'product', 'category', 'sale', 'shift', 'inventory',
  'customer', 'customer_group', 'pricing', 'purchase_order',
  'stock_opname', 'storage_location', 'store', 'report', 'dashboard', 'pos', 'audit',
];

export function groupPermissions(perms: Array<{ code: string }>): Array<{ key: string; label: string; icon: typeof Shield; permissions: Array<{ code: string }> }> {
  const groups: Record<string, Array<{ code: string }>> = {};
  for (const p of perms) {
    const key = permissionGroupKey(p.code);
    if (!groups[key]) groups[key] = [];
    groups[key].push(p);
  }
  const known = GROUP_ORDER.filter(k => groups[k]?.length).map(k => ({
    key: k,
    label: GROUP_META[k]?.label ?? k,
    icon: GROUP_META[k]?.icon ?? Shield,
    permissions: groups[k],
  }));
  const unknown = Object.keys(groups)
    .filter(k => !GROUP_ORDER.includes(k))
    .sort()
    .map(k => ({
      key: k,
      label: k,
      icon: Shield,
      permissions: groups[k],
    }));
  return [...known, ...unknown];
}
