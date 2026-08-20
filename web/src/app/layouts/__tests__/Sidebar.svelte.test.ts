import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'Sidebar.svelte'), 'utf-8');
}

describe('Sidebar.svelte source-structure guards', () => {
  const src = getSource();

  it('imports router helpers from $app/router', () => {
    expect(src).toContain("import { goto, getPath } from '$app/router'");
  });

  it('imports auth module functions', () => {
    expect(src).toContain("import { logout, useAuthStore, updatePreferences } from '$modules/auth'");
  });

  it('imports shift store and Tooltip', () => {
    expect(src).toContain("import { useShiftStore } from '$modules/shifts'");
    expect(src).toContain("import { Tooltip } from '$shared/ui'");
  });

  it('imports settingsStore for dynamic branding', () => {
    expect(src).toContain("import { settingsStore } from '$shared/stores/settings.svelte'");
  });

  it('uses settingsStore for branding text', () => {
    expect(src).toContain('settingsStore.storeName');
    expect(src).toContain('settingsStore.storeJargon');
  });

  it('uses settingsStore.logoPath for conditional logo display', () => {
    expect(src).toContain('settingsStore.logoPath');
  });

  it('uses $bindable for currentPath prop', () => {
    expect(src).toContain('currentPath = $bindable(\'/\')');
  });

  it('has collapsed and expanded state variables', () => {
    expect(src).toContain('let collapsed = $state(false)');
    expect(src).toContain('let adminExpanded = $state(false)');
    expect(src).toContain('let masterDataExpanded = $state(false)');
  });

  it('uses $derived for role-based navigation', () => {
    expect(src).toContain('visibleNavItems');
    expect(src).toContain('visibleMasterDataSubItems');
    expect(src).toContain('showAdminSection');
  });

  it('defines nav items for different roles', () => {
    expect(src).toContain('cashierNavItems');
    expect(src).toContain('staffNavItems');
    expect(src).toContain('managerNavItems');
    expect(src).toContain('adminItems');
  });

  it('has settings item in admin navigation', () => {
    expect(src).toContain('/admin/settings');
    expect(src).toContain('labels.settings');
  });

  it('has aria-label on aside for accessibility', () => {
    expect(src).toContain('aria-label={labels.sidebar}');
  });

  it('has handleLogout function that calls logout()', () => {
    expect(src).toContain('async function handleLogout()');
    expect(src).toContain('await logout()');
  });

  it('disables logout when cashier has active shift', () => {
    expect(src).toContain('canLogout');
    expect(src).toContain('!shiftStore.activeShift');
  });

  it('has collapse toggle buttons', () => {
    expect(src).toContain('labels.collapseSidebar');
    expect(src).toContain('labels.expandSidebar');
  });

  it('renders user info from auth store and RBAC', () => {
    expect(src).toContain('authStore.user?.username');
    expect(src).toContain('rbac.userRole !== Roles.cashier');
    expect(src).toContain('rbac.roleDisplayName');
  });
});
