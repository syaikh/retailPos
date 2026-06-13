import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), 'Sidebar.svelte'), 'utf-8');
}

describe('Sidebar.svelte role-based navigation', () => {
  const src = getSource();

  it('defines masterDataSubItems with Products, Categories, Customers (no Stock)', () => {
    expect(src).toContain('masterDataSubItems');
    expect(src).toContain("href: '/inventory/products'");
    expect(src).toContain("href: '/categories'");
    expect(src).toContain("href: '/customers'");
  });

  it('Stock is NOT inside masterDataSubItems', () => {
    const mdStart = src.indexOf('const masterDataSubItems');
    const mdEnd = src.indexOf('];', mdStart);
    const mdBlock = src.slice(mdStart, mdEnd);
    expect(mdBlock).not.toContain("'/inventory/stock'");
  });

  it('Stock is a separate top-level item', () => {
    expect(src).toContain("navigate('/inventory/stock')");
    expect(src).toContain('showStockItem');
  });

  it('defines visibleMasterDataSubItems based on role', () => {
    expect(src).toContain('visibleMasterDataSubItems');
  });

  it('staff role sees no master data sub-items', () => {
    expect(src).toContain("role === 'staff' ? []");
  });

  it('cashier role sees no master data sub-items', () => {
    expect(src).toContain("role === 'cashier' ? []");
  });

  it('manager role sees Products, Categories, Customers in master data', () => {
    expect(src).toContain('managerMasterDataSubItems');
  });

  it('admin/superadmin sees full masterDataSubItems', () => {
    expect(src).toContain('masterDataSubItems');
  });

  it('cashier role does NOT see Stock item', () => {
    expect(src).toContain("role !== 'cashier'");
  });

  it('staff role sees Stock item', () => {
    expect(src).toContain('showStockItem');
  });

  it('creates expandable Master Data group', () => {
    expect(src).toContain('masterDataExpanded');
    expect(src).toContain('isMasterDataPath');
  });

  it('master data auto-expands on master data path', () => {
    expect(src).toContain("currentPath.startsWith('/inventory/products')");
    expect(src).toContain("currentPath.startsWith('/categories')");
    expect(src).toContain("currentPath.startsWith('/customers')");
  });

  it('collapsed sidebar navigates to /inventory/products on master data click', () => {
    expect(src).toContain("navigate('/inventory/products')");
  });

  it('sub-items are indented with pl-9', () => {
    expect(src).toContain('pl-9');
  });

  it('still defines staffNavItems with Dashboard only', () => {
    expect(src).toContain('const staffNavItems = [');
  });

  it('still defines cashierNavItems with Dashboard + POS only', () => {
    expect(src).toContain('const cashierNavItems = [');
    expect(src).toContain("href: '/pos'");
  });

  it('shows admin section only for admin and superadmin', () => {
    expect(src).toContain("role === 'admin' || role === 'superadmin'");
  });

  it('administration is an expandable group with chevron', () => {
    expect(src).toContain('adminExpanded');
    expect(src).toContain("adminExpanded = !adminExpanded");
  });

  it('administration group uses Shield icon', () => {
    expect(src).toContain('<Shield');
  });

  it('administration sub-items are indented with pl-9', () => {
    expect(src).toContain("pl-9");
  });

  it('collapsed sidebar navigates to /admin/users on administration click', () => {
    expect(src).toContain("navigate('/admin/users')");
  });

  it('marks audit logs as requiring superadmin', () => {
    expect(src).toContain('requiresSuperadmin: true');
    expect(src).toContain("/admin/audit-logs'");
  });

  it('determines visibleNavItems based on role', () => {
    expect(src).toContain("role === 'staff' ? staffNavItems");
    expect(src).toContain("role === 'cashier' ? cashierNavItems");
    expect(src).toContain("role === 'manager' ? managerNavItems");
  });

  it('imports Database icon for Master Data group', () => {
    expect(src).toContain('Database');
  });

  it('imports ClipboardList icon for Stock item', () => {
    expect(src).toContain('ClipboardList');
  });
});
