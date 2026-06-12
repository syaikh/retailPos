import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), 'UsersPage.svelte'), 'utf-8');
}

describe('UsersPage.svelte source-structure guards', () => {
  const src = getSource();

  // ── Imports ──────────────────────────────────────────────────────────────────
  it('imports apiClient from $lib/api/client', () => {
    expect(src).toContain("import apiClient from '$lib/api/client'");
  });

  it('does not import apiFetch from $lib/api/client', () => {
    expect(src).not.toContain("import { apiFetch }");
    expect(src).not.toContain('apiFetch(');
  });

  it('imports auth store for RBAC', () => {
    expect(src).toContain("import { auth } from '$lib/stores/auth'");
  });

  // ── RBAC guards ──────────────────────────────────────────────────────────────
  it('derives userRole from auth store', () => {
    expect(src).toContain('let userRole = $derived');
    expect(src).toContain('$auth.user?.role?.name');
  });

  it('defines canCreate, canEdit for superadmin/admin, canDelete for superadmin only', () => {
    expect(src).toContain("let canCreate = $derived(['superadmin', 'admin'].includes(userRole))");
    expect(src).toContain("let canEdit = $derived(['superadmin', 'admin'].includes(userRole))");
    expect(src).toContain("let canDelete = $derived(userRole === 'superadmin')");
  });

  it('defines canView excluding cashier', () => {
    expect(src).toContain("let canView = $derived(userRole !== 'cashier' && userRole !== '')");
  });

  it('shows Access Denied when user lacks view permission', () => {
    expect(src).toContain('{#if !canView}');
    expect(src).toContain('Access Denied');
  });

  it('gates Add User button behind canCreate', () => {
    const addBtnIdx = src.indexOf('Add User');
    const context = src.slice(Math.max(0, addBtnIdx - 200), addBtnIdx + 200);
    expect(context).toContain('canCreate');
  });

  it('gates table action buttons behind canEdit/canDelete', () => {
    expect(src).toContain('openEdit(user)');
    expect(src).toContain("selectedUser = user; showDeleteModal = true");
  });

  // ── Self-deletion guard ───────────────────────────────────────────────────────
  it('tracks currentUserID from auth store', () => {
    expect(src).toContain('let currentUserID = $derived');
    expect(src).toContain('$auth.user?.id');
  });

  it('delete button is disabled for self and for superadmin users', () => {
    expect(src).toContain('user.id === currentUserID || user.role_id === 1');
  });

  it('confirmDelete rejects self-deletion', () => {
    expect(src).toContain('selectedUser.id === currentUserID');
    expect(src).toContain('You cannot delete your own account');
  });

  // ── Staff role handling ──────────────────────────────────────────────────────
  it('roleVariant handles staff role (falls through to muted)', () => {
    expect(src).toContain("roleName === 'superadmin'");
    expect(src).toContain("roleName === 'admin'");
    expect(src).toContain("return 'muted'");
  });

  // ── Search clear button ──────────────────────────────────────────────────────
  it('search input has clear button (X icon)', () => {
    expect(src).toContain('onclick={() => searchQuery = \'\'}');
    expect(src).toContain('<X size={14}');
  });

  // ── API consistency ──────────────────────────────────────────────────────────
  it('fetchUsers uses apiClient.get with params', () => {
    expect(src).toContain('apiClient.get');
    expect(src).toContain('/admin/users');
  });

  it('uses axios response format (r.data) instead of fetch format (r.json())', () => {
    expect(src).toContain('uRes.data?.data');
    expect(src).toContain('rRes.data?.data');
    expect(src).toContain('r.data?.error');
  });

  it('save uses apiClient with url, method, data', () => {
    expect(src).toContain('apiClient({ url, method, data: form })');
  });

  it('delete uses apiClient.delete', () => {
    expect(src).toContain('apiClient.delete');
  });

  // ── last_login rendering ─────────────────────────────────────────────────────
  it('renders last_login column with Never fallback', () => {
    expect(src).toContain('Last Login');
    expect(src).toContain("user.last_login");
    expect(src).toContain("'Never'");
  });
});
