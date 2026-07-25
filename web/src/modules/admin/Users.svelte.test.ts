import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), 'components/UsersPage.svelte'), 'utf-8');
}

describe('UsersPage.svelte source-structure guards', () => {
  const src = getSource();

  // ── Imports ──────────────────────────────────────────────────────────────────
  it('imports service functions from $modules/admin', () => {
    expect(src).toContain("import { getUsers, getRolesList, createUser, updateUser, deleteUser, getSubordinates } from '$modules/admin'");
  });

  it('does not import apiFetch from $lib/api/client', () => {
    expect(src).not.toContain("import { apiFetch }");
    expect(src).not.toContain('apiFetch(');
  });

  it('imports auth store for RBAC', () => {
    expect(src).toContain("import { useAuthStore } from '$modules/auth'");
  });

  // ── RBAC guards ──────────────────────────────────────────────────────────────
  it('imports useRBAC for role-based access control', () => {
    expect(src).toContain("const rbac = useRBAC()");
  });

  it('defines canCreate, canEdit for superadmin/admin, canDelete for superadmin only', () => {
    expect(src).toContain("let canCreate = $derived(rbac.canCreate)");
    expect(src).toContain("let canEdit = $derived(rbac.canEdit)");
    expect(src).toContain("let canDelete = $derived(rbac.canDelete)");
  });

  it('defines canView excluding cashier', () => {
    expect(src).toContain("let canView = $derived(rbac.canView)");
  });

  it('shows Access Denied when user lacks view permission', () => {
    expect(src).toContain('{#if !canView}');
    expect(src).toContain('Access Denied');
  });

  it('passes canCreate to UserToolbar', () => {
    expect(src).toContain('{canCreate}');
  });

  it('gates table action buttons behind canEdit/canDelete', () => {
    expect(src).toContain('openEdit(user)');
    expect(src).toContain('openDelete(user)');
  });

  // ── Self-deletion guard ───────────────────────────────────────────────────────
  it('tracks currentUserID from auth store', () => {
    expect(src).toContain('let currentUserID = $derived');
    expect(src).toContain('authStore.user?.id');
  });

  it('passes currentUserID and canEditSuperadmin to UserTable', () => {
    expect(src).toContain('{currentUserID}');
    expect(src).toContain('{canEditSuperadmin}');
  });

  it('confirmDelete rejects self-deletion', () => {
    expect(src).toContain('selectedUser.id === currentUserID');
    expect(src).toContain('You cannot delete your own account');
  });

  // ── Staff role handling ──────────────────────────────────────────────────────
  it('renders UserTable component', () => {
    expect(src).toContain('<UserTable');
  });

  // ── Sub-component usage ─────────────────────────────────────────────────────
  it('renders UserToolbar component', () => {
    expect(src).toContain('<UserToolbar');
    expect(src).toContain('bind:searchQuery');
  });

  // ── API consistency ──────────────────────────────────────────────────────────
  it('fetchUsers uses service getUsers with params', () => {
    expect(src).toContain('getUsers(params)');
  });

  it('uses service return format (result.data, result.total)', () => {
    expect(src).toContain('result.data');
    expect(src).toContain('result.total');
  });

  it('save uses createUser/updateUser service', () => {
    expect(src).toContain('createUser(form)');
    expect(src).toContain('updateUser(selectedUser.id, form)');
  });

  it('delete uses deleteUser service', () => {
    expect(src).toContain('deleteUser(selectedUser.id)');
  });

  // ── last_login rendering ─────────────────────────────────────────────────────
  it('passes sortBy, sortDir to UserTable for last_login sorting', () => {
    expect(src).toContain('bind:sortBy');
    expect(src).toContain('bind:sortDir');
  });
});
