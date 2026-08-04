import { describe, it, expect } from 'vitest';
import { fileURLToPath } from 'node:url';
import { readFileSync } from 'node:fs';
import path from 'node:path';

const __filename = fileURLToPath(import.meta.url);
function getSource(): string {
  return readFileSync(path.join(path.dirname(__filename), '..', 'UsersPage.svelte'), 'utf-8');
}

describe('UsersPage.svelte source-structure guards', () => {
  const src = getSource();

  it('imports api services', () => {
    expect(src).toContain("getSubordinates");
    expect(src).not.toContain("apiFetch(");
  });

  it('uses $state for users, loading, pagination', () => {
    expect(src).toContain('let users = $state');
    expect(src).toContain('let loading = $state');
    expect(src).toContain('let total = $state(0)');
    expect(src).toContain('let searchQuery = $state');
  });

  it('has permission-based RBAC (canCreate, canEdit, canDelete)', () => {
    expect(src).toContain("let canCreate = $derived(rbac.can(Permissions.user.create))");
    expect(src).toContain("let canEdit = $derived(rbac.can(Permissions.user.update))");
    expect(src).toContain("let canDelete = $derived(rbac.can(Permissions.user.delete))");
  });

  it('has load, saveUser, confirmDelete functions', () => {
    expect(src).toContain('async function fetchUsers');
    expect(src).toContain('async function saveUser');
    expect(src).toContain('async function confirmDelete');
  });

  it('has inline editing (openEdit, cancelEdit)', () => {
    expect(src).toContain('function openEdit');
  });

  it('imports extracted modal and table components', () => {
    expect(src).toContain("import UserFormModal from './UserFormModal.svelte'");
    expect(src).toContain("import UserDeleteModal from './UserDeleteModal.svelte'");
    expect(src).toContain("import UserToolbar from './UserToolbar.svelte'");
    expect(src).toContain("import UserTable from './UserTable.svelte'");
  });

  it('renders Pagination component', () => {
    expect(src).toContain('<Pagination');
  });
});
