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
    expect(src).toContain("import { logout, useAuthStore } from '$modules/auth'");
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

  it('has aria-label on aside for accessibility', () => {
    expect(src).toContain('aria-label="Sidebar"');
  });

  it('has handleLogout function', () => {
    expect(src).toContain('async function handleLogout()');
    expect(src).toContain("goto('/login')");
  });

  it('has collapse toggle buttons', () => {
    expect(src).toContain('Collapse sidebar');
    expect(src).toContain('Expand sidebar');
  });

  it('renders user info from auth store', () => {
    expect(src).toContain('authStore.user?.username');
    expect(src).toContain('authStore.user?.role');
  });
});
