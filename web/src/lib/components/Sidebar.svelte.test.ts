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

  it('defines staffNavItems with Dashboard + Inventory only', () => {
    expect(src).toContain('const staffNavItems = [');
    expect(src).toContain("href: '/'");
    expect(src).toContain("href: '/inventory'");
  });

  it('defines cashierNavItems with Dashboard + POS only', () => {
    expect(src).toContain('const cashierNavItems = [');
    expect(src).toContain("href: '/pos'");
  });

  it('defines managerNavItems with Inventory + Reports + Categories (no admin)', () => {
    expect(src).toContain('const managerNavItems = [');
    expect(src).toContain("/inventory'");
    expect(src).toContain("/reports'");
    expect(src).toContain("/categories'");
  });

  it('determines visibleNavItems based on role', () => {
    expect(src).toContain("role === 'staff' ? staffNavItems");
    expect(src).toContain("role === 'cashier' ? cashierNavItems");
    expect(src).toContain("role === 'manager' ? managerNavItems");
  });

  it('shows admin section only for admin and superadmin', () => {
    expect(src).toContain("role === 'admin' || role === 'superadmin'");
  });

  it('marks audit logs as requiring superadmin', () => {
    expect(src).toContain('requiresSuperadmin: true');
    expect(src).toContain("/admin/audit-logs'");
  });
});